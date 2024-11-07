package request

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/swaggest/form/v5"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
)

type (
	// Loader loads data from http.Request.
	//
	// Implement this interface on a pointer to your input structure to disable automatic request mapping.
	Loader interface {
		LoadFromHTTPRequest(r *http.Request) error
	}

	// Setter captures original http.Request.
	//
	// Implement this interface on a pointer to your input structure to get access to http.Request.
	Setter interface {
		SetRequest(r *http.Request)
	}

	decoderFunc      func(r *http.Request) (url.Values, error)
	valueDecoderFunc func(r *http.Request, v interface{}, validator rest.Validator) error
)

// EmbeddedSetter can capture *http.Resuest in your input structure.
type EmbeddedSetter struct {
	r *http.Request
}

// SetRequest implements Setter.
func (e *EmbeddedSetter) SetRequest(r *http.Request) {
	e.r = r
}

// Request is an accessor.
func (e *EmbeddedSetter) Request() *http.Request {
	return e.r
}

func decodeValidate(d *form.Decoder, v interface{}, p url.Values, in rest.ParamIn, val rest.Validator) error {
	goValues := make(map[string]interface{}, len(p))

	err := d.Decode(v, p, goValues)
	if err != nil {
		return err
	}

	if len(p) > len(goValues) {
		for k := range p {
			if pos := strings.Index(k, "["); pos > 0 {
				k = k[:pos]
			}

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

func makeDecoder(in rest.ParamIn, formDecoder *form.Decoder, decoderFunc decoderFunc) valueDecoderFunc {
	return func(r *http.Request, v interface{}, validator rest.Validator) error {
		ct := r.Header.Get("Content-Type")
		if in == rest.ParamInFormData && ct != "" && ct != "multipart/form-data" && ct != "application/x-www-form-urlencoded" {
			return nil
		}

		values, err := decoderFunc(r)
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
	decoders    []valueDecoderFunc
	in          []rest.ParamIn
	isReqLoader bool
	isReqSetter bool
}

var _ nethttp.RequestDecoder = &decoder{}

// Decode populates and validates input with data from http request.
func (d *decoder) Decode(r *http.Request, input interface{}, validator rest.Validator) error {
	if d.isReqSetter {
		if i, ok := input.(Setter); ok {
			i.SetRequest(r)
		}
	}

	if d.isReqLoader {
		if i, ok := input.(Loader); ok {
			return i.LoadFromHTTPRequest(r)
		}
	}

	for i, decode := range d.decoders {
		err := decode(r, input, validator)
		if err != nil {
			//nolint:errorlint // Error is not wrapped, type assertion is more performant.
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

const defaultMaxMemory = 32 << 20 // 32 MB

func formDataToURLValues(r *http.Request) (url.Values, error) {
	if r.ContentLength == 0 {
		return nil, nil
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			return nil, err
		}
	} else if err := r.ParseForm(); err != nil {
		return nil, err
	}

	return r.PostForm, nil
}

func headerToURLValues(r *http.Request) (url.Values, error) {
	return url.Values(r.Header), nil
}

func queryToURLValues(r *http.Request) (url.Values, error) {
	return r.URL.Query(), nil
}

func formToURLValues(r *http.Request) (url.Values, error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			return nil, err
		}
	} else if err := r.ParseForm(); err != nil {
		return nil, err
	}

	return r.Form, nil
}

func cookiesToURLValues(r *http.Request) (url.Values, error) {
	cookies := r.Cookies()
	params := make(url.Values, len(cookies))

	for _, c := range cookies {
		params[c.Name] = []string{c.Value}
	}

	return params, nil
}
