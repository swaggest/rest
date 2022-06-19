package request_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jschema "github.com/swaggest/jsonschema-go"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
)

// BenchmarkDecoder_Decode-16    	 2276893	       453.3 ns/op	     440 B/op	       4 allocs/op.
func BenchmarkDecoder_Decode(b *testing.B) {
	df := request.NewDecoderFactory()

	type req struct {
		Q string `query:"q"`
		H int    `header:"X-H"`
	}

	r, err := http.NewRequest(http.MethodGet, "/?q=abc", nil)
	require.NoError(b, err)

	r.Header.Set("X-H", "123")

	d := df.MakeDecoder(http.MethodGet, new(req), nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := new(req)

		err = d.Decode(r, rr, nil)
		if err != nil {
			b.Fail()
		}
	}
}

type reqTest struct {
	Header   int    `header:"X-In-HeAdEr" required:"true"` // Headers are mapped using canonical names.
	Cookie   string `cookie:"in_cookie"`
	Query    string `query:"in_query"`
	Path     string `path:"in_path"`
	FormData string `formData:"inFormData"`
}

type reqTestCustomMapping struct {
	reqEmbedding
	Query    string
	Path     string
	FormData string
}

type reqEmbedding struct {
	Header int `required:"true"`
	Cookie string
}

type reqJSONTest struct {
	Query   string `query:"in_query"`
	BodyOne string `json:"bodyOne" required:"true"`
	BodyTwo []int  `json:"bodyTwo" minItems:"2"`
}

func TestDecoder_Decode(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/?in_query=abc",
		strings.NewReader(url.Values{"inFormData": []string{"def"}}.Encode()))
	assert.NoError(t, err)

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-In-hEaDeR", "123")

	c := http.Cookie{
		Name:  "in_cookie",
		Value: "jkl",
	}

	req.AddCookie(&c)

	df := request.NewDecoderFactory()
	df.SetDecoderFunc(rest.ParamInPath, func(r *http.Request) (url.Values, error) {
		assert.Equal(t, req, r)

		return url.Values{"in_path": []string{"mno"}}, nil
	})

	input := new(reqTest)
	dec := df.MakeDecoder(http.MethodPost, input, nil)

	assert.NoError(t, dec.Decode(req, input, nil))
	assert.Equal(t, "abc", input.Query)
	assert.Equal(t, "def", input.FormData)
	assert.Equal(t, 123, input.Header)
	assert.Equal(t, "jkl", input.Cookie)
	assert.Equal(t, "mno", input.Path)

	inputCM := new(reqTestCustomMapping)
	decCM := df.MakeDecoder(http.MethodPost, input, map[rest.ParamIn]map[string]string{
		rest.ParamInHeader: {
			"Header": "X-In-HeAdEr", // Headers are mapped using canonical names.
		},
		rest.ParamInCookie:   {"Cookie": "in_cookie"},
		rest.ParamInQuery:    {"Query": "in_query"},
		rest.ParamInPath:     {"Path": "in_path"},
		rest.ParamInFormData: {"FormData": "inFormData"},
	})

	assert.NoError(t, decCM.Decode(req, inputCM, nil))
	assert.Equal(t, "abc", inputCM.Query)
	assert.Equal(t, "def", inputCM.FormData)
	assert.Equal(t, 123, inputCM.Header)
	assert.Equal(t, "jkl", inputCM.Cookie)
	assert.Equal(t, "mno", inputCM.Path)
}

// BenchmarkDecoderFunc_Decode-4   	  440503	      2525 ns/op	    1513 B/op	      12 allocs/op.
func BenchmarkDecoderFunc_Decode(b *testing.B) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/?in_query=abc",
		strings.NewReader(url.Values{"inFormData": []string{"def"}}.Encode()))
	assert.NoError(b, err)

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-In-Header", "123")

	c := http.Cookie{
		Name:  "in_cookie",
		Value: "jkl",
	}

	req.AddCookie(&c)

	df := request.NewDecoderFactory()
	df.SetDecoderFunc(rest.ParamInPath, func(r *http.Request) (url.Values, error) {
		return url.Values{"in_path": []string{"mno"}}, nil
	})

	dec := df.MakeDecoder(http.MethodPost, new(reqTest), nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		input := new(reqTest)

		err := dec.Decode(req, input, nil)
		if err != nil {
			b.Fail()
		}

		if input.Header != 123 {
			b.Fail()
		}
	}
}

