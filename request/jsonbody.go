package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/swaggest/rest"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

func decodeJSONBody(r *http.Request, input interface{}, validator rest.Validator) error {
	if r.ContentLength == 0 {
		return ErrMissingRequestBody
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "" {
		if len(contentType) < 16 || contentType[0:16] != "application/json" { // allow 'application/json;charset=UTF-8'
			return fmt.Errorf("%w, received: %s", ErrJSONExpected, contentType)
		}
	}

	b := bufPool.Get().(*bytes.Buffer) // nolint:errcheck // bufPool is configured to provide *bytes.Buffer.
	defer bufPool.Put(b)

	b.Reset()

	rd := io.TeeReader(r.Body, b)
	d := json.NewDecoder(rd)

	err := d.Decode(&input)
	if err != nil {
		return fmt.Errorf("failed to decode json: %w", err)
	}

	if validator != nil {
		err = validator.ValidateJSONBody(b.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}
