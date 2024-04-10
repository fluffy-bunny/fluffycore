package common

import "context"

// Closer is implemented by components that support stopping and cleaning up
// their underlying resources.
type ICloser interface {
	// Close the component, blocks until either the underlying resources are
	// cleaned up or the context is cancelled. Returns an error if the context
	// is cancelled.
	Close(ctx context.Context) error
}
