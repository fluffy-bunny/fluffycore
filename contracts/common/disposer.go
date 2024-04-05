package common

import "context"

// IDispose is implemented by components that support stopping and cleaning up
// their underlying resources.
type IDispose interface {
	// Close the component, blocks until either the underlying resources are
	// cleaned up or the context is cancelled. Returns an error if the context
	// is cancelled.
	Dispose(ctx context.Context) error
}
