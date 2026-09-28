// Package valuecodec converts values to and from the plain strings stored in cookies and
// localStorage. Primitives (and named types over them) are stored as their text form;
// everything else is left to the caller (JSON).
package valuecodec

import (
	"fmt"
	"reflect"
	"strconv"
)

// Deref returns what a (possibly nil) pointer points to. ok is false for a nil pointer.
// Encoders must work on the dereferenced value as an interface{}: asserting it back to T
// (as the old code did) panics whenever T is itself a pointer type.
func Deref(value any) (v any, ok bool) {
	rv := reflect.ValueOf(value)
	for rv.IsValid() && rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil, false
	}
	return rv.Interface(), true
}

// EncodePrimitive returns v's text form if v is a string, bool or number (including named
// types over those). handled is false for anything else.
func EncodePrimitive(v any) (s string, handled bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), true
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 64), true
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool()), true
	}
	return "", false
}

// DecodePrimitive parses s into a new T when T (or what T points to) is a string, bool or
// number kind. handled is false for anything else. Values are built with reflect so named
// types (type Color string) and pointers to them work -- the old code type-asserted a
// plain string to T, which panics for both.
func DecodePrimitive[T any](s string) (result T, handled bool, err error) {
	rt := reflect.TypeOf(result)
	if rt == nil {
		return result, false, nil
	}
	isPtr := rt.Kind() == reflect.Ptr
	if isPtr {
		rt = rt.Elem()
	}
	v := reflect.New(rt).Elem()
	switch rt.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, rt.Bits())
		if err != nil {
			return result, true, err
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, rt.Bits())
		if err != nil {
			return result, true, err
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, rt.Bits())
		if err != nil {
			return result, true, err
		}
		v.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return result, true, err
		}
		v.SetBool(b)
	default:
		return result, false, nil
	}
	if isPtr {
		p := reflect.New(rt)
		p.Elem().Set(v)
		return p.Interface().(T), true, nil
	}
	out, ok := v.Interface().(T)
	if !ok {
		return result, true, fmt.Errorf("valuecodec: cannot convert to %T", result)
	}
	return out, true, nil
}

// NewTarget returns a pointer suitable for json.Unmarshal into T, and a function that
// returns the decoded T. For a pointer T it allocates the pointee, so the result is a
// non-nil pointer.
func NewTarget[T any]() (target any, get func() T) {
	var zero T
	rt := reflect.TypeOf(zero)
	if rt != nil && rt.Kind() == reflect.Ptr {
		p := reflect.New(rt.Elem())
		return p.Interface(), func() T { return p.Interface().(T) }
	}
	var result T
	return &result, func() T { return result }
}
