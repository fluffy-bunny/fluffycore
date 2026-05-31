# cefilter

A lightweight filter expression engine for CloudEvent attributes with built-in webhook fan-out routing.

Write human-readable expressions like SQL `WHERE` clauses, compile them once at startup, then evaluate them against any incoming event with no per-event allocations.

## Imports

```go
import (
    "github.com/fluffy-bunny/fluffycore/cloudevent/cefilter"
    "github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
)
```

---

## Core Concepts

| Type | Package | Description |
|---|---|---|
| `Attrs` | `cefilter` | `map[string]string` — flat view of a CloudEvent's attributes |
| `Predicate` | `cefilter` | Compiled expression tree; call `.Match(attrs)` per event |
| `FilterEntry` | `contracts` | Metadata attached to a registered filter: `Name`, `Hint`, `Metadata` |
| `FilterSet` | `cefilter` | Slice-backed filter collection — **not** thread-safe |
| `IndexedFilterSet` | `cefilter` | Inverted-index filter collection — **thread-safe** (`sync.RWMutex`) |
| `IFilterSet` | `contracts` | Interface satisfied by both filter set types |
| `CloudEventAccessor` | `cefilter` | Adapter interface — decouples the library from any CE SDK |

### FilterEntry

Every registered filter is associated with a `FilterEntry`:

```go
contracts.FilterEntry{
    Name:     "order-placed-filter", // unique key — used by Remove()
    Hint:     "https://svc.example.com/webhook", // opaque routing target
    Metadata: map[string]string{"tier": "gold"}, // optional extra annotations
}
```

`Hint` is entirely caller-defined — use it as a webhook URL, NATS subject, Kafka topic, gRPC handler tag, or any other routing instruction. The engine stores and returns it unchanged.

---

## Expression Syntax

All comparisons are against **string values**. Attribute names are case-sensitive; keywords (`AND`, `OR`, `NOT`, `LIKE`, `IN`, `EXISTS`) are case-insensitive.

### Operators

| Operator | Example | Meaning |
|---|---|---|
| `=` | `type = "com.example.order"` | Exact equality |
| `!=` | `subject != "internal"` | Not equal (also matches if attribute is absent) |
| `LIKE` | `source LIKE "checkout/*"` | Glob match (`*` = any chars, `?` = one char) |
| `IN` | `region IN ("us-east", "eu-west")` | Membership in a set — O(1) lookup |
| `EXISTS` | `EXISTS priority` | Attribute is present (any value) |

### Boolean Combinators

```
NOT EXISTS debug
type = "com.example.order" AND orgid = "org-1"
type = "com.example.order" OR type LIKE "com.acme.*"
(type LIKE "com.example.*" OR type LIKE "com.acme.*") AND orgid = "org-1" AND NOT EXISTS debug
```

Precedence (high to low): `NOT` > `AND` > `OR`. Use parentheses to override.

### String Literals

Both `"double"` and `'single'` quotes are accepted. Backslash escapes work inside strings (`\"`, `\'`, `\\`).

---

## Usage

### 1. Single compiled predicate

```go
pred, err := cefilter.Parse(`type = "com.example.order" AND source LIKE "checkout/*"`)
if err != nil {
    log.Fatal(err) // bad expression — fail at startup
}

attrs := cefilter.Attrs{
    "type":   "com.example.order",
    "source": "checkout/web",
}
if pred.Match(attrs) {
    // handle event
}
```

### 2. With a real CloudEvent (direct SDK integration)

`cloudevents.Event` satisfies `CloudEventAccessor` directly — no adapter needed:

```go
import cloudevents "github.com/cloudevents/sdk-go/v2"

pred, _ := cefilter.Parse(`type = "com.example.order" AND orgid = "org-123"`)

e := cloudevents.NewEvent()
e.SetType("com.example.order")
e.SetSource("checkout/web")
e.SetExtension("orgid", "org-123")

if pred.Match(cefilter.FromSDKEvent(e)) {
    // matched
}
```

For other CE implementations, implement `CloudEventAccessor` once and use `FromCloudEvent`. Extension values are automatically coerced to strings via `fmt.Sprintf`.