func TestDecoder_Decode_required(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	assert.NoError(t, err)

	input := new(reqTest)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodPost, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, input, nil)

	err = dec.Decode(req, input, validator)
	assert.Equal(t, rest.ValidationErrors{"header:X-In-HeAdEr": []string{"missing value"}}, err)
}

func TestDecoder_Decode_json(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/?in_query=cba",
		strings.NewReader(`{"bodyOne":"abc", "bodyTwo": [1,2,3]}`))
	assert.NoError(t, err)

	input := new(reqJSONTest)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodPost, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, input, nil)

	assert.NoError(t, dec.Decode(req, input, validator))
	assert.Equal(t, "cba", input.Query)
	assert.Equal(t, "abc", input.BodyOne)
	assert.Equal(t, []int{1, 2, 3}, input.BodyTwo)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "/",
		strings.NewReader(`{"bodyTwo":[1]}`))
	assert.NoError(t, err)

	err = dec.Decode(req, input, validator)
	assert.Equal(t, rest.ValidationErrors{"body": []string{
		"#: validation failed",
		"#: missing properties: \"bodyOne\"",
		"#/bodyTwo: minimum 2 items allowed, but found 1 items",
	}}, err)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "/",
		strings.NewReader(`{"bodyOne":"abc", "bodyTwo":[1]}`))
	assert.NoError(t, err)

	err = dec.Decode(req, input, validator)
	assert.Error(t, err)
	assert.Equal(t, rest.ValidationErrors{"body": []string{"#/bodyTwo: minimum 2 items allowed, but found 1 items"}}, err)
}

// BenchmarkDecoder_Decode_json-4   	   36660	     29688 ns/op	   12310 B/op	     169 allocs/op.
func BenchmarkDecoder_Decode_json(b *testing.B) {
	input := new(reqJSONTest)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodPost, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, input, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/?in_query=cba",
			strings.NewReader(`{"bodyOne":"abc", "bodyTwo": [1,2,3]}`))
		if err != nil {
			b.Fail()
		}

		err = dec.Decode(req, input, validator)
		if err != nil {
			b.Fail()
		}

		req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "/",
			strings.NewReader(`{"bodyTwo":[1]}`))
		if err != nil {
			b.Fail()
		}

		err = dec.Decode(req, input, validator)
		if err == nil {
			b.Fail()
		}

		req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, "/",
			strings.NewReader(`{"bodyOne":"abc", "bodyTwo":[1]}`))
		if err != nil {
			b.Fail()
		}

		err = dec.Decode(req, input, validator)
		if err == nil {
			b.Fail()
		}
	}
}

func TestDecoder_Decode_queryObject(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?in_query[1]=1.0&in_query[2]=2.1&in_query[3]=0", nil)
	assert.NoError(t, err)

	df := request.NewDecoderFactory()

	input := new(struct {
		InQuery map[int]float64 `query:"in_query"`
	})
	dec := df.MakeDecoder(http.MethodGet, input, nil)

	assert.NoError(t, dec.Decode(req, input, nil))
	assert.Equal(t, map[int]float64{1: 1, 2: 2.1, 3: 0}, input.InQuery)

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?in_query[1]=1.0&in_query[2]=2.1&in_query[c]=0", nil)
	assert.NoError(t, err)

	err = dec.Decode(req, input, nil)
	assert.Error(t, err)
	assert.Equal(t, rest.RequestErrors{"query:in_query": []string{
		"#: invalid integer value 'c' type 'int' namespace 'in_query'",
	}}, err)
}

// BenchmarkDecoder_Decode_queryObject-4   	  170670	      6104 ns/op	    2000 B/op	      36 allocs/op.
func BenchmarkDecoder_Decode_queryObject(b *testing.B) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?in_query[1]=1.0&in_query[2]=2.1&in_query[3]=0", nil)
	assert.NoError(b, err)

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?in_query[1]=1.0&in_query[2]=2.1&in_query[c]=0", nil)
	assert.NoError(b, err)

	df := request.NewDecoderFactory()

	input := new(struct {
		InQuery map[int]float64 `query:"in_query"`
	})
	dec := df.MakeDecoder(http.MethodGet, input, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err = dec.Decode(req, input, nil)
		if err != nil {
			b.Fail()
		}

		err = dec.Decode(req2, input, nil)
		if err == nil {
			b.Fail()
		}
	}
}

