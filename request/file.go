package request

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"

	"github.com/swaggest/rest"
	"github.com/valyala/fasthttp"
)

var (
	multipartFileType        = reflect.TypeOf((*multipart.File)(nil)).Elem()
	multipartFilesType       = reflect.TypeOf(([]multipart.File)(nil))
	multipartFileHeaderType  = reflect.TypeOf((*multipart.FileHeader)(nil))
	multipartFileHeadersType = reflect.TypeOf(([]*multipart.FileHeader)(nil))
)

func decodeFiles(rc *fasthttp.RequestCtx, input interface{}, _ rest.Validator) error {
	v := reflect.ValueOf(input)
	return decodeFilesInStruct(rc, v)
}

func decodeFilesInStruct(rc *fasthttp.RequestCtx, v reflect.Value) error {
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
			err := reflectFile(rc, field, v.Field(i))
			if err != nil {
				return err
			}

			continue
		}

		if field.Anonymous {
			if err := decodeFilesInStruct(rc, v.Field(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

func reflectFile(rc *fasthttp.RequestCtx, field reflect.StructField, v reflect.Value) error {
	name := ""
	if tag := field.Tag.Get("file"); tag != "" && tag != "-" {
		name = tag
	} else if tag := field.Tag.Get("formData"); tag != "" && tag != "-" {
		name = tag
	}

	if name != "" {
		header, err := rc.FormFile(name)
		if err != nil {
			if err == http.ErrMissingFile {
				if field.Tag.Get("required") == "true" {
					return fmt.Errorf("required file is missing: %q", name)
				}
			}

			return fmt.Errorf("failed to get file %q from request: %w", name, err)
		}

		if field.Type == multipartFileType {
			file, err := header.Open()
			if err != nil {
				return fmt.Errorf("failed to open file %q from request: %w", name, err)
			}

			v.Set(reflect.ValueOf(file))
		}

		if field.Type == multipartFileHeaderType {
			v.Set(reflect.ValueOf(header))
		}
	}

	return nil
}
