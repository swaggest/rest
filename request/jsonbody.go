package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	rest2 "github.com/swaggest/rest"
	"io"

	"github.com/valyala/fasthttp"
)

func readJSON(rd io.Reader, v interface{}) error {
	d := json.NewDecoder(rd)

	return d.Decode(v)
}

func decodeJSONBody(readJSON func(rd io.Reader, v interface{}) error) valueDecoderFunc {
	return func(rc *fasthttp.RequestCtx, input interface{}, validator rest2.Validator) error {
		if len(rc.Request.Body()) == 0 {
			return errors.New("missing request body to decode json")
		}

		contentType := rc.Request.Header.ContentType()
		if len(contentType) > 0 {
			if len(contentType) < 16 || !bytes.Equal(contentType[0:16], []byte("application/json")) { // allow 'application/json;charset=UTF-8'
				return fmt.Errorf("%w, received: %s", ErrJSONExpected, contentType)
			}
		}

		b := rc.Request.Body()

		validate := validator != nil && validator.HasConstraints(rest2.ParamInBody)

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
