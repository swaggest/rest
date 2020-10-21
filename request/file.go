package request

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"

	"github.com/swaggest/rest"
)

var (
	multipartFileType       = reflect.TypeOf((*multipart.File)(nil)).Elem()
	multipartFileHeaderType = reflect.TypeOf((*multipart.FileHeader)(nil))
)

func decodeFiles(r *http.Request, input interface{}, _ rest.Validator) error {
	v := reflect.ValueOf(input)

	return decodeFilesInStruct(r, v)
}

func decodeFilesInStruct(r *http.Request, v reflect.Value) error {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Type == multipartFileType || field.Type == multipartFileHeaderType {
			err := setFile(r, field, v.Field(i))
			if err != nil {
				return err
			}

			continue
		}

		if field.Anonymous {
			if err := decodeFilesInStruct(r, v.Field(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

func setFile(r *http.Request, field reflect.StructField, v reflect.Value) error {
	name := ""
	if tag := field.Tag.Get(fileTag); tag != "" && tag != "-" {
		name = tag
	} else if tag := field.Tag.Get(formDataTag); tag != "" && tag != "-" {
		name = tag
	}

	if name == "" {
		return nil
	}

	file, header, err := r.FormFile(name)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			if field.Tag.Get("required") == "true" {
				return fmt.Errorf("%w: %q", ErrMissingRequiredFile, name)
			}
		}

		return fmt.Errorf("failed to get file %q from request: %w", name, err)
	}

	if field.Type == multipartFileType {
		v.Set(reflect.ValueOf(file))
	}

	if field.Type == multipartFileHeaderType {
		v.Set(reflect.ValueOf(header))
	}

	return nil
}
