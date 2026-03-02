package reflect

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasReturnType_ErrorLastReturn(t *testing.T) {
	fn := func() error { return nil }
	require.True(t, HasReturnType[error](fn))
}

func TestHasReturnType_ErrorNotLast(t *testing.T) {
	fn := func() (error, int) { return nil, 0 }
	require.False(t, HasReturnType[error](fn))
}

func TestHasReturnType_StringReturn(t *testing.T) {
	fn := func() string { return "" }
	require.True(t, HasReturnType[string](fn))
}

func TestHasReturnType_InterfaceAssignable(t *testing.T) {
	fn := func() *os.File { return nil }
	require.True(t, HasReturnType[io.Reader](fn))
}

func TestHasReturnType_NoReturnValues(t *testing.T) {
	fn := func() {}
	require.False(t, HasReturnType[error](fn))
}

func TestHasReturnType_NotAFunction(t *testing.T) {
	require.False(t, HasReturnType[error]("not a function"))
}

func TestHasReturnType_MultipleReturns_LastIsError(t *testing.T) {
	fn := func() (int, error) { return 0, nil }
	require.True(t, HasReturnType[error](fn))
}

func TestHasReturnType_TypeMismatch(t *testing.T) {
	fn := func() int { return 0 }
	require.False(t, HasReturnType[string](fn))
}

func TestHasReturnType_NilValue(t *testing.T) {
	// nil causes a panic in reflect.TypeOf — verify the function panics
	require.Panics(t, func() {
		HasReturnType[error](nil)
	})
}

func TestHasReturnType_MultiReturn_LastIsString(t *testing.T) {
	fn := func() (int, bool, string) { return 0, false, "" }
	require.True(t, HasReturnType[string](fn))
	require.False(t, HasReturnType[int](fn))
	require.False(t, HasReturnType[bool](fn))
}
