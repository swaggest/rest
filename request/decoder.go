package request

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/swaggest/form/v5"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/valyala/fasthttp"
)

type (
	// Loader loads data from http.Request.
	//
	// Implement this interface on a pointer to your input structure to disable automatic request mapping.
	Loader interface {
		LoadFromFastHTTPRequest(rc *fasthttp.RequestCtx) error
	}

	decoderFunc      func(rc *fasthttp.RequestCtx, v map[string]string) error
	valueDecoderFunc func(rc *fasthttp.RequestCtx, v interface{}, validator rest.Validator) error
)

func b2s(b []byte) string {
	p := unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&b)).Data)

	var s string
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	hdr.Data = uintptr(p)
	hdr.Len = len(b)

	return s
}

func decodeValidate(d *form.Decoder, v interface{}, p map[string]string, in rest.ParamIn, val rest.Validator) error {
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
		return make(map[string]string)
	},
}

func makeDecoder(in rest.ParamIn, formDecoder *form.Decoder, decoderFunc decoderFunc) valueDecoderFunc {
	return func(rc *fasthttp.RequestCtx, v interface{}, validator rest.Validator) error {
		values := valuesPool.Get().(map[string]string) // nolint:errcheck
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
	in       []rest.ParamIn
}

var _ nethttp.RequestDecoder = &decoder{}

// Decode populates and validates input with data from http request.
func (d *decoder) Decode(rc *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error {
	if i, ok := input.(Loader); ok {
		return i.LoadFromFastHTTPRequest(rc)
	}

	for i, decode := range d.decoders {
		err := decode(rc, input, validator)
		if err != nil {
			// nolint:errorlint // Error is not wrapped, type assertion is more performant.
			if de, ok := err.(form.DecodeErrors); ok {
				errs := make(rest.RequestErrors, len(de))
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

func formDataToURLValues(rc *fasthttp.RequestCtx, params map[string]string) error {
	args := rc.Request.PostArgs()

	if args.Len() == 0 {
		return nil
	}

	args.VisitAll(func(key, value []byte) {
		if params == nil {
			params = make(map[string]string, 1)
		}

		params[b2s(key)] = b2s(value)
	})

	return nil
}

func headerToURLValues(rc *fasthttp.RequestCtx, params map[string]string) error {
	rc.Request.Header.VisitAll(func(key, value []byte) {
		//if params == nil {
		//	params = make(map[string]string, 1)
		//}

		params[b2s(key)] = b2s(value)
	})

	return nil
}

func queryToURLValues(rc *fasthttp.RequestCtx, params map[string]string) error {
	rc.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		//if params == nil {
		//	params = make(map[string]string, 1)
		//}

		params[b2s(key)] = b2s(value)
	})

	return nil
}

func cookiesToURLValues(rc *fasthttp.RequestCtx, params map[string]string) error {
	rc.Request.Header.VisitAllCookie(func(key, value []byte) {
		if params == nil {
			params = make(map[string]string, 1)
		}

		params[b2s(key)] = b2s(value)
	})

	return nil
}
