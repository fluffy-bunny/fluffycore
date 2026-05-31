package mongostore

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FilterStore performs CRUD operations on the cefilters MongoDB collection.
// It is safe to call from multiple goroutines.
type FilterStore struct {
	col *mongo.Collection
}

// NewFilterStore creates a FilterStore backed by the given collection.
func NewFilterStore(col *mongo.Collection) *FilterStore {
	return &FilterStore{col: col}
}

// Collection returns the underlying *mongo.Collection for callers that need
// direct access (e.g. to build a FilterCDC).
func (s *FilterStore) Collection() *mongo.Collection { return s.col }

// Upsert inserts or updates a filter by Name.
//   - On insert:  CreatedAt and UpdatedAt are set to now.
//   - On update:  UpdatedAt is refreshed; CreatedAt is preserved; DeletedAt is cleared.
//
// The Expr field is validated by the caller before Upsert — the store does not
// call cefilter.Parse itself so that the store package stays free of that dependency.
func (s *FilterStore) Upsert(ctx context.Context, doc FilterDocument) error {
	now := time.Now().UTC()
	_, err := s.col.UpdateOne(ctx,
		bson.M{"_id": doc.Name},
		bson.M{
			// Explicit field list avoids sending _id in $set, which MongoDB
			// rejects on existing documents even when the value is unchanged.
			"$set": bson.M{
				"hint":      doc.Hint,
				"metadata":  doc.Metadata,
				"expr":      doc.Expr,
				"updatedAt": now,
				"deletedAt": nil, // re-activate if previously soft-deleted
			},
			"$setOnInsert": bson.M{"createdAt": now},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// SoftDelete sets deletedAt on the named filter without removing the document.
// A change stream watcher will see this as an update and emit ChangeOpDelete.
// Returns nil if the document does not exist or is already soft-deleted.
func (s *FilterStore) SoftDelete(ctx context.Context, name string) error {
	now := time.Now().UTC()
	_, err := s.col.UpdateOne(ctx,
		bson.M{"_id": name, "deletedAt": nil},
		bson.M{"$set": bson.M{"deletedAt": now, "updatedAt": now}},
	)
	return err
}

// HardDelete permanently removes the filter document.
// The change stream will emit a delete event whose documentKey._id is the name.
// Returns nil if the document does not exist.
func (s *FilterStore) HardDelete(ctx context.Context, name string) error {
	_, err := s.col.DeleteOne(ctx, bson.M{"_id": name})
	return err
}

// Get returns the filter with the given name, or (nil, nil) if not found.
// The document is returned regardless of its DeletedAt status.
func (s *FilterStore) Get(ctx context.Context, name string) (*FilterDocument, error) {
	var doc FilterDocument
	err := s.col.FindOne(ctx, bson.M{"_id": name}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// ListActive returns all non-deleted filter documents, sorted by Name (_id).
func (s *FilterStore) ListActive(ctx context.Context) ([]FilterDocument, error) {
	cursor, err := s.col.Find(ctx,
		bson.M{"deletedAt": nil},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []FilterDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
