package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/swaggest/rest"
	"github.com/valyala/fasthttp"
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

func decodeJSONBody(readJSON func(rd io.Reader, v interface{}) error) valueDecoderFunc {
	return func(rc *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error {
		if rc.Request.Header.ContentLength() == 0 {
			return errors.New("missing request body to decode json")
		}

		contentType := rc.Request.Header.ContentType()
		if len(contentType) > 0 {
			if len(contentType) < 16 || !bytes.Equal(contentType[0:16], []byte("application/json")) { // allow 'application/json;charset=UTF-8'
				return fmt.Errorf("%w, received: %s", ErrJSONExpected, contentType)
			}
		}

		b := rc.Request.Body()

		validate := validator != nil && validator.HasConstraints(rest.ParamInBody)

		rd := bytes.NewReader(b)
		err := readJSON(rd, &input)
		if err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}

		if validator != nil && validate {
			err = validator.ValidateJSONBody(b)
			if err != nil {
				return err
			}
		}

		return nil
	}
}
