package probe

import "context"

type (
	// IProbe ...
	IProbe interface {
		GetName() string
		Probe(ctx context.Context) error
	}
)
