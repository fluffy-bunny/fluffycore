# cefilter — JetStream KV Sync Design

This document covers the design for using MongoDB as the system of record (SOR) for filter registrations, with NATS JetStream KV as the fast operational sync layer that fans changes out to all running pods in real time.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Write API  (your service — gRPC, REST, whatever)               │
│  Add / Update / Delete filter registrations                     │
└───────────────────────┬─────────────────────────────────────────┘
                        │ write
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│  MongoDB  —  "filters" collection                               │
│  System of Record. Durable, queryable, auditable.               │
│  Survives NATS outages.                                         │
└───────────────────────┬─────────────────────────────────────────┘
                        │ MongoDB Change Stream (CDC process)
                        │ resume token persisted → survives restarts
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│  NATS JetStream KV  —  bucket "cefilters"                       │
│  Operational cache. Current state of every active filter.       │
│  O(filter count) snapshot for new pods. Auto-resume on reconnect│
└──────┬────────────────┬────────────────┬────────────────────────┘
       │ WatchAll        │ WatchAll        │ WatchAll
       ▼                 ▼                 ▼
   Pod A             Pod B             Pod C
IndexedFilterSet  IndexedFilterSet  IndexedFilterSet
  (in-process)      (in-process)      (in-process)
  ~25 ns/lookup     ~25 ns/lookup     ~25 ns/lookup
```

**Why two layers?**

- MongoDB is the SOR: survives NATS loss, supports complex queries, audit history, RBAC.
- JetStream KV is the operational cache: O(filter count) snapshot (not O(event history)), auto-resume on reconnect, and `WatchAll` delivers current state + live tail in one call.
- Pods never touch MongoDB on the hot path. All runtime lookups are in-process.

---

## MongoDB Schema

```go
// FilterDocument is the MongoDB representation of one registered filter.
type FilterDocument struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    Name      string             `bson:"name"`        // unique — KV key
    Hint      string             `bson:"hint"`        // routing target
    Metadata  map[string]string  `bson:"metadata,omitempty"`
    Expr      string             `bson:"expr"`        // cefilter expression
    CreatedAt time.Time          `bson:"createdAt"`
    UpdatedAt time.Time          `bson:"updatedAt"`
    DeletedAt *time.Time         `bson:"deletedAt,omitempty"` // nil = active
}
```

Indexes:
```js
db.filters.createIndex({ name: 1 }, { unique: true })
db.filters.createIndex({ deletedAt: 1 })  // for active-filter scans
```

Use **soft deletes** (`deletedAt` timestamp). This keeps the change stream simple — every change is an `update` or `insert`, never a hard `delete`. Hard deletes are supported too but require handling `operationType: "delete"` separately in the CDC.

---

## JetStream KV Design

### Bucket

```go
kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
    Bucket:      "cefilters",
    Description: "Active cefilter registrations — derived from MongoDB filters collection",
    History:     1,         // only latest value per key; we don't need revision history
    Storage:     nats.FileStorage,
    Replicas:    3,         // match your NATS cluster size
    MaxValueSize: 64 * 1024, // 64 KB per filter — generous upper bound
})
```

`History: 1` is important — it keeps the bucket size proportional to the number of active filters, not the number of edits ever made. On pod startup `WatchAll` delivers only current values.

### Key / Value

```
Key   = FilterDocument.Name   (e.g. "order-placed-org-123")
Value = JSON-encoded FilterKVValue
```

```go
// FilterKVValue is the value stored in the KV bucket.
type FilterKVValue struct {
    Hint     string            `json:"hint"`
    Metadata map[string]string `json:"metadata,omitempty"`
    Expr     string            `json:"expr"`
    // Version lets pods detect a no-op Put (same expr/hint, same version).
    // Use UpdatedAt unix nanoseconds from MongoDB.
    Version  int64             `json:"version"`
}
```

---

## Write Path

All writes go through MongoDB first. The KV is never written by the API directly.

```
1. API receives AddFilter / UpdateFilter / DeleteFilter request
2. Validate the cefilter expression (call cefilter.Parse — fail fast)
3. Upsert MongoDB (name unique index prevents duplicates)
4. Return success to caller
                ↓
           (async, sub-second)
