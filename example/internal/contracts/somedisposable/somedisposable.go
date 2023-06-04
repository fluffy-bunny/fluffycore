package somedisposable

import (
	"context"

	di "github.com/dozm/di"
)

type (
	IScopedSomeDisposable interface {
		di.Disposable
		DoSomething(ctx context.Context) (string, error)
	}
)
