package mongostore

import "time"

// FilterDocument is the MongoDB representation of a cefilter registration.
//
// Name is used as the document _id. This means:
//   - Uniqueness is enforced by the primary key — no extra unique index needed.
//   - Hard-delete change stream events carry the name in documentKey._id, so
//     the CDC can emit the correct ChangeOpDelete without a secondary lookup.
type FilterDocument struct {
	// Name is the unique filter identifier and the MongoDB _id.
	Name string `bson:"_id"`
	// Hint is the opaque routing target (webhook URL, NATS subject, etc.).
	Hint string `bson:"hint"`
	// Metadata holds optional caller-defined annotations.
	Metadata map[string]string `bson:"metadata,omitempty"`
	// Expr is the cefilter expression string, e.g. `type = "com.example.order"`.
	Expr string `bson:"expr"`

	CreatedAt time.Time  `bson:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"` // nil = active
}

// ChangeOp describes the operation carried by a FilterChange event.
type ChangeOp string

const (
	// ChangeOpUpsert signals that a filter was added or its definition changed.
	// FilterChange.Doc is populated.
	ChangeOpUpsert ChangeOp = "upsert"

	// ChangeOpDelete signals that a filter was removed (soft or hard delete).
	// FilterChange.Doc is nil.
	ChangeOpDelete ChangeOp = "delete"

	// ChangeOpReady signals that the initial snapshot phase is complete and
	// the CDC is now tailing live changes.
	// On first run this arrives after all active documents have been emitted as
	// ChangeOpUpsert events.
	// On reconnect this arrives immediately — the local state is already current.
	// Callers should not serve traffic against their local IndexedFilterSet until
	// they receive this event.
	ChangeOpReady ChangeOp = "ready"
)

// FilterChange is emitted on the channel passed to FilterCDC.Run.
type FilterChange struct {
	// Op is the type of change.
	Op ChangeOp
	// Name is the filter name (_id). Always set.
	Name string
	// Doc is the current document state. Set only when Op == ChangeOpUpsert.
	Doc *FilterDocument
}
