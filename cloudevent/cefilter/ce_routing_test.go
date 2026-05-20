package cefilter_test

// ce_routing_test.go — real CloudEvent routing scenarios.
//
// Each filter is registered with a FilterEntry that carries:
//   - Name: unique identifier for this subscription
//   - Hint: where to deliver the matching event (NATS subject, handler tag, etc.)
//   - Metadata: optional extra routing annotations
//
// When a CE arrives, Match() returns []FilterEntry — one per matched subscription.
// The caller iterates the results and dispatches to each entry's Hint.

import (
	"fmt"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/fluffy-bunny/fluffycore/cloudevent/cefilter"
	"github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newEvent(typ, source, subject string, extensions map[string]string) cloudevents.Event {
	e := cloudevents.NewEvent()
	e.SetType(typ)
	e.SetSource(source)
	if subject != "" {
		e.SetSubject(subject)
	}
	e.SetID("test-id")
	for k, v := range extensions {
		e.SetExtension(k, v)
	}
	return e
}

// entryNames extracts just the Name field from a slice of FilterEntry.
func entryNames(entries []contracts.FilterEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Test 1: fan-out with hints — the core use-case
//
// A CE matches 5 subscriptions. Each has a Hint saying where to send it.
// The caller iterates []FilterEntry and dispatches to each entry's Hint.
// ---------------------------------------------------------------------------

func TestCE_FanOut_WithHints(t *testing.T) {
	var gate cefilter.IndexedFilterSet

	subscriptions := []struct {
		entry contracts.FilterEntry
		expr  string
	}{
		{contracts.FilterEntry{Name: "order-svc", Hint: "nats://orders.placed"}, `type = "com.shop.order.placed"`},
		{contracts.FilterEntry{Name: "fraud-svc", Hint: "nats://fraud.check"}, `type LIKE "com.shop.order.*"`},
		{contracts.FilterEntry{Name: "email-svc", Hint: "nats://email.dispatch"}, `type IN ("com.shop.order.placed","com.shop.order.shipped")`},
		{contracts.FilterEntry{
			Name:     "audit-svc",
			Hint:     "nats://audit.log",
			Metadata: map[string]string{"persist": "true", "tier": "compliance"},
		}, `source LIKE "shop/*"`},
		{contracts.FilterEntry{Name: "analytics-svc", Hint: "kafka://events.raw"}, `type LIKE "com.shop.*"`},
	}
	for _, s := range subscriptions {
		if err := gate.Add(s.entry, s.expr); err != nil {
			t.Fatalf("Add(%q): %v", s.entry.Name, err)
		}
	}

	// "order placed" matches all 5 subscriptions.
	e := newEvent("com.shop.order.placed", "shop/checkout", "order/42", nil)
	matched := gate.Match(cefilter.FromSDKEvent(e))

	if len(matched) != 5 {
		t.Fatalf("expected 5 matches, got %d: %v", len(matched), entryNames(matched))
	}

	// Simulate what a real dispatcher does — deliver to each Hint.
	delivered := make(map[string]string) // name → hint
	for _, entry := range matched {
		delivered[entry.Name] = entry.Hint
		t.Logf("dispatch %q → %s", entry.Name, entry.Hint)
	}

	wantHints := map[string]string{
		"order-svc":     "nats://orders.placed",
		"fraud-svc":     "nats://fraud.check",
		"email-svc":     "nats://email.dispatch",
		"audit-svc":     "nats://audit.log",
		"analytics-svc": "kafka://events.raw",
	}
	for name, want := range wantHints {
		if got := delivered[name]; got != want {
			t.Errorf("%s: hint = %q, want %q", name, got, want)
		}
	}

	// Metadata is preserved on the audit entry.
	for _, entry := range matched {
		if entry.Name == "audit-svc" {
			if entry.Metadata["persist"] != "true" || entry.Metadata["tier"] != "compliance" {
				t.Errorf("audit-svc metadata not preserved: %v", entry.Metadata)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Test 2: partial fan-out — region-scoped subscriptions
// ---------------------------------------------------------------------------

func TestCE_FanOut_PartialMatch(t *testing.T) {
	var gate cefilter.IndexedFilterSet
	gate.Add(contracts.FilterEntry{Name: "us-handler", Hint: "nats://region.us"}, `type = "com.shop.order.placed" AND region = "us-east"`)
	gate.Add(contracts.FilterEntry{Name: "eu-handler", Hint: "nats://region.eu"}, `type = "com.shop.order.placed" AND region = "eu-west"`)
	gate.Add(contracts.FilterEntry{Name: "global-handler", Hint: "nats://region.all"}, `type = "com.shop.order.placed"`)

	// US order → us-handler + global-handler; eu-handler excluded.
	usOrder := newEvent("com.shop.order.placed", "shop/checkout", "", map[string]string{"region": "us-east"})
	matched := gate.Match(cefilter.FromSDKEvent(usOrder))

	names := make(map[string]bool)
	for _, e := range matched {
		names[e.Name] = true
	}

	if !names["us-handler"] {
		t.Error("us-handler should have matched")
	}
	if !names["global-handler"] {
		t.Error("global-handler should have matched")
	}
	if names["eu-handler"] {
		t.Error("eu-handler should NOT have matched for a us-east event")
	}
}

// ---------------------------------------------------------------------------
// Test 3: subscriber routing — verifies which named services receive each CE
// ---------------------------------------------------------------------------

func TestCE_SubscriberRouting(t *testing.T) {
	var fs cefilter.FilterSet

	mustAdd := func(name, hint, expr string) {
		if err := fs.Add(contracts.FilterEntry{Name: name, Hint: hint}, expr); err != nil {
			t.Fatalf("Add(%q): %v", name, err)
		}
	}

	mustAdd("order-service", "nats://orders", `type = "com.shop.order.placed"`)
	mustAdd("fraud-service", "nats://fraud", `type LIKE "com.shop.order.*" AND region = "us-east"`)
	mustAdd("email-service", "nats://email", `type IN ("com.shop.order.placed","com.shop.order.shipped")`)
	mustAdd("audit-service", "nats://audit", `source LIKE "shop/*"`)
	mustAdd("billing-service", "nats://billing", `type = "com.shop.payment.captured" AND EXISTS invoiceid`)

	tests := []struct {
		name      string
		event     cloudevents.Event
		wantNames []string
	}{
		{
			name:      "order placed — four services receive it",
			event:     newEvent("com.shop.order.placed", "shop/checkout", "order/42", map[string]string{"region": "us-east"}),
			wantNames: []string{"order-service", "fraud-service", "email-service", "audit-service"},
		},
		{
			name:      "order shipped — email + audit only",
			event:     newEvent("com.shop.order.shipped", "shop/warehouse", "order/42", map[string]string{"region": "eu-west"}),
			wantNames: []string{"email-service", "audit-service"},
		},
		{
			name:      "payment with invoice — billing + audit",
			event:     newEvent("com.shop.payment.captured", "shop/payments", "", map[string]string{"invoiceid": "inv-99"}),
			wantNames: []string{"audit-service", "billing-service"},
		},
		{
			name:      "payment without invoice — billing excluded by EXISTS",
			event:     newEvent("com.shop.payment.captured", "shop/payments", "", nil),
			wantNames: []string{"audit-service"},
		},
		{
			name:      "fraud from eu — fraud-service excluded by region filter",
			event:     newEvent("com.shop.order.placed", "shop/checkout", "order/99", map[string]string{"region": "eu-west"}),
			wantNames: []string{"order-service", "email-service", "audit-service"},
		},
		{
			name:      "internal metrics from non-shop source — nobody wants it",
			event:     newEvent("com.shop.internal.metrics", "monitor/internal", "", nil),
			wantNames: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := entryNames(fs.Match(cefilter.FromSDKEvent(tc.event)))
			if !stringSliceEqual(got, tc.wantNames) {
				t.Errorf("got  %v\nwant %v", got, tc.wantNames)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test 4: ingress gate — drop events no subscriber cares about
// ---------------------------------------------------------------------------

func TestCE_IngressGate(t *testing.T) {
	var gate cefilter.IndexedFilterSet

	mustAdd := func(name, expr string) {
		if err := gate.Add(contracts.FilterEntry{Name: name}, expr); err != nil {
			t.Fatalf("Add(%q): %v", name, err)
		}
	}
	mustAdd("order-service", `type = "com.shop.order.placed"`)
	mustAdd("fraud-service", `type LIKE "com.shop.order.*"`)
	mustAdd("email-service", `type IN ("com.shop.order.placed","com.shop.order.shipped")`)
	mustAdd("billing-service", `type = "com.shop.payment.captured"`)

	tests := []struct {
		name     string
		event    cloudevents.Event
		wantPass bool // true = at least one subscriber wants it
	}{
		{
			name:     "order placed — passes gate",
			event:    newEvent("com.shop.order.placed", "shop/checkout", "", nil),
			wantPass: true,
		},
		{
			name:     "order shipped — passes gate",
			event:    newEvent("com.shop.order.shipped", "shop/warehouse", "", nil),
			wantPass: true,
		},
		{
			name:     "payment captured — passes gate",
			event:    newEvent("com.shop.payment.captured", "shop/payments", "", nil),
			wantPass: true,
		},
		{
			name:     "internal metrics — DROPPED at gate (O(1) hash miss)",
			event:    newEvent("com.shop.internal.metrics", "shop/monitor", "", nil),
			wantPass: false,
		},
		{
			name:     "user login event — DROPPED",
			event:    newEvent("com.auth.user.login", "auth/idp", "", nil),
			wantPass: false,
		},
		{
			name:     "completely unknown type — DROPPED",
			event:    newEvent("com.noise.irrelevant", "somewhere/else", "", nil),
			wantPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := cefilter.FromSDKEvent(tc.event)
			got := gate.MatchAny(attrs)
			if got != tc.wantPass {
				t.Errorf("MatchAny=%v, want %v (event type=%q)", got, tc.wantPass, tc.event.Type())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test 3: dynamic filter addition — new subscriber joins while events flow
// ---------------------------------------------------------------------------

func TestCE_DynamicSubscriberAdd(t *testing.T) {
	var gate cefilter.IndexedFilterSet

	event := newEvent("com.shop.order.placed", "shop/checkout", "", nil)
	attrs := cefilter.FromSDKEvent(event)

	// Before any subscriber registers → event dropped.
	if gate.MatchAny(attrs) {
		t.Fatal("expected no match before any filters registered")
	}

	// First subscriber registers.
	if err := gate.Add(contracts.FilterEntry{Name: "order-service", Hint: "nats://orders"}, `type = "com.shop.order.placed"`); err != nil {
		t.Fatal(err)
	}
	if !gate.MatchAny(attrs) {
		t.Fatal("expected match after order-service registered")
	}

	// Second subscriber registers — gate still passes.
	if err := gate.Add(contracts.FilterEntry{Name: "audit-service", Hint: "nats://audit"}, `source LIKE "shop/*"`); err != nil {
		t.Fatal(err)
	}
	if !gate.MatchAny(attrs) {
		t.Fatal("expected match with two subscribers")
	}

	// An event no current subscriber wants is still dropped.
	noise := cefilter.FromSDKEvent(newEvent("com.noise.ping", "heartbeat/checker", "", nil))
	if gate.MatchAny(noise) {
		t.Fatal("expected noise event to be dropped")
	}
}

// ---------------------------------------------------------------------------
// Test 5: IndexedFilterSet.Match — returns FilterEntry for each matched filter
// ---------------------------------------------------------------------------

func TestCE_IndexedMatch_ReturnsSubscriberNames(t *testing.T) {
	var gate cefilter.IndexedFilterSet
	gate.Add(contracts.FilterEntry{Name: "order-service", Hint: "nats://orders"}, `type = "com.shop.order.placed"`)
	gate.Add(contracts.FilterEntry{Name: "email-service", Hint: "nats://email"}, `type IN ("com.shop.order.placed","com.shop.order.shipped")`)
	gate.Add(contracts.FilterEntry{Name: "audit-service", Hint: "nats://audit"}, `source LIKE "shop/*"`)

	e := newEvent("com.shop.order.placed", "shop/checkout", "", nil)
	matched := gate.Match(cefilter.FromSDKEvent(e))

	want := map[string]bool{
		"order-service": true,
		"email-service": true,
		"audit-service": true,
	}
	for _, entry := range matched {
		if !want[entry.Name] {
			t.Errorf("unexpected subscriber %q in result", entry.Name)
		}
		if entry.Hint == "" {
			t.Errorf("%s: Hint should not be empty", entry.Name)
		}
		delete(want, entry.Name)
	}
	for name := range want {
		t.Errorf("subscriber %q was expected but not returned", name)
	}
}

// ---------------------------------------------------------------------------
// Poison pill / circuit breaker tests
//
// If someone accidentally registers EXISTS type (matches everything) or a
// wildcard LIKE "*" filter, a single CE could fan-out to thousands of destinations.
// MaxMatches + OnLimitExceeded guard against this at runtime.
// ---------------------------------------------------------------------------

// TestCE_PoisonPill_IndexedFilterSet verifies that IndexedFilterSet.Match
// stops at MaxMatches and fires the callback when the limit is reached.
func TestCE_PoisonPill_IndexedFilterSet(t *testing.T) {
	var gate cefilter.IndexedFilterSet
	gate.MaxMatches = 3

	var callbackAttrs cefilter.Attrs
	var callbackCount int
	gate.OnLimitExceeded = func(a cefilter.Attrs, count int) {
		callbackAttrs = a
		callbackCount = count
	}

	// Register 6 broad filters — all match any "com.shop.*" event.
	for i := range 6 {
		gate.Add(
			contracts.FilterEntry{Name: fmt.Sprintf("sub-%d", i), Hint: fmt.Sprintf("nats://dest.%d", i)},
			`type LIKE "com.shop.*"`,
		)
	}

	e := newEvent("com.shop.order.placed", "shop/checkout", "", nil)
	matched := gate.Match(cefilter.FromSDKEvent(e))

	if len(matched) != 3 {
		t.Errorf("expected 3 (MaxMatches), got %d", len(matched))
	}
	if callbackCount != 3 {
		t.Errorf("OnLimitExceeded called with count=%d, want 3", callbackCount)
	}
	if callbackAttrs["type"] != "com.shop.order.placed" {
		t.Errorf("OnLimitExceeded attrs type = %q, want %q", callbackAttrs["type"], "com.shop.order.placed")
	}
}

// TestCE_PoisonPill_FilterSet verifies the same behaviour for FilterSet.
func TestCE_PoisonPill_FilterSet(t *testing.T) {
	var fs cefilter.FilterSet
	fs.MaxMatches = 2

	callCount := 0
	fs.OnLimitExceeded = func(_ cefilter.Attrs, _ int) { callCount++ }

	for i := range 5 {
		fs.Add(
			contracts.FilterEntry{Name: fmt.Sprintf("f-%d", i)},
			`type = "com.example.event"`,
		)
	}

	matched := fs.Match(cefilter.Attrs{"type": "com.example.event"})

	if len(matched) != 2 {
		t.Errorf("expected 2 (MaxMatches), got %d", len(matched))
	}
	if callCount != 1 {
		t.Errorf("OnLimitExceeded called %d time(s), want exactly 1", callCount)
	}
}

// TestCE_PoisonPill_NoTriggerUnderLimit verifies that OnLimitExceeded is NOT
// called when the number of matches stays below MaxMatches.
func TestCE_PoisonPill_NoTriggerUnderLimit(t *testing.T) {
	var gate cefilter.IndexedFilterSet
	gate.MaxMatches = 10 // higher than the number of matching filters

	triggered := false
	gate.OnLimitExceeded = func(_ cefilter.Attrs, _ int) { triggered = true }

	for i := range 3 {
		gate.Add(
			contracts.FilterEntry{Name: fmt.Sprintf("s-%d", i)},
			`type = "com.example.event"`,
		)
	}

	matched := gate.Match(cefilter.Attrs{"type": "com.example.event"})

	if len(matched) != 3 {
		t.Errorf("expected 3 matches, got %d", len(matched))
	}
	if triggered {
		t.Error("OnLimitExceeded should NOT fire when matches < MaxMatches")
	}
}

// TestCE_PoisonPill_ZeroMaxMatchesUnlimited verifies that MaxMatches=0 means
// no limit — all matches are returned and the callback is never invoked.
func TestCE_PoisonPill_ZeroMaxMatchesUnlimited(t *testing.T) {
	var gate cefilter.IndexedFilterSet
	// MaxMatches zero value = unlimited (no circuit breaker)
	triggered := false
	gate.OnLimitExceeded = func(_ cefilter.Attrs, _ int) { triggered = true }

	for i := range 20 {
		gate.Add(
			contracts.FilterEntry{Name: fmt.Sprintf("s-%d", i)},
			`type = "com.example.event"`,
		)
	}

	matched := gate.Match(cefilter.Attrs{"type": "com.example.event"})

	if len(matched) != 20 {
		t.Errorf("expected 20 matches (unlimited), got %d", len(matched))
	}
	if triggered {
		t.Error("OnLimitExceeded must not fire when MaxMatches = 0")
	}
}

// ---------------------------------------------------------------------------
// Webhook fan-out — the primary use case
//
// Pattern:
//  1. Register webhook endpoints (store URL → endpoint metadata externally).
//  2. For each endpoint, add one or more filter rules with Hint = endpoint URL.
//  3. On CE ingress: MatchAny() to gate (drop if nobody wants it — O(1)).
//  4. On CE routing:  MatchEndpoints() to get the unique set of destination URLs.
//     Even if 3 filter rules share the same endpoint URL, you POST only once.
// ---------------------------------------------------------------------------

// TestCE_WebhookFanOut is the canonical webhook fan-out scenario.
// Two endpoints are registered. One endpoint has three filter rules.
// MatchEndpoints deduplicates so each endpoint receives the event exactly once.
func TestCE_WebhookFanOut(t *testing.T) {
	var gate cefilter.IndexedFilterSet

	const (
		orderEndpoint = "https://order-svc.example.com/webhook"
		auditEndpoint = "https://audit-svc.example.com/webhook"
	)

	// Endpoint 1 — three filter rules, all pointing to the same URL.
	gate.Add(contracts.FilterEntry{Name: "order-placed", Hint: orderEndpoint},
		`type = "com.shop.order.placed"`)
	gate.Add(contracts.FilterEntry{Name: "order-shipped", Hint: orderEndpoint},
		`type = "com.shop.order.shipped"`)
	gate.Add(contracts.FilterEntry{Name: "order-any", Hint: orderEndpoint},
		`type LIKE "com.shop.order.*"`)

	// Endpoint 2 — one broad audit filter.
	gate.Add(contracts.FilterEntry{Name: "audit-all-shop", Hint: auditEndpoint},
		`source LIKE "shop/*"`)

	e := newEvent("com.shop.order.placed", "shop/checkout", "", nil)
	attrs := cefilter.FromSDKEvent(e)

	// Phase 1: ingress gate — should this CE be processed at all?
	if !gate.MatchAny(attrs) {
		t.Fatal("CE should pass the ingress gate")
	}

	// Phase 2: routing — which endpoint URLs should receive it?
	// Three order-svc rules match, but MatchEndpoints deduplicates:
	// result is exactly [orderEndpoint, auditEndpoint].
	endpoints := gate.MatchEndpoints(attrs)

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 distinct endpoints, got %d: %v", len(endpoints), endpoints)
	}
	endpointSet := make(map[string]bool)
	for _, url := range endpoints {
		endpointSet[url] = true
		t.Logf("POST → %s", url)
	}
	if !endpointSet[orderEndpoint] {
		t.Error("order endpoint missing from dispatch list")
	}
	if !endpointSet[auditEndpoint] {
		t.Error("audit endpoint missing from dispatch list")
	}

	// A CE that nobody subscribes to is dropped at the gate — no POSTs.
	noise := cefilter.FromSDKEvent(newEvent("com.noise.ping", "heartbeat/checker", "", nil))
	if gate.MatchAny(noise) {
		t.Fatal("noise event should be dropped at gate")
	}
}

// TestCE_WebhookFanOut_EndpointDeregistration verifies that Remove() cleanly
// unregisters all filter rules for an endpoint without affecting others.
func TestCE_WebhookFanOut_EndpointDeregistration(t *testing.T) {
	var gate cefilter.IndexedFilterSet

	gate.Add(contracts.FilterEntry{Name: "order-placed", Hint: "https://svc-a/hook"},
		`type = "com.shop.order.placed"`)
	gate.Add(contracts.FilterEntry{Name: "order-shipped", Hint: "https://svc-a/hook"},
		`type = "com.shop.order.shipped"`)
	gate.Add(contracts.FilterEntry{Name: "audit-all", Hint: "https://svc-b/hook"},
		`source LIKE "shop/*"`)

	attrs := cefilter.FromSDKEvent(newEvent("com.shop.order.placed", "shop/checkout", "", nil))

	// Both endpoints active.
	endpoints := gate.MatchEndpoints(attrs)
	if len(endpoints) != 2 {
		t.Fatalf("before remove: expected 2 endpoints, got %d", len(endpoints))
	}

	// svc-a deregisters — remove both its filter rules.
	gate.Remove("order-placed")
	gate.Remove("order-shipped")

	endpoints = gate.MatchEndpoints(attrs)
	if len(endpoints) != 1 || endpoints[0] != "https://svc-b/hook" {
		t.Errorf("after remove: expected [svc-b], got %v", endpoints)
	}

	// After removing all order-type filters, an order event still passes the gate
	// because audit-all (source LIKE "shop/*") is still registered.
	if !gate.MatchAny(attrs) {
		t.Error("gate should still pass — audit-all filter is still active")
	}

	// Remove the last filter.
	gate.Remove("audit-all")
	if gate.MatchAny(attrs) {
		t.Error("gate should drop event after all filters removed")
	}
}

// TestCE_Remove_FilterSet verifies Remove on FilterSet.
func TestCE_Remove_FilterSet(t *testing.T) {
	var fs cefilter.FilterSet
	fs.Add(contracts.FilterEntry{Name: "f1", Hint: "https://svc-a/hook"}, `type = "com.example.a"`)
	fs.Add(contracts.FilterEntry{Name: "f2", Hint: "https://svc-b/hook"}, `type = "com.example.b"`)
	fs.Add(contracts.FilterEntry{Name: "f3", Hint: "https://svc-a/hook"}, `type = "com.example.a"`)

	attrs := cefilter.Attrs{"type": "com.example.a"}

	if got := fs.MatchEndpoints(attrs); len(got) != 1 || got[0] != "https://svc-a/hook" {
		t.Fatalf("before remove: expected [svc-a], got %v", got)
	}

	fs.Remove("f1")
	// f3 still points to svc-a
	if got := fs.MatchEndpoints(attrs); len(got) != 1 || got[0] != "https://svc-a/hook" {
		t.Errorf("after remove f1: expected [svc-a] (f3 still active), got %v", got)
	}

	fs.Remove("f3")
	// now no filter matches com.example.a
	if got := fs.MatchEndpoints(attrs); len(got) != 0 {
		t.Errorf("after remove f3: expected [], got %v", got)
	}
	if fs.MatchAny(attrs) {
		t.Error("MatchAny should be false after all matching filters removed")
	}
}
