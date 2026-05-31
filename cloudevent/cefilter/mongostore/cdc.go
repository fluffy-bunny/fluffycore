package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FilterCDC watches the cefilters collection for changes and emits FilterChange
// events on a caller-supplied channel.
//
// Lifecycle on first run (no persisted resume token):
//  1. Pin the current oplog position (hello command) to avoid missing events
//     that arrive during the full scan.
//  2. Emit one ChangeOpUpsert per active document (the snapshot).
//  3. Emit ChangeOpReady — the caller's local state is now fully initialised.
//  4. Open a change stream starting at the pinned oplog position and tail
//     live changes indefinitely.
//
// Lifecycle on reconnect (resume token present):
//  1. Emit ChangeOpReady immediately — local state was preserved across the
//     connection drop; only the delta needs to be replayed.
//  2. Open a change stream resuming from the saved token and tail live changes.
//
// The resume token is persisted to the Tokens collection after each event so
// that a process restart never replays more than one event.
//
// Run blocks until ctx is cancelled or a fatal error occurs. Call it in a
// goroutine and restart it on transient errors (e.g. network blips).
type FilterCDC struct {
	// Filters is the cefilters collection to watch.
	Filters *mongo.Collection

	// Tokens is the collection used to persist the resume token between
	// process restarts. A single document per TokenID is maintained here.
	Tokens *mongo.Collection

	// TokenID is the _id of the resume token document in the Tokens collection.
	// Use a unique value per logical CDC process if you run multiple.
	// Defaults to "cefilters_cdc".
	TokenID string
}

// tokenDoc is the document stored in the Tokens collection.
type tokenDoc struct {
	ID          string   `bson:"_id"`
	ResumeToken bson.Raw `bson:"resumeToken"`
}

func (c *FilterCDC) tokenID() string {
	if c.TokenID != "" {
		return c.TokenID
	}
	return "cefilters_cdc"
}

// Run starts the CDC event loop. It blocks until ctx is cancelled.
// The caller must drain the changes channel to prevent blocking.
func (c *FilterCDC) Run(ctx context.Context, changes chan<- FilterChange) error {
	token := c.loadToken(ctx)

	csOpts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	if token == nil {
		// First run: pin the oplog position BEFORE scanning so that any writes
		// arriving during the scan are replayed by the change stream.
		startTime, err := c.currentOpTime(ctx)
		if err != nil {
			return fmt.Errorf("cefilter cdc: get optime: %w", err)
		}
		csOpts.SetStartAtOperationTime(startTime)

		store := NewFilterStore(c.Filters)
		docs, err := store.ListActive(ctx)
		if err != nil {
			return fmt.Errorf("cefilter cdc: full sync: %w", err)
		}
		for i := range docs {
			if err := chanSend(ctx, changes, FilterChange{
				Op:   ChangeOpUpsert,
				Name: docs[i].Name,
				Doc:  &docs[i],
			}); err != nil {
				return err
			}
		}
	} else {
		csOpts.SetResumeAfter(token)
	}

	// ChangeOpReady fires after snapshot (or immediately on resume).
	// Callers gate their event-routing loop on this signal.
	if err := chanSend(ctx, changes, FilterChange{Op: ChangeOpReady}); err != nil {
		return err
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "operationType", Value: bson.D{
				{Key: "$in", Value: bson.A{"insert", "update", "replace", "delete"}},
			}},
		}}},
	}

	cs, err := c.Filters.Watch(ctx, pipeline, csOpts)
	if err != nil {
		return fmt.Errorf("cefilter cdc: watch: %w", err)
	}
	defer cs.Close(ctx)

	for cs.Next(ctx) {
		change, err := decodeEvent(cs.Current)
		if err != nil {
			// Undecodable event: log in production, then skip.
			// Replacing this with your structured logger is recommended.
			_ = fmt.Errorf("cefilter cdc: decode event: %w", err)
			continue
		}
		if change == nil {
			continue // operation type not relevant
		}
		if err := chanSend(ctx, changes, *change); err != nil {
			return err
		}
		// Persist the token after every successful delivery so a restart
		// at worst replays one event (idempotent in the caller).
		c.saveToken(ctx, cs.ResumeToken())
	}

	if err := cs.Err(); err != nil {
		return fmt.Errorf("cefilter cdc: stream: %w", err)
	}
	return ctx.Err()
}

// decodeEvent converts a raw change stream document into a FilterChange.
// Returns (nil, nil) for operation types the CDC does not act on.
func decodeEvent(raw bson.Raw) (*FilterChange, error) {
	opType := raw.Lookup("operationType").StringValue()

	switch opType {
	case "insert", "update", "replace":
		fdVal := raw.Lookup("fullDocument")
		fdRaw, ok := fdVal.DocumentOK()
		if !ok {
			return nil, fmt.Errorf("fullDocument missing or not a document for op %q", opType)
		}
		var doc FilterDocument
		if err := bson.Unmarshal(fdRaw, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal fullDocument: %w", err)
		}
		if doc.DeletedAt != nil {
			// Soft delete — the update set deletedAt; treat as removal.
			return &FilterChange{Op: ChangeOpDelete, Name: doc.Name}, nil
		}
		return &FilterChange{Op: ChangeOpUpsert, Name: doc.Name, Doc: &doc}, nil

	case "delete":
		// Hard delete — documentKey._id is the filter Name because we use Name as _id.
		name := raw.Lookup("documentKey", "_id").StringValue()
		if name == "" {
			return nil, fmt.Errorf("documentKey._id empty in delete event")
		}
		return &FilterChange{Op: ChangeOpDelete, Name: name}, nil
	}

	return nil, nil
}

// currentOpTime returns the current MongoDB cluster operation time by running
// the hello command. Requires a replica set or sharded cluster; returns an
// error on standalone instances (which also don't support change streams).
func (c *FilterCDC) currentOpTime(ctx context.Context) (*primitive.Timestamp, error) {
	var result bson.M
	if err := c.Filters.Database().RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&result); err != nil {
		return nil, err
	}
	t, ok := result["operationTime"].(primitive.Timestamp)
	if !ok {
		return nil, fmt.Errorf(
			"operationTime absent from hello response — change streams require a replica set or sharded cluster",
		)
	}
	return &t, nil
}

func (c *FilterCDC) loadToken(ctx context.Context) bson.Raw {
	var doc tokenDoc
	if err := c.Tokens.FindOne(ctx, bson.M{"_id": c.tokenID()}).Decode(&doc); err != nil {
		return nil
	}
	return doc.ResumeToken
}

func (c *FilterCDC) saveToken(ctx context.Context, token bson.Raw) {
	_, _ = c.Tokens.UpdateOne(ctx,
		bson.M{"_id": c.tokenID()},
		bson.M{"$set": tokenDoc{ID: c.tokenID(), ResumeToken: token}},
		options.Update().SetUpsert(true),
	)
}

// chanSend writes a change to ch, returning ctx.Err() if the context is cancelled.
func chanSend(ctx context.Context, ch chan<- FilterChange, change FilterChange) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- change:
		return nil
	}
}