func TestDecoder_Decode_jsonParam(t *testing.T) {
	type inp struct {
		Filter struct {
			A int    `json:"a"`
			B string `json:"b"`
		} `query:"filter"`
	}

	df := request.NewDecoderFactory()
	dec := df.MakeDecoder(http.MethodGet, new(inp), nil)

	req, err := http.NewRequest(http.MethodGet, "/?filter=%7B%22a%22%3A123%2C%22b%22%3A%22abc%22%7D", nil)
	require.NoError(t, err)

	v := new(inp)
	require.NoError(t, dec.Decode(req, v, nil))

	assert.Equal(t, 123, v.Filter.A)
	assert.Equal(t, "abc", v.Filter.B)
}

// BenchmarkDecoder_Decode_jsonParam-4   	  525867	      2306 ns/op	     752 B/op	      12 allocs/op.
func BenchmarkDecoder_Decode_jsonParam(b *testing.B) {
	type inp struct {
		Filter struct {
			A int    `json:"a"`
			B string `json:"b"`
		} `query:"filter"`
	}

	df := request.NewDecoderFactory()
	dec := df.MakeDecoder(http.MethodGet, new(inp), nil)

	req, err := http.NewRequest(http.MethodGet, "/?filter=%7B%22a%22%3A123%2C%22b%22%3A%22abc%22%7D", nil)
	require.NoError(b, err)

	v := new(inp)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := dec.Decode(req, v, nil)
		if err != nil {
			b.Fail()
		}
	}
	assert.Equal(b, 123, v.Filter.A)
	assert.Equal(b, "abc", v.Filter.B)
}

func TestDecoder_Decode_error(t *testing.T) {
	type req struct {
		Q int `default:"100" query:"q"`
	}

	df := request.NewDecoderFactory()
	df.ApplyDefaults = true

	d := df.MakeDecoder(http.MethodGet, new(req), nil)
	r, err := http.NewRequest(http.MethodGet, "?q=undefined", nil)
	require.NoError(t, err)

	in := new(req)
	err = d.Decode(r, in, nil)
	assert.EqualError(t, err, "bad request")
	assert.Equal(t, rest.RequestErrors{"query:q": []string{
		"#: invalid integer value 'undefined' type 'int' namespace 'q'",
	}}, err)
}

func TestDecoder_Decode_dateTime(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?time=2020-04-04T00:00:00Z&date=2020-04-04", nil)
	assert.NoError(t, err)

	type reqTest struct {
		Time time.Time    `query:"time"`
		Date jschema.Date `query:"date"`
	}

	input := new(reqTest)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodGet, input, nil)

	err = dec.Decode(req, input, validator)
	assert.NoError(t, err, fmt.Sprintf("%v", err))
}

type inputWithLoader struct {
	Time time.Time    `query:"time"`
	Date jschema.Date `query:"date"`

	load func(r *http.Request) error
}

func (i *inputWithLoader) LoadFromHTTPRequest(r *http.Request) error {
	return i.load(r)
}

func TestDecoder_Decode_manualLoader(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?time=2020-04-04T00:00:00Z&date=2020-04-04", nil)
	assert.NoError(t, err)

	input := new(inputWithLoader)
	loadTriggered := false

	input.load = func(r *http.Request) error {
		assert.Equal(t, "/?time=2020-04-04T00:00:00Z&date=2020-04-04", r.URL.String())

		loadTriggered = true

		return nil
	}

	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodGet, input, nil)

	err = dec.Decode(req, input, validator)
	assert.NoError(t, err, fmt.Sprintf("%v", err))
	assert.True(t, loadTriggered)
	assert.True(t, input.Time.IsZero())
}

func TestDecoder_Decode_unknownParams(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?foo=1&bar=1&bar=2&baz&quux=123", nil)
	assert.NoError(t, err)

	type input struct {
		Foo string   `query:"foo"`
		Bar []string `query:"bar"`
		Baz *string  `query:"baz"`

		_ struct{} `query:"_" additionalProperties:"false"`
	}

	in := new(input)

	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, in, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodGet, in, nil)

	err = dec.Decode(req, in, validator)
	assert.Equal(t, rest.ValidationErrors{"query:quux": []string{"unknown parameter with value 123"}}, err,
		fmt.Sprintf("%#v", err))
}

func TestDecoder_Decode_multi(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?foo=1&foo=2&foo=3", nil)
	require.NoError(t, err)

	val := req.URL.Query()
	assert.Equal(t, []string{"1", "2", "3"}, val["foo"])

	type input struct {
		Foo1 int   `query:"foo"`
		Foo2 []int `query:"foo"`
	}

	in := new(input)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, in, nil)

	err = dec.Decode(req, in, nil)
	assert.NoError(t, err)

	assert.Equal(t, 1, in.Foo1)
	assert.Equal(t, []int{1, 2, 3}, in.Foo2)
}
