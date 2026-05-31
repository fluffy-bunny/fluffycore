# cefilter — development notes

Last updated: 2026-05-20  
Branch: `nats_stream_proto`

## Package locations

| Path | Contents |
|---|---|
| `cloudevent/contracts/cefilter.go` | `FilterEntry`, `IFilterSet` interface |
| `cloudevent/cefilter/filter.go` | `Attrs`, `Predicate`, leaf predicates (`eq`/`ne`/`like`/`in`/`exists`), `AND`/`OR`/`NOT` |
| `cloudevent/cefilter/lexer.go` | tokenizer |
| `cloudevent/cefilter/parser.go` | recursive-descent parser |
| `cloudevent/cefilter/accessor.go` | `CloudEventAccessor`, `FromCloudEvent`, `FilterSet` |
| `cloudevent/cefilter/index.go` | `IndexedFilterSet` (RWMutex, inverted type index) |
| `cloudevent/cefilter/sdk.go` | `FromSDKEvent(cloudevents.Event)` — direct SDK shortcut |
| `cloudevent/cefilter/README.md` | user-facing documentation |

## Test files

| File | What it covers |
|---|---|
| `filter_test.go` | `Parse` + `FilterSet` (mockEvent, all operators) |
| `index_test.go` | `IndexedFilterSet` tests + benchmarks (1000-filter scale) |
| `sdk_test.go` | `FromSDKEvent` integration with real `cloudevents.Event` |
| `ce_routing_test.go` | CE routing, fan-out, webhook pattern, circuit breaker, `Remove` |

Run all: `go test ./cloudevent/cefilter/... -v -run . -count=1`  
Total: **32 tests**, all passing.

---

## API surface

### `contracts.FilterEntry`

```go
type FilterEntry struct {
    Name     string            // unique filter ID — used by Remove()
    Hint     string            // opaque routing target (URL, NATS subject, etc.)
    Metadata map[string]string // optional extra annotations
}
```

### `contracts.IFilterSet`

Satisfied by both `FilterSet` and `IndexedFilterSet`:

```go
type IFilterSet interface {
    Add(entry FilterEntry, expr string) error
    Remove(name string)
    Match(attrs map[string]string) []FilterEntry
    MatchEndpoints(attrs map[string]string) []string // unique Hints, deduped
    MatchAny(attrs map[string]string) bool
}
```

### `FilterSet` — not thread-safe

- Fields: `MaxMatches int`, `OnLimitExceeded func(Attrs, int)`
- Slice scan, O(N) per call
- Good for: tests, single-goroutine, build-once-never-modify scenarios

### `IndexedFilterSet` — thread-safe (`sync.RWMutex`)

- Fields: `MaxMatches int`, `OnLimitExceeded func(Attrs, int)`
- Internal buckets: `exactIndex map[string][]*indexedEntry`, `globEntries`, `wildcardEntries`
- `Add`: O(1), write lock only during map insert
- `MatchAny`: RLock + 1 hash lookup ≈ 25 ns for unknown type
- `Match` / `MatchEndpoints`: RLock + hash lookup over matching bucket
- `Remove`: write lock, walks all buckets, O(N filters)
- `Stats() IndexStats`: capacity planning helper

---

## Circuit breaker

```go
var gate cefilter.IndexedFilterSet
gate.MaxMatches = 10
gate.OnLimitExceeded = func(a cefilter.Attrs, count int) {
    slog.Error("fan-out limit exceeded", "type", a["type"], "matched", count)
}
```

- `MaxMatches = 0` (default) → unlimited
- `MatchAny` is unaffected — always short-circuits on first match
- Set fields before concurrent use begins

---

## Webhook fan-out pattern

```go
// Register (call from any goroutine with IndexedFilterSet)
gate.Add(contracts.FilterEntry{Name: "f1", Hint: "https://svc/hook"}, `type = "com.shop.order.placed"`)
gate.Add(contracts.FilterEntry{Name: "f2", Hint: "https://svc/hook"}, `type LIKE "com.shop.order.*"`)

// Phase 1: ingress gate — drop events nobody wants at O(1) cost
attrs := cefilter.FromSDKEvent(event)
if !gate.MatchAny(attrs) {
    return // dropped
}

// Phase 2: routing — one URL per endpoint, even if multiple rules matched
for _, url := range gate.MatchEndpoints(attrs) {
    go postWebhook(url, event)
}

// Deregister when endpoint is deleted
gate.Remove("f1")
gate.Remove("f2")
```

