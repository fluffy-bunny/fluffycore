package utils

import (
	"reflect"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
)

const MaxDepth = 32

// ReplaceMergeMap merges src into dst, replacing values for existing keys recursively up to MaxDepth.
func ReplaceMergeMap(dst, src map[string]interface{}) error {
	return replaceOnlyMergeMap(dst, src, 0)
}

func replaceOnlyMergeMap(dst, src map[string]interface{}, depth int) error {
	if depth > MaxDepth {
		return status.Error(codes.InvalidArgument, "merge recursion max depth exceeded")
	}
	if dst == nil {
		return status.Error(codes.InvalidArgument, "dst map is nil")
	}
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			srcMap, srcMapOk := mapify(srcVal)
			dstMap, dstMapOk := mapify(dstVal)
			if srcMapOk && dstMapOk {
				err := replaceOnlyMergeMap(dstMap, srcMap, depth+1)
				if err != nil {
					return err
				}
				srcVal = dstMap
			}
			// Otherwise src replaces dst. (This used to `continue`, keeping dst's value.)
		}
		dst[key] = srcVal
	}
	return nil
}
func mapify(i interface{}) (map[string]interface{}, bool) {
	value := reflect.ValueOf(i)
	// Only string-keyed maps: other key kinds would be stringified to junk like "<int Value>".
	if value.Kind() == reflect.Map && value.Type().Key().Kind() == reflect.String {
		m := map[string]interface{}{}
		for _, k := range value.MapKeys() {
			m[k.String()] = value.MapIndex(k).Interface()
		}
		return m, true
	}
	return nil, false
}
