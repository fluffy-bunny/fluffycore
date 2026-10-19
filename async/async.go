package async

import (
	"fmt"

	"github.com/reugn/async"
)

type (
	// AsyncResponse ...
	AsyncResponse struct {
		Message string
		Error   error
	}
	// PromiseResponse ...
	PromiseResponse[T any] struct {
		Future async.Future[T]
		done   chan struct{}
	}
)

// IsComplete checks to see if a promise has been completed
func (s *PromiseResponse[T]) IsComplete() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// FutureResponse ...
type FutureResponse struct {
	Err   error
	Value interface{}
}

// ExecuteAsync returns a response that contains a future and a helper method to check if the future has been completed
func ExecuteAsync(f func() (interface{}, error)) *PromiseResponse[*FutureResponse] {
	p := async.NewPromise[*FutureResponse]()
	// done is CLOSED once the result is set. The Map callback below used to SEND on this
	// unbuffered channel, so the future only completed if someone happened to be calling
	// IsComplete at that moment: Join() on its own deadlocked and leaked the goroutine, and
	// IsComplete consumed the single signal (a second call said false again).
	done := make(chan struct{})
	go func() {
		var response *FutureResponse
		defer func() {
			if r := recover(); r != nil {
				response = &FutureResponse{Value: nil, Err: fmt.Errorf("panic: %v", r)}
			}
			p.Success(response)
			close(done)
		}()
		value, err := f()
		response = &FutureResponse{Value: value, Err: err}
	}()
	future := p.Future().Map(func(v *FutureResponse) (*FutureResponse, error) {
		return v, v.Err
	})
	return &PromiseResponse[*FutureResponse]{
		Future: future,
		done:   done,
	}
}

// ExecuteWithPromiseAsync ...
func ExecuteWithPromiseAsync[T any](fn func(async.Promise[T])) async.Future[T] {
	promise := async.NewPromise[T]()
	go fn(promise)
	return promise.Future()
}
