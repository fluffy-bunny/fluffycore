package cefilter_test

import (
	"fmt"
	"testing"

	"github.com/fluffy-bunny/fluffycore/cloudevent/cefilter"
	"github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
)

// mockEvent satisfies CloudEventAccessor without pulling in the CE SDK.
type mockEvent struct {
	typ, source, subject, id string
	exts                     map[string]any
}

func (m mockEvent) Type() string               { return m.typ }
func (m mockEvent) Source() string             { return m.source }
func (m mockEvent) Subject() string            { return m.subject }
func (m mockEvent) ID() string                 { return m.id }
func (m mockEvent) Extensions() map[string]any { return m.exts }

func TestParse_EqualityMatch(t *testing.T) {
	pred, err := cefilter.Parse(`type = "com.example.order"`)
	if err != nil {
		t.Fatal(err)
	}
	attrs := cefilter.Attrs{"type": "com.example.order"}
	if !pred.Match(attrs) {
		t.Errorf("expected match, got none")
	}
	attrs["type"] = "com.example.payment"
	if pred.Match(attrs) {
		t.Errorf("expected no match, got one")
	}
}

func TestParse_LikeGlob(t *testing.T) {
	pred, err := cefilter.Parse(`source LIKE "checkout/*"`)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		source string
		want   bool
	}{
		{"checkout/web", true},
		{"checkout/mobile/v2", true},
		{"payments/web", false},
		{"checkout", false},
	}
	for _, c := range cases {
		got := pred.Match(cefilter.Attrs{"source": c.source})
		if got != c.want {
			t.Errorf("source=%q: got %v, want %v", c.source, got, c.want)
		}
	}
}

func TestParse_TypeGlob(t *testing.T) {
	// com.acme.* — trailing wildcard
	pred, _ := cefilter.Parse(`type LIKE "com.acme.*"`)
	if !pred.Match(cefilter.Attrs{"type": "com.acme.order"}) {
		t.Error("expected match for com.acme.order")
	}
	// com.*.order — middle wildcard
	pred2, _ := cefilter.Parse(`type LIKE "com.*.order"`)
	if !pred2.Match(cefilter.Attrs{"type": "com.acme.order"}) {
		t.Error("expected match for com.acme.order with middle wildcard")
	}
	if pred2.Match(cefilter.Attrs{"type": "com.acme.payment"}) {
		t.Error("expected no match for com.acme.payment")
	}
}

func TestParse_In(t *testing.T) {
	pred, err := cefilter.Parse(`region IN ("us-east", "eu-west", "ap-south")`)
	if err != nil {
		t.Fatal(err)
	}
	if !pred.Match(cefilter.Attrs{"region": "eu-west"}) {
		t.Error("expected match for eu-west")
	}
	if pred.Match(cefilter.Attrs{"region": "us-west"}) {
		t.Error("expected no match for us-west")
	}
}

func TestParse_Exists(t *testing.T) {
	pred, _ := cefilter.Parse(`EXISTS priority`)
	if pred.Match(cefilter.Attrs{}) {
		t.Error("expected no match when priority absent")
	}
	if !pred.Match(cefilter.Attrs{"priority": "high"}) {
		t.Error("expected match when priority present")
	}
}

func TestParse_NotExistsAndComposed(t *testing.T) {
	expr := `(type = "com.example.order" OR type LIKE "com.acme.*") AND orgid = "org-123" AND NOT EXISTS debug`
	pred, err := cefilter.Parse(expr)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	pass := cefilter.Attrs{
		"type":  "com.acme.order",
		"orgid": "org-123",
		// debug absent — NOT EXISTS should pass
	}
	if !pred.Match(pass) {
		t.Errorf("expected match, expr: %s", pred)
	}

	withDebug := cefilter.Attrs{
		"type":  "com.acme.order",
		"orgid": "org-123",
		"debug": "true",
	}
	if pred.Match(withDebug) {
		t.Error("expected no match when debug present")
	}
}

func TestParse_Ne(t *testing.T) {
	pred, _ := cefilter.Parse(`subject != "internal"`)
	if !pred.Match(cefilter.Attrs{"subject": "public"}) {
		t.Error("expected match for non-internal subject")
	}
	if pred.Match(cefilter.Attrs{"subject": "internal"}) {
		t.Error("expected no match for internal subject")
	}
	// missing attribute — != should pass (attribute not present, so ≠ "internal")
	if !pred.Match(cefilter.Attrs{}) {
		t.Error("expected match when subject absent")
	}
}

func TestFilterSet_MatchRouting(t *testing.T) {
	var fs cefilter.FilterSet
	// Imagine 3 orgs, each with different filter criteria
	mustAdd := func(name, expr string) {
		if err := fs.Add(contracts.FilterEntry{Name: name}, expr); err != nil {
			t.Fatalf("Add(%q): %v", name, err)
		}
	}
	mustAdd("webhook-org1", `type = "com.example.order" AND source LIKE "checkout/*" AND orgid = "org-1"`)
	mustAdd("webhook-org2", `type LIKE "com.example.*" AND orgid = "org-2"`)
	mustAdd("webhook-org3", `type LIKE "com.*.order" AND region IN ("us-east", "eu-west") AND orgid = "org-3"`)

	event := cefilter.Attrs{
		"type":   "com.example.order",
		"source": "checkout/web",
		"orgid":  "org-1",
		"region": "us-east",
	}
	matched := fs.Match(event)
	fmt.Printf("matched webhooks: %v\n", matched)
	// org-1 matches its exact filter, org-3 also matches (com.*.order + us-east)
	if len(matched) == 0 {
		t.Error("expected at least one match")
	}
}

func TestFromCloudEvent(t *testing.T) {
	pred, _ := cefilter.Parse(`type = "com.example.order" AND orgid = "org-123"`)

	event := mockEvent{
		typ:    "com.example.order",
		source: "checkout",
		id:     "abc-123",
		exts:   map[string]any{"orgid": "org-123"},
	}
	attrs := cefilter.FromCloudEvent(event)
	if !pred.Match(attrs) {
		t.Errorf("expected match from CloudEvent attrs: %v", attrs)
	}
}