### 3. Webhook fan-out with `IndexedFilterSet` (recommended)

The primary use case: an ingress gate that drops irrelevant CEs at O(1) cost, plus routing to the exact set of endpoint URLs that subscribed to the event.

Multiple filter rules can share the same `Hint` (endpoint URL). `MatchEndpoints` deduplicates automatically — you POST to each endpoint exactly once regardless of how many rules matched.

```go
var gate cefilter.IndexedFilterSet

// Register endpoint subscriptions.
// One endpoint can register multiple filter rules — all share the same Hint.
gate.Add(contracts.FilterEntry{Name: "order-placed",  Hint: "https://order-svc/hook"},
    `type = "com.shop.order.placed"`)
gate.Add(contracts.FilterEntry{Name: "order-shipped", Hint: "https://order-svc/hook"},
    `type = "com.shop.order.shipped"`)
gate.Add(contracts.FilterEntry{Name: "audit-all",     Hint: "https://audit-svc/hook"},
    `source LIKE "shop/*"`)

// Per-event — two phases:

// Phase 1: ingress gate — O(1) hash lookup, drops CEs nobody wants.
attrs := cefilter.FromSDKEvent(event)
if !gate.MatchAny(attrs) {
    return // dropped — no subscriber cares about this type
}

// Phase 2: routing — returns unique endpoint URLs, deduplicated by Hint.
// Even though two "order-svc" rules matched, the endpoint appears once.
for _, url := range gate.MatchEndpoints(attrs) {
    go postWebhook(url, event)
}

// Deregister when an endpoint is deleted:
gate.Remove("order-placed")
gate.Remove("order-shipped")
```

### 4. Multi-subscriber routing with `FilterSet`

`FilterSet` is simpler but performs an O(N) scan on every call and is **not thread-safe**. Use it for single-goroutine scenarios or when the filter set is built once and never modified concurrently.

```go
var fs cefilter.FilterSet

fs.Add(contracts.FilterEntry{Name: "webhook-org1", Hint: "https://org1.example.com/hook"},
    `type = "com.example.order" AND orgid = "org-1"`)
fs.Add(contracts.FilterEntry{Name: "webhook-org2", Hint: "https://org2.example.com/hook"},
    `type LIKE "com.example.*" AND orgid = "org-2"`)

// Returns a FilterEntry per matched rule — use .Hint to dispatch:
for _, entry := range fs.Match(attrs) {
    dispatch(entry.Hint, event)
    // entry.Name, entry.Metadata also available
}

// Or get unique endpoint URLs directly:
for _, url := range fs.MatchEndpoints(attrs) {
    dispatch(url, event)
}
```

### 5. Choosing between FilterSet and IndexedFilterSet

| | `FilterSet` | `IndexedFilterSet` |
|---|---|---|
| Thread-safe | ✗ | ✓ |
| Dynamic Add/Remove | ✓ | ✓ |
| No-match cost | O(N) scan | O(1) hash miss |
| Match cost | O(N) | O(matching bucket) |
| Best for | Tests, single goroutine | Production event streams |

---

## Thread Safety

### `FilterSet` — caller must serialize all access

`FilterSet` has **no internal synchronization**. Every method (`Add`, `Remove`, `Match`, `MatchAny`, `MatchEndpoints`) mutates or reads shared slice state. Calling any two of them concurrently — even two `Match` calls — is a data race.

**Correct:** build the `FilterSet` in a single goroutine, then use it from that same goroutine only.

```go
// Safe: build once, use in one goroutine (e.g. inside a single event-processing loop)
var fs cefilter.FilterSet
fs.Add(contracts.FilterEntry{Name: "f1", Hint: "..."}, `type = "com.example.order"`)

for event := range eventChan { // only one goroutine here
    for _, entry := range fs.Match(cefilter.FromSDKEvent(event)) {
        dispatch(entry.Hint, event)
    }
}
```

If you need concurrent access with `FilterSet`, wrap it in your own `sync.RWMutex`.

