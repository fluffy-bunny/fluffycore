package mongostore

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FilterCDC watches the cefilters collection for changes and emits FilterChange
// events on a caller-supplied channel.
//
// Use ONE FilterCDC per local filter set (e.g. one IndexedFilterSet), for that set's
// whole lifetime, and call Run again on the same FilterCDC after a transient error.
// A FilterCDC remembers its resume position and which filters it has reported live, so
// that it can keep the caller's set exact across reconnects.
//
// First Run (and any Run after the resume position was lost):
//  1. Pin the current oplog position (hello command) so writes that arrive during the
//     full scan are replayed by the change stream.
//  2. Emit one ChangeOpUpsert per active document (the snapshot), then one
//     ChangeOpDelete for every filter reported live earlier that is no longer active
//     (deleted while the stream could not be followed).
//  3. Emit ChangeOpReady.
//  4. Tail the change stream from the pinned position.
//
// Later Runs (reconnect after a transient error): emit ChangeOpReady immediately and
// resume the change stream from the last position seen, which replays every change
// made in between -- including deletes.
//
// If the resume position has fallen off the oplog (or the server rejects it), Run
// re-snapshots as in the first Run. If the stream is invalidated (collection dropped or
// renamed), Run returns an error and the next Run re-snapshots.
//
// The resume position is deliberately NOT persisted: it only means something relative to
// the in-memory filter set built from this FilterCDC's events. A new process has an empty
// set and must snapshot; resuming a previous process's position would leave it empty.
//
// Run blocks until ctx is cancelled or an error occurs. Call it in a goroutine and call it
// again (on the same FilterCDC) after transient errors. Run must not be called
// concurrently on one FilterCDC.
type FilterCDC struct {
	// Filters is the cefilters collection to watch.
	Filters *mongo.Collection

	// Deprecated: ignored. The resume token is kept in memory; see the type doc.
	Tokens *mongo.Collection

	// Deprecated: ignored, as Tokens.
	TokenID string

	running atomic.Bool
	// Resume position: the last change stream token, or -- before the stream has produced
	// one -- the operation time the last snapshot was pinned at. Both nil = snapshot.
	resumeToken bson.Raw
	resumeTime  *bson.Timestamp
	live        map[string]struct{} // names reported as ChangeOpUpsert and not since deleted
}

// Server error codes meaning the change stream cannot be resumed from the given position.
var resumeLostCodes = []int{
	136, // CappedPositionLost (pre-4.4 servers)
	260, // InvalidResumeToken
	280, // ChangeStreamFatalError
	286, // ChangeStreamHistoryLost
}

// Run starts the CDC event loop. It blocks until ctx is cancelled or an error occurs.
// The caller must drain the changes channel to prevent blocking.
func (c *FilterCDC) Run(ctx context.Context, changes chan<- FilterChange) error {
	if !c.running.CompareAndSwap(false, true) {
		return errors.New("cefilter cdc: Run is already running on this FilterCDC")
	}
	defer c.running.Store(false)

	err := c.runOnce(ctx, changes)
	if isResumeLost(err) && ctx.Err() == nil {
		// Changes since the saved (or pinned) position, including deletes, can no longer be
		// replayed. Rebuild from a snapshot; it reconciles deletes against what was reported
		// live. Only once per Run, so a snapshot slower than the oplog window cannot loop.
		c.forgetPosition()
		err = c.runOnce(ctx, changes)
	}
	return err
}

