package cefilter

import cloudevents "github.com/cloudevents/sdk-go/v2"

// FromSDKEvent converts a cloudevents.Event directly to the flat Attrs map
// without requiring any adapter boilerplate.
//
// Usage:
//
//	pred, _ := cefilter.Parse(`type = "com.example.order" AND orgid = "org-123"`)
//	attrs   := cefilter.FromSDKEvent(event)
//	if pred.Match(attrs) { ... }
//
// cloudevents.Event satisfies CloudEventAccessor, so you can also pass it
// directly to FromCloudEvent if you prefer the interface-based path.
func FromSDKEvent(e cloudevents.Event) Attrs {
	return FromCloudEvent(e)
}
