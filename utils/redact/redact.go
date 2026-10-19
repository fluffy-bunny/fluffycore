package utils

import (
	"fmt"
	"reflect"

	"github.com/fluffy-bunny/fluffycore/utils"
	"github.com/jinzhu/copier"
)

const RedactedString = "[[REDACTED]]"

// PrettyPrintRedacted ...
func PrettyPrintRedacted(source, dest interface{}) {
	MakeRedactedCopy(source, dest)
	fmt.Println(utils.PrettyJSON(dest))
}

// clone returns a new zero value of oldObj's type; MakeRedactedCopy then deep-copies into
// it. It used to shallow-copy the fields first, so dest shared src's pointers and the deep
// copy wrote into (then Redact rewrote) the source's own nested structs.
func clone(oldObj interface{}) interface{} {
	return reflect.New(reflect.TypeOf(oldObj).Elem()).Interface()
}

// CloneAndRedact logs a redacted version of the object
func CloneAndRedact(source interface{}) (dest interface{}, err error) {
	dest = clone(source)
	err = MakeRedactedCopy(source, dest)
	return dest, err
}

// MakeRedactedCopy ...
func MakeRedactedCopy(src, dst interface{}) error {
	// DeepCopy: a shallow copy shares pointers and slices with src, so Redact walking into
	// them overwrote the ORIGINAL config's secrets -- e.g. after logging a redacted copy of
	// the config, a nested *OTELConfig's API key was "[[REDACTED]]" in the live config.
	err := copier.CopyWithOption(dst, src, copier.Option{DeepCopy: true})
	if err != nil {
		return err
	}
	Redact(reflect.ValueOf(dst))
	return nil
}

// Redact ...
func Redact(reflectValue reflect.Value) {
	reflectValue = indirect(reflectValue)

	if reflectValue.Kind() == reflect.Struct {
		l := reflectValue.NumField()
		for i := 0; i < l; i++ {
			field := indirect(reflectValue.Field(i))
			fieldKind := field.Kind()

			if fieldKind == reflect.Struct || fieldKind == reflect.Slice {
				Redact(field)
				continue
			}

			tt := reflectValue.Type().Field(i)
			if hasTag(tt, "redact") {
				if fieldKind == reflect.String {
					field.SetString(redactString(field.String()))
				}
			}
		}
		return
	}

	if reflectValue.Kind() == reflect.Slice {
		l := reflectValue.Len()
		for i := 0; i < l; i++ {
			Redact(reflectValue.Index(i))
		}
		return
	}
}
func redactString(s string) string {
	if len(s) >= 8 {
		return s[:4] + RedactedString + s[len(s)-2:]
	} else if len(s) >= 6 {
		return s[:2] + RedactedString
	}
	return RedactedString
}
func hasTag(val reflect.StructField, tag string) bool {
	c := val.Tag
	g := c.Get(tag)
	return len(g) > 0
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
}
func indirectType(v reflect.Type) reflect.Type {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice {
		return v.Elem()
	}
	return v
}
