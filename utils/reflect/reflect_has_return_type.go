package reflect

import (
	"reflect"
)

func HasReturnType[T any](fn any) bool {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		// Not a function type
		return false
	}

	// Get the last output parameter (if any)
	numOut := fnType.NumOut()
	if numOut == 0 {
		// No return values
		return false
	}

	lastOutType := fnType.Out(numOut - 1)
	targetType := reflect.TypeOf((*T)(nil)).Elem()

	return lastOutType.AssignableTo(targetType)
}
