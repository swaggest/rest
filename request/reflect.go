package request

import (
	"reflect"

	"github.com/swaggest/refl"
)

// HasFileFields checks if the structure has fields to receive uploaded files.
func hasFileFields(i interface{}, tagname string) bool {
	found := false

	refl.WalkTaggedFields(reflect.ValueOf(i), func(_ reflect.Value, sf reflect.StructField, _ string) {
		if sf.Type == multipartFileType || sf.Type == multipartFileHeaderType ||
			sf.Type == multipartFilesType || sf.Type == multipartFileHeadersType {
			found = true

			return
		}
	}, tagname)

	return found
}