`MatchEndpoints` deduplicates by `Hint`, so f1 + f2 above still yield only one POST per event.

---

## Expression syntax reference

```
type = "com.example.order"           // exact equality
subject != "internal"                // not equal (also matches if attribute absent)
source LIKE "checkout/*"             // glob: * = any chars, ? = one char
region IN ("us-east", "eu-west")     // membership, O(1) hash set
EXISTS priority                      // attribute is present
(A OR B) AND C AND NOT EXISTS debug  // combinators; precedence: NOT > AND > OR
```

Expressions are compiled once via `Parse(expr)` or `FilterSet/IndexedFilterSet.Add(...)`. Per-event `Match` is allocation-free.

---

## Performance notes (index_test.go benchmarks, 1000 filters)

| Scenario | Implementation | ns/op |
|---|---|---|
| MatchAny, no-match (unknown type) | `IndexedFilterSet` | ~10 |
| MatchAny, no-match | `FilterSet` | ~6692 |
| MatchAny, match | `IndexedFilterSet` | proportional to bucket size |

The inverted index makes `IndexedFilterSet` ~650× faster than `FilterSet` for the common case of events whose type no filter covers.

---

## Design decisions

- **No SDK dependency in core.** `CloudEventAccessor` keeps `cefilter` import-free from `cloudevents/sdk-go`. `FromSDKEvent` lives in `sdk.go` as an opt-in.
- **`cloudevents.Event` satisfies `CloudEventAccessor` natively** — same method signatures, no adapter struct needed.
- **RWMutex over atomic snapshot.** Copy-on-write with `atomic.Pointer` caused O(N) hangs during Add on large filter sets. RWMutex gives microsecond write locks and zero-alloc reads.
- **Inverted index keyed on `type`.** Most CE filters constrain `type`; indexing on it gives the best hit rate for the hash lookup.
- **`LIKE` compiled to `*regexp.Regexp` at parse time**, never per event call.
- **`IN` uses a `map[string]struct{}` set** at parse time — O(1) membership, never O(M·N).

---

## mongostore — persisting filter rules in MongoDB

`cloudevent/cefilter/mongostore` is the durable backing store for cefilter registrations.  
Its role in the overall system is:

```
Admin API / CLI
     │
     ▼
FilterStore (CRUD)  ──writes──▶  MongoDB "cefilters" collection
                                         │
                                   Change stream
                                         │
                                         ▼
                                    FilterCDC
                                         │
                                         ▼
                              IndexedFilterSet (in-memory)
                                         │
                                         ▼
                              CloudEvent ingress handler
```

Every running process keeps its own in-memory `IndexedFilterSet`. `FilterCDC` seeds it on startup and keeps it current as rules change, so writes to the DB are automatically propagated to all instances without polling.

### Files

| File | Contents |
|---|---|
| `doc.go` | `FilterDocument`, `ChangeOp` constants, `FilterChange` event struct |
| `store.go` | `FilterStore` — CRUD on the `cefilters` collection |
| `cdc.go` | `FilterCDC` — change-data-capture watcher (snapshot + live tail) |
| `migrations/000001_create_cefilters.up.json` | creates collection + compound indexes |
| `migrations/000001_create_cefilters.down.json` | drops collection |

### `FilterDocument` (stored in `cefilters`, `_id = Name`)

```go
type FilterDocument struct {
    Name      string            // _id — unique filter identifier
    Hint      string            // routing target (URL, NATS subject, …)
    Metadata  map[string]string // optional caller annotations
    Expr      string            // cefilter expression, e.g. `type = "com.example.order"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt *time.Time        // nil = active; non-nil = soft-deleted
}
```

Using `Name` as `_id` means uniqueness is enforced by the primary key and hard-delete events carry the name in `documentKey._id` — no secondary lookup required.

### `FilterStore` — CRUD

```go
store := mongostore.NewFilterStore(col)

// Insert or update (clears DeletedAt if previously soft-deleted)
store.Upsert(ctx, mongostore.FilterDocument{
    Name: "order-hook",
    Hint: "https://svc/webhook",
    Expr: `type = "com.shop.order.placed"`,
})