### `IndexedFilterSet` — all methods are concurrency-safe

`Add`, `Remove`, `Match`, `MatchAny`, and `MatchEndpoints` may all be called simultaneously from any number of goroutines. Internally, reads use `sync.RWMutex` (multiple readers in parallel) and writes hold an exclusive lock only for the map insert — typically microseconds.

```go
var gate cefilter.IndexedFilterSet

// These two goroutines are safe — no external lock needed:
go func() {
    gate.Add(contracts.FilterEntry{Name: "f1", Hint: "..."}, `type = "com.shop.order"`)
}()
go func() {
    entries := gate.Match(cefilter.FromSDKEvent(event))
    _ = entries
}()
```

**Exception — the two exported fields:** `MaxMatches` and `OnLimitExceeded` are plain struct fields, not protected by the internal mutex. Set them **before** the first concurrent call.

```go
// Correct: configure fields, then start goroutines.
var gate cefilter.IndexedFilterSet
gate.MaxMatches = 20
gate.OnLimitExceeded = func(a cefilter.Attrs, count int) { /* log */ }

// Now safe to pass gate to multiple goroutines.
go processEvents(&gate)
go manageSubscriptions(&gate)
```

Mutating `MaxMatches` or `OnLimitExceeded` after concurrent use begins is a data race.

### `Predicate` (from `Parse`) — immutable, safe to share

A `Predicate` value returned by `Parse` is a read-only expression tree. Calling `pred.Match(attrs)` from many goroutines simultaneously is safe with no synchronization.

```go
pred, _ := cefilter.Parse(`type = "com.example.order"`)

// Safe: pred is immutable after Parse returns.
for i := 0; i < 100; i++ {
    go func() {
        _ = pred.Match(attrs)
    }()
}
```

### `Attrs` and result slices — owned by the caller

- `FromCloudEvent` / `FromSDKEvent` return a **new** `Attrs` map on every call. There is no shared state to worry about.
- `Match` returns a **new** `[]contracts.FilterEntry` slice each call.
- `MatchEndpoints` returns a **new** `[]string` slice each call.

You can safely pass these return values to other goroutines without copying.

### Summary

| What | Safe to share across goroutines? | Action required |
|---|---|---|
| `FilterSet` | ✗ | Use from one goroutine, or add your own mutex |
| `IndexedFilterSet` methods | ✓ | Nothing — internal RWMutex handles it |
| `IndexedFilterSet.MaxMatches` | ✗ (plain field) | Set once before concurrent use |
| `IndexedFilterSet.OnLimitExceeded` | ✗ (plain field) | Set once before concurrent use |
| `Predicate` (from `Parse`) | ✓ | Nothing — immutable after creation |
| `Attrs` map (from `FromSDKEvent`) | ✓ | Each call returns a fresh map |
| `[]FilterEntry` / `[]string` results | ✓ | Each call returns a fresh slice |

---

## Circuit Breaker

Guard against accidentally broad filters (e.g. `EXISTS type` matches everything) flooding downstream endpoints.

```go
var gate cefilter.IndexedFilterSet
gate.MaxMatches = 10
gate.OnLimitExceeded = func(a cefilter.Attrs, count int) {
    slog.Error("CE fan-out limit hit — possible misconfigured wildcard filter",
        "type", a["type"], "source", a["source"], "matched", count)
}
```

- `MaxMatches = 0` means unlimited (default zero value).
- When the limit is reached, `Match` / `MatchEndpoints` stops collecting and calls `OnLimitExceeded` exactly once, then returns the capped slice.
- `MatchAny` is unaffected — it always short-circuits on the first match.
- Set `MaxMatches` and `OnLimitExceeded` before concurrent use begins.

---

## IFilterSet Interface

Both `FilterSet` and `IndexedFilterSet` satisfy `contracts.IFilterSet`:

```go
type IFilterSet interface {
    Add(entry FilterEntry, expr string) error
    Remove(name string)
    Match(attrs map[string]string) []FilterEntry
    MatchEndpoints(attrs map[string]string) []string
    MatchAny(attrs map[string]string) bool
}
```