5. CDC process detects change stream event
6. CDC publishes kv.Put / kv.Delete
7. All pod watchers receive the update
8. Each pod calls gate.Remove(name) then gate.Add(entry, expr)
```

**Why validate the expression at the API layer?**
The CDC process does a best-effort publish to KV. If it encounters an invalid expression in MongoDB (e.g., imported via bulk load), it should log and skip — not crash. Validate at write time so MongoDB only ever holds valid expressions.

---

## CDC Process

The CDC process is a single goroutine (or single-replica deployment) that watches the MongoDB change stream and publishes to the KV bucket. It persists its resume token durably so it survives restarts without replaying history from the beginning.

```go
type FilterCDC struct {
    filters     *mongo.Collection
    tokens      *mongo.Collection // separate collection for resume token storage
    kv          nats.KeyValue
}

// tokenDoc is stored in a "cdc_tokens" collection.
type tokenDoc struct {
    ID          string   `bson:"_id"`  // e.g. "cefilters_cdc"
    ResumeToken bson.Raw `bson:"resumeToken"`
}
```

### Startup Sequence

```go
func (c *FilterCDC) Run(ctx context.Context) error {
    // 1. Load persisted resume token (nil on first ever run)
    token := c.loadResumeToken(ctx)

    if token == nil {
        // First run or token was purged — do a full sync from MongoDB to KV.
        // This makes the KV consistent with MongoDB before we start streaming.
        if err := c.fullSync(ctx); err != nil {
            return err
        }
    }

    // 2. Open change stream, resuming after the saved token if present.
    //    If token is nil, we start from "now" — fullSync already covered history.
    pipeline := mongo.Pipeline{
        // Only watch the fields we care about
        {{Key: "$match", Value: bson.D{
            {Key: "operationType", Value: bson.D{
                {Key: "$in", Value: bson.A{"insert", "update", "replace", "delete"}},
            }},
        }}},
    }
    opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
    if token != nil {
        opts.SetResumeAfter(token)
    }

    cs, err := c.filters.Watch(ctx, pipeline, opts)
    if err != nil {
        return err
    }
    defer cs.Close(ctx)

    for cs.Next(ctx) {
        if err := c.handleEvent(ctx, cs.Current); err != nil {
            return err // caller retries Run()
        }
        // Persist resume token after each successful publish.
        c.saveResumeToken(ctx, cs.ResumeToken())
    }
    return cs.Err()
}
```

### Full Sync

```go
func (c *FilterCDC) fullSync(ctx context.Context) error {
    // Only active filters go into the KV.
    cursor, err := c.filters.Find(ctx,
        bson.M{"deletedAt": nil},
        options.Find().SetBatchSize(500),
    )
    if err != nil {
        return err
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var doc FilterDocument
        if err := cursor.Decode(&doc); err != nil {
            return err
        }
        if err := c.putKV(doc); err != nil {
            return err
        }
    }

    // Any keys in KV that are no longer in MongoDB (deleted while CDC was down)
    // need to be pruned. Get current KV keys and diff against MongoDB.
    return c.pruneStaleKeys(ctx)
}
```

`pruneStaleKeys` queries `kv.Keys()` and for any key not present (or soft-deleted) in MongoDB it calls `kv.Delete(key)`. This handles the case where a filter was deleted while the CDC was offline.

### Handling Change Events

```go
func (c *FilterCDC) handleEvent(ctx context.Context, raw bson.Raw) error {
    opType := raw.Lookup("operationType").StringValue()

    switch opType {
    case "insert", "update", "replace":
        // Use fullDocument (UpdateLookup ensures it's always present)
        var doc FilterDocument
        if err := bson.Unmarshal(raw.Lookup("fullDocument").Value, &doc); err != nil {
            return err
        }
        if doc.DeletedAt != nil {
            // Soft delete — remove from KV
            return c.kv.Delete(doc.Name)
        }
        return c.putKV(doc)

    case "delete":
        // Hard delete — extract name from documentKey
        name := raw.Lookup("documentKey", "name").StringValue()
        return c.kv.Delete(name)
    }
    return nil
}

func (c *FilterCDC) putKV(doc FilterDocument) error {
    val := FilterKVValue{
        Hint:     doc.Hint,
        Metadata: doc.Metadata,
        Expr:     doc.Expr,
        Version:  doc.UpdatedAt.UnixNano(),
    }
    b, err := json.Marshal(val)
    if err != nil {
        return err
    }
    _, err = c.kv.Put(doc.Name, b)
    return err
}
```

---

## Pod Watcher

Each pod runs one watcher goroutine that keeps the local `IndexedFilterSet` in sync. The pod must not serve event traffic until the initial snapshot is complete.

```go
type FilterWatcher struct {
    kv   nats.KeyValue
    gate *cefilter.IndexedFilterSet
    // Ready is closed when the initial KV snapshot has been fully applied.
    // Block ingress processing on this channel before routing events.
    Ready chan struct{}
}

