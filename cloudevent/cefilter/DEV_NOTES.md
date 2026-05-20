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