Use the interface to swap implementations or write tests against a mock.

---

## Standard Attributes

`FromCloudEvent` / `FromSDKEvent` always populates these keys:

| Key | Source |
|---|---|
| `type` | `e.Type()` |
| `source` | `e.Source()` |
| `id` | `e.ID()` |
| `subject` | `e.Subject()` (omitted if empty) |
| `<extension>` | All keys from `e.Extensions()` |

---

## LIKE Glob Reference

| Pattern | Matches | Does not match |
|---|---|---|
| `checkout/*` | `checkout/web`, `checkout/mobile/v2` | `checkout`, `payments/web` |
| `com.acme.*` | `com.acme.order`, `com.acme.payment` | `com.example.order` |
| `com.*.order` | `com.acme.order`, `com.example.order` | `com.acme.payment` |
| `event-?` | `event-1`, `event-a` | `event-12` |

Globs are anchored to the full value — `checkout/*` will not match `x/checkout/web`.

---

## Error Handling

`Parse` and `Add` return an error for any invalid expression. Validate at startup so bad configuration fails fast.

```go
pred, err := cefilter.Parse(`type = "missing close`)
// err: unterminated string starting at position ...
```

The compiled `Predicate` exposes `String()` for debug-friendly reconstruction:

```go
fmt.Println(pred.String())
// ((type = "com.example.order" OR type LIKE "com.acme.*") AND orgid = "org-1")
```

---

## Design Notes

- **Compile once, evaluate many times.** `Parse` allocates a predicate tree; `Match` is allocation-free.
- **Short-circuit evaluation.** `AND` stops at the first false; `OR` stops at the first true.
- **`IN` uses a hash set** — O(1) membership check regardless of set size.
- **`LIKE` globs are compiled to `*regexp.Regexp` at parse time**, never per call.
- **`IndexedFilterSet` builds an inverted index on `type`** at `Add` time. For an event whose type no filter cares about: one hash lookup + RUnlock ≈ 25 ns.
- **`MatchEndpoints` deduplicates by `Hint`** — multiple rules pointing to the same endpoint result in one dispatch.
- **No SDK dependency in the core.** `CloudEventAccessor` keeps `cefilter` import-free from `cloudevents/sdk-go`. `FromSDKEvent` lives in the separate `sdk.go` file for callers who do import the SDK.


## Import

```go
import "github.com/fluffy-bunny/fluffycore/cloudevent/cefilter"
```

---

## Core Concepts

| Type | Description |
|---|---|
| `Attrs` | `map[string]string` — flat view of an event's attributes |
| `Predicate` | Compiled expression tree; call `.Match(attrs)` per event |
| `FilterSet` | Named collection of predicates; route one event to many handlers |
| `CloudEventAccessor` | Interface adapter that decouples the library from the CE SDK |

---

## Expression Syntax

All comparisons are against **string values**. Attribute names are case-sensitive; keywords (`AND`, `OR`, `NOT`, `LIKE`, `IN`, `EXISTS`) are case-insensitive.

### Operators

| Operator | Example | Meaning |
|---|---|---|
| `=` | `type = "com.example.order"` | Exact equality |
| `!=` | `subject != "internal"` | Not equal (also matches if attribute is absent) |
| `LIKE` | `source LIKE "checkout/*"` | Glob match (`*` = any chars, `?` = one char) |
| `IN` | `region IN ("us-east", "eu-west")` | Membership in a set — O(1) lookup |
| `EXISTS` | `EXISTS priority` | Attribute is present (any value) |

### Boolean Combinators

```
NOT EXISTS debug
type = "com.example.order" AND orgid = "org-1"
type = "com.example.order" OR type LIKE "com.acme.*"
(type LIKE "com.example.*" OR type LIKE "com.acme.*") AND orgid = "org-1" AND NOT EXISTS debug
```

Precedence (high to low): `NOT` > `AND` > `OR`. Use parentheses to override.

### String Literals

Both `"double"` and `'single'` quotes are accepted. Backslash escapes work inside strings (`\"`, `\'`, `\\`).

---

## Usage

