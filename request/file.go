package request

import (
	"errors"
	"fmt"
	rest2 "github.com/swaggest/rest"
	"mime/multipart"
	"net/http"
	"reflect"

	"github.com/valyala/fasthttp"
)

var (
	multipartFileType        = reflect.TypeOf((*multipart.File)(nil)).Elem()
	multipartFilesType       = reflect.TypeOf(([]multipart.File)(nil))
	multipartFileHeaderType  = reflect.TypeOf((*multipart.FileHeader)(nil))
	multipartFileHeadersType = reflect.TypeOf(([]*multipart.FileHeader)(nil))
)

func decodeFiles(rc *fasthttp.RequestCtx, input interface{}, _ rest2.Validator) error {
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

		if field.Type == multipartFileType || field.Type == multipartFileHeaderType ||
			field.Type == multipartFilesType || field.Type == multipartFileHeadersType {
			err := setFile(rc, field, v.Field(i))
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

// nolint:funlen // Maybe later.
func setFile(rc *fasthttp.RequestCtx, field reflect.StructField, v reflect.Value) error {
	name := ""
	if tag := field.Tag.Get(fileTag); tag != "" && tag != "-" {
		name = tag
	} else if tag := field.Tag.Get(formDataTag); tag != "" && tag != "-" {
		name = tag
	}

	if name == "" {
		return nil
	}

	header, err := rc.FormFile(name)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			if field.Tag.Get("required") == "true" {
				return fmt.Errorf("%w: %q", ErrMissingRequiredFile, name)
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

	if field.Type == multipartFilesType {
		mf, err := rc.MultipartForm()
		if err != nil {
			return fmt.Errorf("failed to get multipart form from request: %w", err)
		}

		res := make([]multipart.File, 0, len(mf.File[name]))

		for _, h := range mf.File[name] {
			f, err := h.Open()
			if err != nil {
				return fmt.Errorf("failed to open uploaded file %s (%s): %w", name, h.Filename, err)
			}

			res = append(res, f)
		}

		v.Set(reflect.ValueOf(res))
	}

	if field.Type == multipartFileHeadersType {
		mf, err := rc.MultipartForm()
		if err != nil {
			return fmt.Errorf("failed to get multipart form from request: %w", err)
		}

		v.Set(reflect.ValueOf(mf.File[name]))
	}

	return nil
}
