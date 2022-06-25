package request

import (
	"net/url"
	"sync"

	"github.com/swaggest/form/v5"
	rest2 "github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/valyala/fasthttp"
)

type (
	// Loader loads data from http.Request.
	//
	// Implement this interface on a pointer to your input structure to disable automatic request mapping.
	Loader interface {
		LoadFromFastHTTPRequest(rc *fasthttp.RequestCtx) error
	}

	decoderFunc      func(rc *fasthttp.RequestCtx, v url.Values) error
	valueDecoderFunc func(rc *fasthttp.RequestCtx, v interface{}, validator rest2.Validator) error
)

func decodeValidate(d *form.Decoder, v interface{}, p url.Values, in rest2.ParamIn, val rest2.Validator) error {
	goValues := make(map[string]interface{}, len(p))

	err := d.Decode(v, p, goValues)
	if err != nil {
		return err
	}

	if len(p) > len(goValues) {
		for k := range p {
			if _, exists := goValues[k]; !exists {
				pk := p[k]
				switch len(pk) {
				case 0:
					goValues[k] = nil
				case 1:
					goValues[k] = p[k][0]
				default:
					goValues[k] = p[k]
				}
			}
		}
	}

	return val.ValidateData(in, goValues)
}

var valuesPool = &sync.Pool{
	New: func() interface{} {
		return make(url.Values)
	},
}

func makeDecoder(in rest2.ParamIn, formDecoder *form.Decoder, decoderFunc decoderFunc) valueDecoderFunc {
	return func(rc *fasthttp.RequestCtx, v interface{}, validator rest2.Validator) error {
		values := valuesPool.Get().(url.Values) // nolint:errcheck
		for k := range values {
			delete(values, k)
		}

		defer func() {
			valuesPool.Put(values)
		}()

		err := decoderFunc(rc, values)
		if err != nil {
			return err
		}

		if validator != nil {
			return decodeValidate(formDecoder, v, values, in, validator)
		}

		return formDecoder.Decode(v, values)
	}
}

// decoder extracts Go value from *http.Request.
type decoder struct {
	decoders []valueDecoderFunc
	in       []rest2.ParamIn
}

var _ fhttp.RequestDecoder = &decoder{}

// Decode populates and validates input with data from http request.
func (d *decoder) Decode(rc *fasthttp.RequestCtx, input interface{}, validator rest2.Validator) error {
	if i, ok := input.(Loader); ok {
		return i.LoadFromFastHTTPRequest(rc)
	}

	for i, decode := range d.decoders {
		err := decode(rc, input, validator)
		if err != nil {
			// nolint:errorlint // Error is not wrapped, type assertion is more performant.
			if de, ok := err.(form.DecodeErrors); ok {
				errs := make(rest2.RequestErrors, len(de))
				for name, e := range de {
					errs[string(d.in[i])+":"+name] = []string{"#: " + e.Error()}
				}

				return errs
			}

			return err
		}
	}

	return nil
}

func formDataToURLValues(rc *fasthttp.RequestCtx, params url.Values) error {
	args := rc.Request.PostArgs()

	if args.Len() == 0 {
		return nil
	}

	args.VisitAll(func(key, value []byte) {
		k := b2s(key)
		v := params[k]
		params[k] = append(v, b2s(value))
	})

	return nil
}

func headerToURLValues(rc *fasthttp.RequestCtx, params url.Values) error {
	rc.Request.Header.VisitAll(func(key, value []byte) {
		k := b2s(key)
		v := params[k]
		params[k] = append(v, b2s(value))
	})

	return nil
}

func queryToURLValues(rc *fasthttp.RequestCtx, params url.Values) error {
	rc.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		k := b2s(key)
		v := params[k]
		params[k] = append(v, b2s(value))
	})

	return nil
}

func cookiesToURLValues(rc *fasthttp.RequestCtx, params url.Values) error {
	rc.Request.Header.VisitAllCookie(func(key, value []byte) {
		k := b2s(key)
		v := params[k]
		params[k] = append(v, b2s(value))
	})

	return nil
}
