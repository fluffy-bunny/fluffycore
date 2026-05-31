package contracts

// FilterEntry is the metadata attached to every registered filter.
// Name uniquely identifies the filter within a set.
// Hint is an opaque caller-defined routing instruction — examples:
//   - a NATS subject to publish the event to
//   - a gRPC handler tag or service endpoint
//   - a Kafka topic, SQS queue URL, or webhook URL
//
// The cefilter engine stores and returns it unchanged; the caller decides
// what it means and where to send the event.
// Metadata holds additional routing annotations that do not fit in Hint.
type FilterEntry struct {
	Name     string
	Hint     string
	Metadata map[string]string
}

// IFilterSet is the interface satisfied by both FilterSet and IndexedFilterSet.
// Callers may use this interface to swap implementations without changing routing logic.
type IFilterSet interface {
	// Add compiles expr and registers it under entry.
	// Returns an error if expr is invalid — fail at startup, not per-event.
	Add(entry FilterEntry, expr string) error

	// Remove unregisters the filter with the given Name.
	// No-op if the name does not exist.
	Remove(name string)

	// Match evaluates all filters against attrs and returns every entry that matched.
	// Use this for fan-out: each returned entry's Hint tells you where to deliver the event.
	Match(attrs map[string]string) []FilterEntry

	// MatchEndpoints returns the distinct, non-empty Hint values of all matching filters.
	// If multiple filter rules share the same Hint (endpoint), that endpoint appears once.
	// Use this for webhook fan-out: iterate the result and POST to each URL exactly once.
	// Respects MaxMatches (treated as max distinct endpoints to dispatch to).
	MatchEndpoints(attrs map[string]string) []string

	// MatchAny returns true as soon as the first filter matches (short-circuits).
	// Use this for ingress gating: drop events no subscriber cares about.
	MatchAny(attrs map[string]string) bool
}
