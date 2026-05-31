package cefilter_test

import (
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/fluffy-bunny/fluffycore/cloudevent/cefilter"
)

func TestFromSDKEvent(t *testing.T) {
	pred, err := cefilter.Parse(`type = "com.example.order" AND orgid = "org-123"`)
	if err != nil {
		t.Fatal(err)
	}

	e := cloudevents.NewEvent()
	e.SetType("com.example.order")
	e.SetSource("checkout/web")
	e.SetID("evt-001")
	e.SetExtension("orgid", "org-123")

	attrs := cefilter.FromSDKEvent(e)
	if !pred.Match(attrs) {
		t.Errorf("expected match, attrs=%v", attrs)
	}

	// Wrong org — should not match
	e2 := cloudevents.NewEvent()
	e2.SetType("com.example.order")
	e2.SetSource("checkout/web")
	e2.SetID("evt-002")
	e2.SetExtension("orgid", "org-999")

	if pred.Match(cefilter.FromSDKEvent(e2)) {
		t.Error("expected no match for wrong orgid")
	}
}

func TestFromSDKEvent_LikeAndExists(t *testing.T) {
	pred, err := cefilter.Parse(`source LIKE "checkout/*" AND EXISTS priority`)
	if err != nil {
		t.Fatal(err)
	}

	e := cloudevents.NewEvent()
	e.SetType("com.example.order")
	e.SetSource("checkout/mobile")
	e.SetID("evt-003")
	e.SetExtension("priority", "high")

	if !pred.Match(cefilter.FromSDKEvent(e)) {
		t.Error("expected match")
	}

	// Missing extension
	e2 := cloudevents.NewEvent()
	e2.SetType("com.example.order")
	e2.SetSource("checkout/mobile")
	e2.SetID("evt-004")
	// no priority extension

	if pred.Match(cefilter.FromSDKEvent(e2)) {
		t.Error("expected no match when priority absent")
	}
}
