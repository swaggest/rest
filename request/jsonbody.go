package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/swaggest/rest"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

func readJSON(rd io.Reader, v interface{}) error {
	d := json.NewDecoder(rd)

	return d.Decode(v)
}

func decodeJSONBody(readJSON func(rd io.Reader, v interface{}) error, tolerateFormData bool) valueDecoderFunc {
	return func(r *http.Request, input interface{}, validator rest.Validator) error {
		if r.ContentLength == 0 {
			return ErrMissingRequestBody
		}

		if ret, err := checkJSONBodyContentType(r.Header.Get("Content-Type"), tolerateFormData); err != nil {
			return err
		} else if ret {
			return nil
		}

		var b *bytes.Buffer

		b = bufPool.Get().(*bytes.Buffer) //nolint:errcheck // bufPool is configured to provide *bytes.Buffer.
		defer bufPool.Put(b)
		b.Reset()

		// Read body into buffer.
		if _, err := b.ReadFrom(r.Body); err != nil {
			return err
		}

		validate := validator != nil && validator.HasConstraints(rest.ParamInBody)

		if validate {
			// Perform validation before unmarshalling into input object.
			err := validator.ValidateJSONBody(b.Bytes())
			if err != nil {
				return err
			}
		}

		return readJSON(b, input)
	}
}

func checkJSONBodyContentType(contentType string, tolerateFormData bool) (ret bool, err error) {
	if contentType == "" {
		return false, nil
	}

	if len(contentType) < 16 || strings.ToLower(contentType[0:16]) != "application/json" { // allow 'application/json;charset=UTF-8'
		if tolerateFormData && (contentType == "application/x-www-form-urlencoded" || contentType == "multipart/form-data") {
			return true, nil
		}

		return true, fmt.Errorf("%w, received: %s", ErrJSONExpected, contentType)
	}

	return false, nil
}
