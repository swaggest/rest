package request

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/swaggest/form/v5"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/refl"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/valyala/fasthttp"
)

var _ DecoderMaker = &DecoderFactory{}

const (
	defaultTag  = "default"
	jsonTag     = "json"
	fileTag     = "file"
	formDataTag = "formData"
)

// DecoderFactory decodes http requests.
//
// Please use NewDecoderFactory to create instance.
type DecoderFactory struct {
	// ApplyDefaults enables default value assignment for fields missing explicit value in request.
	// Default value is retrieved from `default` field tag.
	ApplyDefaults bool

	// JSONReader allows custom JSON decoder for request body.
	// If not set encoding/json.Decoder is used.
	JSONReader func(rd io.Reader, v interface{}) error

	formDecoders      map[rest.ParamIn]*form.Decoder
	decoderFunctions  map[rest.ParamIn]decoderFunc
	defaultValDecoder *form.Decoder
	customDecoders    []customDecoder
}

type customDecoder struct {
	types []interface{}
	fn    form.DecodeFunc
}

// NewDecoderFactory creates request decoder factory.
func NewDecoderFactory() *DecoderFactory {
	df := DecoderFactory{}
	df.SetDecoderFunc(rest.ParamInCookie, cookiesToURLValues)
	df.SetDecoderFunc(rest.ParamInFormData, formDataToURLValues)
	df.SetDecoderFunc(rest.ParamInHeader, headerToURLValues)
	df.SetDecoderFunc(rest.ParamInQuery, queryToURLValues)

	defaultValDecoder := form.NewDecoder()
	defaultValDecoder.RegisterTagNameFunc(func(field reflect.StructField) string {
		return field.Name
	})

	df.defaultValDecoder = defaultValDecoder

	return &df
}

// SetDecoderFunc adds custom decoder function for values of particular field tag name.
func (df *DecoderFactory) SetDecoderFunc(tagName rest.ParamIn, d func(rc *fasthttp.RequestCtx, v map[string]string) error) {
	if df.decoderFunctions == nil {
		df.decoderFunctions = make(map[rest.ParamIn]decoderFunc)
	}

	if df.formDecoders == nil {
		df.formDecoders = make(map[rest.ParamIn]*form.Decoder)
	}

	df.decoderFunctions[tagName] = d
	dec := form.NewDecoder()
	dec.SetTagName(string(tagName))
	dec.SetMode(form.ModeExplicit)
	df.formDecoders[tagName] = dec
}

// MakeDecoder creates request.RequestDecoder for a http method and request structure.
//
// Input is checked for `json`, `file` tags only for methods with body semantics (POST, PUT, PATCH) or
// if input implements openapi3.RequestBodyEnforcer.
//
// CustomMapping can be nil, otherwise it is used instead of field tags to match decoded fields with struct.
func (df *DecoderFactory) MakeDecoder(
	method string,
	input interface{},
	customMapping rest.RequestMapping,
) nethttp.RequestDecoder {
	m := decoder{
		decoders: make([]valueDecoderFunc, 0),
		in:       make([]rest.ParamIn, 0),
	}

	if df.ApplyDefaults && refl.HasTaggedFields(input, defaultTag) {
		df.makeDefaultDecoder(input, &m)
	}

	cm := df.prepareCustomMapping(input, customMapping)

	if len(cm) > 0 {
		df.makeCustomMappingDecoder(cm, &m)
	}

	for in, formDecoder := range df.formDecoders {
		if _, exists := cm[in]; exists {
			continue
		}

		if refl.HasTaggedFields(input, string(in)) {
			df.jsonParams(formDecoder, in, input)
			m.decoders = append(m.decoders, makeDecoder(in, formDecoder, df.decoderFunctions[in]))
			m.in = append(m.in, in)
		}
	}

	method = strings.ToUpper(method)

	_, forceRequestBody := input.(openapi3.RequestBodyEnforcer)

	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && !forceRequestBody {
		return &m
	}

	// Checking for body tags.
	if refl.HasTaggedFields(input, jsonTag) || refl.FindEmbeddedSliceOrMap(input) != nil {
		if df.JSONReader != nil {
			m.decoders = append(m.decoders, decodeJSONBody(df.JSONReader))
		} else {
			m.decoders = append(m.decoders, decodeJSONBody(readJSON))
		}

		m.in = append(m.in, rest.ParamInBody)
	}

	if hasFileFields(input, fileTag) || hasFileFields(input, formDataTag) {
		m.decoders = append(m.decoders, decodeFiles)
		m.in = append(m.in, rest.ParamInFormData)
	}

	return &m
}