func (c *FilterCDC) runOnce(ctx context.Context, changes chan<- FilterChange) error {
	csOpts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	switch {
	case c.resumeToken != nil:
		csOpts.SetResumeAfter(c.resumeToken)
	case c.resumeTime != nil:
		// Snapshot done, but the stream never produced a token (its first batch was not
		// empty, or it never opened). The snapshot is consistent as of resumeTime.
		csOpts.SetStartAtOperationTime(c.resumeTime)
	default:
		// Pin the oplog position BEFORE scanning so that any writes arriving during the
		// scan are replayed by the change stream.
		startTime, err := c.currentOpTime(ctx)
		if err != nil {
			return fmt.Errorf("cefilter cdc: get optime: %w", err)
		}
		csOpts.SetStartAtOperationTime(startTime)
		if err := c.snapshot(ctx, changes); err != nil {
			return err
		}
		c.resumeTime = startTime
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
	defer cs.Close(context.WithoutCancel(ctx))
	// The initial (post-batch) token lets a reconnect before the first event resume
	// instead of re-snapshotting.
	c.rememberPosition(cs)

	// ChangeOpReady fires once the stream is open: after the snapshot, or immediately on
	// resume. Callers gate their event-routing loop on this signal.
	if err := chanSend(ctx, changes, FilterChange{Op: ChangeOpReady}); err != nil {
		return err
	}

	for cs.Next(ctx) {
		change, err := decodeEvent(cs.Current)
		if err != nil {
			// Undecodable event: log in production, then skip.
			// Replacing this with your structured logger is recommended.
			_ = fmt.Errorf("cefilter cdc: decode event: %w", err)
		} else if change != nil {
			if err := chanSend(ctx, changes, *change); err != nil {
				return err
			}
			c.track(*change)
		}
		c.rememberPosition(cs)
	}

	if err := cs.Err(); err != nil {
		return fmt.Errorf("cefilter cdc: stream: %w", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// The stream ended without an error: it was invalidated (collection dropped or
	// renamed). It cannot be resumed; the next Run re-snapshots.
	c.forgetPosition()
	return errors.New("cefilter cdc: change stream invalidated (collection dropped or renamed)")
}

// snapshot emits every active filter as ChangeOpUpsert, then ChangeOpDelete for every
// filter reported live earlier that is no longer active.
func (c *FilterCDC) snapshot(ctx context.Context, changes chan<- FilterChange) error {
	docs, err := NewFilterStore(c.Filters).ListActive(ctx)
	if err != nil {
		return fmt.Errorf("cefilter cdc: full sync: %w", err)
	}
	active := make(map[string]struct{}, len(docs))
	for i := range docs {
		active[docs[i].Name] = struct{}{}
		change := FilterChange{Op: ChangeOpUpsert, Name: docs[i].Name, Doc: &docs[i]}
		if err := chanSend(ctx, changes, change); err != nil {
			return err
		}
		c.track(change)
	}
	for _, name := range staleNames(c.live, active) {
		change := FilterChange{Op: ChangeOpDelete, Name: name}
		if err := chanSend(ctx, changes, change); err != nil {
			return err
		}
		c.track(change)
	}
	return nil
}

// staleNames returns the names in live that are not in active.
func staleNames(live, active map[string]struct{}) []string {
	var stale []string
	for name := range live {
		if _, ok := active[name]; !ok {
			stale = append(stale, name)
		}
	}
	return stale
}

// track records what the caller has been told is live.
func (c *FilterCDC) track(change FilterChange) {
	switch change.Op {
	case ChangeOpUpsert:
		if c.live == nil {
			c.live = make(map[string]struct{})
		}
		c.live[change.Name] = struct{}{}
	case ChangeOpDelete:
		delete(c.live, change.Name)
	}
}

func (c *FilterCDC) forgetPosition() {
	c.resumeToken = nil
	c.resumeTime = nil
}

func (c *FilterCDC) rememberPosition(cs *mongo.ChangeStream) {
	if token := cs.ResumeToken(); token != nil {
		c.resumeToken = token
	}
}

// isResumeLost reports whether err means the change stream cannot be resumed from the
// saved position.
func isResumeLost(err error) bool {
	var se mongo.ServerError
	if !errors.As(err, &se) {
		return false
	}
	for _, code := range resumeLostCodes {
		if se.HasErrorCode(code) {
			return true
		}
	}
	return false
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
func (c *FilterCDC) currentOpTime(ctx context.Context) (*bson.Timestamp, error) {
	var result bson.M
	if err := c.Filters.Database().RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&result); err != nil {
		return nil, err
	}
	t, ok := result["operationTime"].(bson.Timestamp)
	if !ok {
		return nil, fmt.Errorf(
			"operationTime absent from hello response — change streams require a replica set or sharded cluster",
		)
	}
	return &t, nil
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