// Soft delete — sets deletedAt; CDC emits ChangeOpDelete
store.SoftDelete(ctx, "order-hook")

// Hard delete — removes document; CDC emits ChangeOpDelete
store.HardDelete(ctx, "order-hook")

// Point read (returns nil, nil if not found)
doc, err := store.Get(ctx, "order-hook")

// All active (deletedAt = nil), sorted by Name
docs, err := store.ListActive(ctx)
```

Expression validation (`cefilter.Parse`) is the caller's responsibility — `FilterStore` does not import the filter parser so the two packages stay decoupled.

### `FilterCDC` — snapshot + live tail

`FilterCDC` implements the standard **snapshot-then-tail** CDC pattern using MongoDB change streams.

**First run (no persisted resume token):**
1. Pin the current oplog position (`hello` command) — writes that arrive during the scan won't be missed.
2. Stream all active documents as `ChangeOpUpsert` events (the snapshot).
3. Emit `ChangeOpReady` — the caller's `IndexedFilterSet` is fully initialised.
4. Open a change stream starting at the pinned position and tail indefinitely.

**Reconnect (resume token present):**
1. Emit `ChangeOpReady` immediately — local state was preserved across the connection drop.
2. Open a change stream resuming from the saved token and tail live changes.

The resume token is persisted to a separate Tokens collection after every delivered event, so a process restart at worst replays one event (callers should make their `Add`/`Remove` idempotent — `IndexedFilterSet` already is).

```go
cdc := &mongostore.FilterCDC{
    Filters: db.Collection("cefilters"),
    Tokens:  db.Collection("cdc_tokens"),
    TokenID: "my-service-cefilters", // unique per logical CDC process
}

changes := make(chan mongostore.FilterChange, 64)

go func() {
    for {
        if err := cdc.Run(ctx, changes); err != nil && ctx.Err() != nil {
            return // context cancelled — normal shutdown
        }
        // Transient error (network blip). Backoff then restart.
        time.Sleep(5 * time.Second)
    }
}()
```

### Wiring CDC → IndexedFilterSet

```go
var gate cefilter.IndexedFilterSet

// Block serving until snapshot is complete.
for change := range changes {
    switch change.Op {
    case mongostore.ChangeOpUpsert:
        entry := contracts.FilterEntry{
            Name:     change.Doc.Name,
            Hint:     change.Doc.Hint,
            Metadata: change.Doc.Metadata,
        }
        gate.Add(entry, change.Doc.Expr) // parse error = corrupt DB row; log and skip
    case mongostore.ChangeOpDelete:
        gate.Remove(change.Name)
    case mongostore.ChangeOpReady:
        // Gate is now fully populated — start serving CloudEvents.
        go serveEvents(&gate, changes) // hand off the channel to live-update loop
        return
    }
}
```

### Indexes (migration 000001)

| Index | Purpose |
|---|---|
| `{ deletedAt: 1, updatedAt: -1 }` | `ListActive` filter + ordering during cold-start snapshot |
| `{ updatedAt: -1 }` | Full-scan ordering when CDC needs to replay unindexed deletes |

### CDC change operations

| `ChangeOp` | When emitted | `Doc` populated? |
|---|---|---|
| `ChangeOpUpsert` | Insert, update, replace (and soft-un-delete) | Yes |
| `ChangeOpDelete` | Soft delete (deletedAt set) or hard delete | No |
| `ChangeOpReady` | After snapshot phase completes (or immediately on resume) | No |

### Design decisions

- **Soft delete vs hard delete.** Soft delete (`DeletedAt` field) gives an audit trail and lets CDC emit a `ChangeOpDelete` via an update event (which carries `fullDocument`). Hard delete also works — the change stream's `documentKey._id` provides the name without a secondary lookup.
- **`Name` as `_id`.** Avoids a unique index on a separate `name` field and makes hard-delete CDC events self-contained.
- **Resume token persistence.** Stored in a dedicated Tokens collection (not the filter document itself) so it can survive collection drops/migrations.
- **`Run` is blocking + restartable.** Callers wrap it in a retry loop with backoff; `FilterCDC` itself has no internal retry to keep the failure handling strategy in the caller's hands.