func (df *DecoderFactory) prepareCustomMapping(input interface{}, customMapping rest.RequestMapping) rest.RequestMapping {
	// Copy custom mapping to avoid mutability issues on original map.
	cm := make(rest.RequestMapping, len(customMapping))
	for k, v := range customMapping {
		cm[k] = v
	}

	// Move header names to custom mapping and/or apply canonical form to match net/http request decoder.
	if hdm, exists := cm[rest.ParamInHeader]; !exists && refl.HasTaggedFields(input, string(rest.ParamInHeader)) {
		hdm = make(map[string]string)

		refl.WalkTaggedFields(reflect.ValueOf(input), func(v reflect.Value, sf reflect.StructField, tag string) {
			hdm[sf.Name] = http.CanonicalHeaderKey(tag)
		}, string(rest.ParamInHeader))

		cm[rest.ParamInHeader] = hdm
	} else if exists {
		for k, v := range hdm {
			hdm[k] = http.CanonicalHeaderKey(v)
		}
	}

	fields := make(map[string]bool)

	refl.WalkTaggedFields(reflect.ValueOf(input), func(v reflect.Value, sf reflect.StructField, tag string) {
		fields[sf.Name] = true
	}, "")

	// Check if there are non-existent fields in mapping.
	var nonExistent []string

	for _, items := range cm {
		for k := range items {
			if _, exists := fields[k]; !exists {
				nonExistent = append(nonExistent, k)
			}
		}
	}

	if len(nonExistent) > 0 {
		sort.Strings(nonExistent)
		panic("non existent fields in mapping: " + strings.Join(nonExistent, ", "))
	}

	return cm
}

// jsonParams configures custom decoding for parameters with JSON struct values.
func (df *DecoderFactory) jsonParams(formDecoder *form.Decoder, in rest.ParamIn, input interface{}) {
	// Check fields for struct values with json tags. E.g. query parameter with json value.
	refl.WalkTaggedFields(reflect.ValueOf(input), func(v reflect.Value, sf reflect.StructField, tag string) {
		// Skip unexported fields.
		if sf.PkgPath != "" {
			return
		}

		fieldVal := v.Interface()

		if refl.HasTaggedFields(fieldVal, jsonTag) {
			// If value is a struct with `json` tags, custom decoder unmarshals json
			// from a string value into a struct.
			formDecoder.RegisterFunc(func(s string) (interface{}, error) {
				var err error
				f := reflect.New(sf.Type)
				if df.JSONReader != nil {
					err = df.JSONReader(bytes.NewBufferString(s), f.Interface())
				} else {
					err = json.Unmarshal([]byte(s), f.Interface())
				}

				if err != nil {
					return nil, err
				}

				return reflect.Indirect(f).Interface(), nil
			}, fieldVal)
		}
	}, string(in))
}

func (df *DecoderFactory) makeDefaultDecoder(input interface{}, m *decoder) {
	defaults := map[string]string{}

	refl.WalkTaggedFields(reflect.ValueOf(input), func(v reflect.Value, sf reflect.StructField, tag string) {
		defaults[sf.Name] = tag
	}, defaultTag)

	dec := df.defaultValDecoder

	m.decoders = append(m.decoders, func(rc *fasthttp.RequestCtx, v interface{}, validator rest.Validator) error {
		return dec.Decode(v, defaults)
	})
	m.in = append(m.in, defaultTag)
}

func (df *DecoderFactory) makeCustomMappingDecoder(customMapping rest.RequestMapping, m *decoder) {
	for in, mapping := range customMapping {
		dec := form.NewDecoder()
		dec.SetTagName(string(in))

		// Copy mapping to avoid mutability.
		mm := make(map[string]string, len(mapping))
		for k, v := range mapping {
			mm[k] = v
		}

		dec.RegisterTagNameFunc(func(field reflect.StructField) string {
			n := mm[field.Name]
			if n == "" && !field.Anonymous {
				return "-"
			}

			return n
		})

		for _, c := range df.customDecoders {
			dec.RegisterFunc(c.fn, c.types...)
		}

		m.decoders = append(m.decoders, makeDecoder(in, dec, df.decoderFunctions[in]))
		m.in = append(m.in, in)
	}
}

// RegisterFunc adds custom type handling.
func (df *DecoderFactory) RegisterFunc(fn form.DecodeFunc, types ...interface{}) {
	for _, fd := range df.formDecoders {
		fd.RegisterFunc(fn, types...)
	}

	df.defaultValDecoder.RegisterFunc(fn, types...)

	df.customDecoders = append(df.customDecoders, customDecoder{
		fn:    fn,
		types: types,
	})
}