### 1. Single filter

```go
pred, err := cefilter.Parse(`type = "com.example.order" AND source LIKE "checkout/*"`)
if err != nil {
    log.Fatal(err) // bad expression — fail at startup
}

// Later, per event:
attrs := cefilter.Attrs{
    "type":   "com.example.order",
    "source": "checkout/web",
}
if pred.Match(attrs) {
    // handle event
}
```

### 2. With a real CloudEvent (SDK-agnostic)

Implement `CloudEventAccessor` once for your event type, then use `FromCloudEvent`:

```go
// Adapter for cloudevents.Event from github.com/cloudevents/sdk-go
type sdkEvent struct{ e cloudevents.Event }

func (s sdkEvent) Type() string               { return s.e.Type() }
func (s sdkEvent) Source() string             { return s.e.Source() }
func (s sdkEvent) Subject() string            { return s.e.Subject() }
func (s sdkEvent) ID() string                 { return s.e.ID() }
func (s sdkEvent) Extensions() map[string]any { return s.e.Extensions() }

// Compile once:
pred, _ := cefilter.Parse(`type = "com.example.order" AND orgid = "org-123"`)

// Per event:
attrs := cefilter.FromCloudEvent(sdkEvent{event})
if pred.Match(attrs) {
    // ...
}
```

Extension values are automatically coerced to strings via `fmt.Sprintf`.

### 3. Multi-tenant webhook routing with `FilterSet`

```go
var fs cefilter.FilterSet

// Register at startup — all expressions are compiled here, not per-event:
fs.Add("webhook-org1", `type = "com.example.order" AND source LIKE "checkout/*" AND orgid = "org-1"`)
fs.Add("webhook-org2", `type LIKE "com.example.*" AND orgid = "org-2"`)
fs.Add("webhook-org3", `type LIKE "com.*.order" AND region IN ("us-east", "eu-west") AND orgid = "org-3"`)

// Per event — returns names of every filter that matched:
matched := fs.Match(attrs)
// e.g. ["webhook-org1", "webhook-org3"]

// Or short-circuit as soon as any filter matches:
if fs.MatchAny(attrs) {
    // ...
}
```

---

## Standard Attributes

`FromCloudEvent` always populates these keys:

| Key | Source |
|---|---|
| `type` | `e.Type()` |
| `source` | `e.Source()` |
| `id` | `e.ID()` |
| `subject` | `e.Subject()` (omitted if empty) |
| `<extension>` | All keys from `e.Extensions()` |

---

## LIKE Glob Reference

| Pattern | Matches | Does not match |
|---|---|---|
| `checkout/*` | `checkout/web`, `checkout/mobile/v2` | `checkout`, `payments/web` |
| `com.acme.*` | `com.acme.order`, `com.acme.payment` | `com.example.order` |
| `com.*.order` | `com.acme.order`, `com.example.order` | `com.acme.payment` |
| `event-?` | `event-1`, `event-a` | `event-12` |

Globs are anchored to the full value — `checkout/*` will not match `x/checkout/web`.

---

## Error Handling

`Parse` and `FilterSet.Add` return an error for any invalid expression. Validate at startup so bad configuration fails fast rather than silently passing everything at runtime.

```go
pred, err := cefilter.Parse(`type = "missing close`)
// err: unterminated string starting at position ...
```

The compiled `Predicate` exposes `String()` for debug-friendly reconstruction of the expression tree:

```go
fmt.Println(pred.String())
// ((type = "com.example.order" OR type LIKE "com.acme.*") AND orgid = "org-1")
```

---

## Design Notes

- **Compile once, evaluate many times.** `Parse` allocates a predicate tree; `Match` is allocation-free.
- **Short-circuit evaluation.** `AND` stops at the first false; `OR` stops at the first true.
- **`IN` uses a hash set**, not a slice scan — O(1) regardless of set size.
- **`LIKE` globs are compiled to `*regexp.Regexp` at parse time**, never per call.
- **No SDK dependency.** The `CloudEventAccessor` interface keeps this package import-free from `cloudevents/sdk-go`.
