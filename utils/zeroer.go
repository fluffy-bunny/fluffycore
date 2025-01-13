package utils

import "reflect"

// Zeroer :
type Zeroer interface {
	IsZero() bool
}

/*
IsZero : determine is zero value
1. string => empty string
2. bool => false
3. function => nil
4. map => nil (uninitialized map)
5. slice => nil (uninitialized slice)
*/
func IsZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	it := v.Interface()
	x, ok := it.(Zeroer)
	if ok {
		return x.IsZero()
	}

	switch v.Kind() {
	case reflect.Interface, reflect.Func:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.Array:
		if v.Len() == 0 {
			return true
		}
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && IsZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && IsZero(v.Field(i))
		}
		return z
	}

	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return it == z.Interface()
}

// IsEmptyOrNil checks if a value is empty or nil, useful for strings and arrays
func _isEmptyOrNil(i interface{}) bool {
	if _isNil(i) {
		return true
	}

	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		if reflect.ValueOf(i).IsNil() {
			return true
		}
		return IsZero(reflect.ValueOf(i))
	case reflect.String:
		return i == ""
	}
	return false //everything else here is a primitive
}

// define an alias for IsEmptyOrNil
var IsNilOrEmpty = _isEmptyOrNil
var IsEmptyOrNil = _isEmptyOrNil

// InNotEmptyOrNil checks if a value is not empty or nil, useful for strings and arrays
func _isNotEmptyOrNil(i interface{}) bool {
	return !IsEmptyOrNil(i)
}

var IsNotEmptyOrNil = _isNotEmptyOrNil
var IsNotNilOrEmpty = _isNotEmptyOrNil

// IsNil is a wholistic nil checker
func _isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	case reflect.Func:
		return reflect.ValueOf(i).IsNil()
	}
	return false //everything else here is a primitive
}

// IsNotNil is a wholistic nil checker
func _isNotNil(i interface{}) bool {
	return !IsNil(i)
}

var IsNil = _isNil
var IsNotNil = _isNotNil