func (w *FilterWatcher) Run(ctx context.Context) error {
    watcher, err := w.kv.WatchAll(nats.Context(ctx))
    if err != nil {
        return err
    }
    defer watcher.Stop()

    for entry := range watcher.Updates() {
        if entry == nil {
            // nil sentinel = initial snapshot complete; live tail begins now.
            close(w.Ready)
            continue
        }

        switch entry.Operation() {
        case nats.KeyValuePut:
            var val FilterKVValue
            if err := json.Unmarshal(entry.Value(), &val); err != nil {
                slog.Error("cefilter: bad KV value", "key", entry.Key(), "err", err)
                continue
            }
            // Always Remove first — handles the update case where the
            // expression or hint changed. Add is idempotent by Name only
            // if the predicate is the same; Remove+Add is always safe.
            w.gate.Remove(entry.Key())
            if addErr := w.gate.Add(contracts.FilterEntry{
                Name:     entry.Key(),
                Hint:     val.Hint,
                Metadata: val.Metadata,
            }, val.Expr); addErr != nil {
                // Invalid expression in KV — skip and alert.
                // This should never happen if the API validates at write time.
                slog.Error("cefilter: invalid expr in KV", "key", entry.Key(), "err", addErr)
            }

        case nats.KeyValueDelete, nats.KeyValuePurge:
            w.gate.Remove(entry.Key())
        }
    }
    return ctx.Err()
}
```

### Blocking ingress until ready

```go
watcher := &FilterWatcher{kv: kv, gate: &gate, Ready: make(chan struct{})}
go watcher.Run(ctx)

// Wait for snapshot before starting the event loop.
// Add a timeout so a NATS outage doesn't silently drop all events.
select {
case <-watcher.Ready:
    slog.Info("cefilter: filter set ready", "count", gate.Stats().Total)
case <-time.After(30 * time.Second):
    return fmt.Errorf("cefilter: timed out waiting for KV snapshot")
case <-ctx.Done():
    return ctx.Err()
}

// Now start routing events
for event := range eventStream {
    attrs := cefilter.FromSDKEvent(event)
    if !gate.MatchAny(attrs) {
        continue
    }
    for _, url := range gate.MatchEndpoints(attrs) {
        go postWebhook(url, event)
    }
}
```

On reconnect after a NATS blip, `WatchAll` automatically resumes from the watcher's internal consumer sequence. The pod's local `IndexedFilterSet` stays intact — only the delta (events missed during the outage) is replayed.

---

## Failure Modes

| Failure | Effect | Recovery |
|---|---|---|
| NATS KV unavailable | Pods keep their local `IndexedFilterSet` from before the outage — stale but functional. No events are dropped. | KV reconnects, watcher resumes from last consumer seq, delta applied. |
| CDC process restarts | KV not updated during downtime. | CDC resumes from persisted resume token — replays only missed events. No full sync needed. |
| MongoDB and CDC both unavailable | KV and pods are frozen at last known state. | CDC restarts, loads resume token, catches up. If token is stale beyond change stream window, triggers full sync + prune. |
| KV bucket purged / NATS cluster replaced | Pods lose their source of truth on next restart. | CDC detects empty KV (no keys), runs `fullSync` + `pruneStaleKeys` to repopulate from MongoDB. |
| Pod restarts | Local `IndexedFilterSet` is empty. | Watcher runs `WatchAll`, gets current snapshot from KV (O(filter count)), pod is ready in milliseconds. |

---

## Operational Notes

**CDC is a singleton.** Run it as a single-replica deployment (or leader-elected via a NATS KV lock). Multiple CDC instances publishing the same events is safe (KV Put is idempotent) but wasteful. Use NATS KV for the leader lock too:

```go
lock, _ := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: "cefilters_cdc_lock", TTL: 10 * time.Second})
// Compete for leadership by creating the key; loser watches for expiry and retries.
```

**Schema evolution.** Add new fields to `FilterKVValue` with `omitempty`. Old pods ignore unknown fields. Old CDC writes old structs; new pods handle missing fields gracefully. No coordinated rollout required.

**Metrics to watch:**

- CDC lag: `updatedAt` of the latest MongoDB event vs wall clock
- KV key count vs MongoDB active filter count (should match after CDC catches up)
- Pod watcher reconnect count (elevated → NATS instability)
- `gate.Stats().Total` per pod at startup (should match KV key count)
