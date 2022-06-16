package request_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jschema "github.com/swaggest/jsonschema-go"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/valyala/fasthttp"
)

// BenchmarkDecoder_Decode-12    	 1410783	       797.9 ns/op	     866 B/op	      10 allocs/op.
// BenchmarkDecoder_Decode-12    	 2104834	       599.5 ns/op	      65 B/op	       6 allocs/op
// BenchmarkDecoder_Decode-12    	 1999123	       568.2 ns/op	      65 B/op	       6 allocs/op

// BenchmarkDecoder_Decode-4   	 1314788	       857 ns/op	     448 B/op	       4 allocs/op.
// BenchmarkDecoder_Decode-12    	 1633274	       638.5 ns/op	      65 B/op	       6 allocs/op. -- orig
// BenchmarkDecoder_Decode-12    	 2053461	       537.4 ns/op	      56 B/op	       3 allocs/op. -- unsafe
func BenchmarkDecoder_Decode(b *testing.B) {
	df := request.NewDecoderFactory()

	type req struct {
		Q string `query:"q"`
		H int    `header:"X-H"`
	}

	rc := fasthttp.RequestCtx{}

	rc.Request.SetRequestURI("/?q=abc")
	rc.Request.Header.Set("X-H", "123")

	d := df.MakeDecoder(http.MethodGet, new(req), nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := new(req)

		err := d.Decode(&rc, rr, nil)
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
	rc := &fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("/?in_query=abc")
	rc.Request.SetBody([]byte(url.Values{"inFormData": []string{"def"}}.Encode()))
	rc.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rc.Request.Header.Set("X-In-hEaDeR", "123")
	rc.Request.Header.SetCookie("in_cookie", "jkl")

	df := request.NewDecoderFactory()
	df.SetDecoderFunc(rest.ParamInPath, func(r *fasthttp.RequestCtx, params url.Values) error {
		assert.Equal(t, rc, r)
		params["in_path"] = []string{"mno"}

		return nil
	})

	input := new(reqTest)
	dec := df.MakeDecoder(http.MethodPost, input, nil)

	assert.NoError(t, dec.Decode(rc, input, nil))
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

	assert.NoError(t, decCM.Decode(rc, inputCM, nil))
	assert.Equal(t, "abc", inputCM.Query)
	assert.Equal(t, "def", inputCM.FormData)
	assert.Equal(t, 123, inputCM.Header)
	assert.Equal(t, "jkl", inputCM.Cookie)
	assert.Equal(t, "mno", inputCM.Path)
}

// BenchmarkDecoderFunc_Decode-4   	  440503	      2525 ns/op	    1513 B/op	      12 allocs/op.
func BenchmarkDecoderFunc_Decode(b *testing.B) {
	rc := &fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("/?in_query=abc")
	rc.Request.SetBody([]byte(url.Values{"inFormData": []string{"def"}}.Encode()))
	rc.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rc.Request.Header.Set("X-In-hEaDeR", "123")
	rc.Request.Header.SetCookie("in_cookie", "jkl")

	df := request.NewDecoderFactory()
	df.SetDecoderFunc(rest.ParamInPath, func(r *fasthttp.RequestCtx, params url.Values) error {
		params["in_path"] = []string{"mno"}

		return nil
	})

	dec := df.MakeDecoder(http.MethodPost, new(reqTest), nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		input := new(reqTest)

		err := dec.Decode(rc, input, nil)
		if err != nil {
			b.Fail()
		}

		if input.Header != 123 {
			b.Fail()
		}
	}
}

func TestDecoder_Decode_required(t *testing.T) {
	rc := &fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("/")

	input := new(reqTest)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodPost, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, input, nil)

	err := dec.Decode(rc, input, validator)
	assert.Equal(t, rest.ValidationErrors{"header:X-In-HeAdEr": []string{"missing value"}}, err)
}

func TestDecoder_Decode_json(t *testing.T) {
	rc := &fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("/?in_query=cba")
	rc.Request.SetBody([]byte(`{"bodyOne":"abc", "bodyTwo": [1,2,3]}`))

	input := new(reqJSONTest)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodPost, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, input, nil)

	assert.NoError(t, dec.Decode(rc, input, validator))
	assert.Equal(t, "cba", input.Query)
	assert.Equal(t, "abc", input.BodyOne)
	assert.Equal(t, []int{1, 2, 3}, input.BodyTwo)

	rc = &fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("/")
	rc.Request.SetBody([]byte(`{"bodyTwo":[1]}`))

	err := dec.Decode(rc, input, validator)
	assert.Equal(t, rest.ValidationErrors{"body": []string{
		"#: validation failed",
		"#: missing properties: \"bodyOne\"",
		"#/bodyTwo: minimum 2 items allowed, but found 1 items",
	}}, err)

	rc = &fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("/")
	rc.Request.SetBody([]byte(`{"bodyOne":"abc", "bodyTwo":[1]}`))

	err = dec.Decode(rc, input, validator)
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
		rc := fasthttp.RequestCtx{}
		rc.Request.Header.SetMethod(http.MethodPost)
		rc.Request.SetRequestURI("/?in_query=cba")
		rc.Request.SetBody([]byte(`{"bodyOne":"abc", "bodyTwo": [1,2,3]}`))

		err := dec.Decode(&rc, input, validator)
		if err != nil {
			b.Fail()
		}

		rc = fasthttp.RequestCtx{}
		rc.Request.Header.SetMethod(http.MethodPost)
		rc.Request.SetRequestURI("/")
		rc.Request.SetBody([]byte(`{"bodyTwo":[1]}`))

		err = dec.Decode(&rc, input, validator)
		if err == nil {
			b.Fail()
		}

		rc = fasthttp.RequestCtx{}
		rc.Request.Header.SetMethod(http.MethodPost)
		rc.Request.SetRequestURI("/")
		rc.Request.SetBody([]byte(`{"bodyOne":"abc", "bodyTwo":[1]}`))

		err = dec.Decode(&rc, input, validator)
		if err == nil {
			b.Fail()
		}
	}
}

func TestDecoder_Decode_queryObject(t *testing.T) {
	rc := req("/?in_query[1]=1.0&in_query[2]=2.1&in_query[3]=0")

	df := request.NewDecoderFactory()

	input := new(struct {
		InQuery map[int]float64 `query:"in_query"`
	})
	dec := df.MakeDecoder(http.MethodGet, input, nil)

	assert.NoError(t, dec.Decode(rc, input, nil))
	assert.Equal(t, map[int]float64{1: 1, 2: 2.1, 3: 0}, input.InQuery)

	rc = req("/?in_query[1]=1.0&in_query[2]=2.1&in_query[c]=0")

	err := dec.Decode(rc, input, nil)
	assert.Error(t, err)
	assert.Equal(t, rest.RequestErrors{"query:in_query": []string{
		"#: invalid integer value 'c' type 'int' namespace 'in_query'",
	}}, err)
}

// BenchmarkDecoder_Decode_queryObject-4   	  170670	      6104 ns/op	    2000 B/op	      36 allocs/op.
func BenchmarkDecoder_Decode_queryObject(b *testing.B) {
	rc := req("/?in_query[1]=1.0&in_query[2]=2.1&in_query[3]=0")

	rc2 := req("/?in_query[1]=1.0&in_query[2]=2.1&in_query[c]=0")

	df := request.NewDecoderFactory()

	input := new(struct {
		InQuery map[int]float64 `query:"in_query"`
	})
	dec := df.MakeDecoder(http.MethodGet, input, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := dec.Decode(rc, input, nil)
		if err != nil {
			b.Fail()
		}

		err = dec.Decode(rc2, input, nil)
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

	rc := req("/?filter=%7B%22a%22%3A123%2C%22b%22%3A%22abc%22%7D")

	v := new(inp)
	require.NoError(t, dec.Decode(rc, v, nil))

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

	rc := req("/?filter=%7B%22a%22%3A123%2C%22b%22%3A%22abc%22%7D")

	v := new(inp)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := dec.Decode(rc, v, nil)
		if err != nil {
			b.Fail()
		}
	}
	assert.Equal(b, 123, v.Filter.A)
	assert.Equal(b, "abc", v.Filter.B)
}

func TestDecoder_Decode_error(t *testing.T) {
	type reqs struct {
		Q int `default:"100" query:"q"`
	}

	df := request.NewDecoderFactory()
	df.ApplyDefaults = true

	d := df.MakeDecoder(http.MethodGet, new(reqs), nil)
	rc := req("?q=undefined")

	in := new(reqs)
	err := d.Decode(rc, in, nil)
	assert.EqualError(t, err, "bad request")
	assert.Equal(t, rest.RequestErrors{"query:q": []string{
		"#: invalid integer value 'undefined' type 'int' namespace 'q'",
	}}, err)
}

func TestDecoder_Decode_dateTime(t *testing.T) {
	rc := req("/?time=2020-04-04T00:00:00Z&date=2020-04-04")

	type reqTest struct {
		Time time.Time    `query:"time"`
		Date jschema.Date `query:"date"`
	}

	input := new(reqTest)
	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodGet, input, nil)

	err := dec.Decode(rc, input, validator)
	assert.NoError(t, err, fmt.Sprintf("%v", err))
}

type inputWithLoader struct {
	Time time.Time    `query:"time"`
	Date jschema.Date `query:"date"`

	load func(rc *fasthttp.RequestCtx) error
}

func (i *inputWithLoader) LoadFromFastHTTPRequest(r *fasthttp.RequestCtx) error {
	return i.load(r)
}

func TestDecoder_Decode_manualLoader(t *testing.T) {
	rc := req("/?time=2020-04-04T00:00:00Z&date=2020-04-04")

	input := new(inputWithLoader)
	loadTriggered := false

	input.load = func(rc *fasthttp.RequestCtx) error {
		assert.Equal(t, "/?time=2020-04-04T00:00:00Z&date=2020-04-04", string(rc.Request.RequestURI()))

		loadTriggered = true

		return nil
	}

	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, input, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodGet, input, nil)

	err := dec.Decode(rc, input, validator)
	assert.NoError(t, err, fmt.Sprintf("%v", err))
	assert.True(t, loadTriggered)
	assert.True(t, input.Time.IsZero())
}

func TestDecoder_Decode_unknownParams(t *testing.T) {
	rc := req("/?foo=1&bar=1&bar=2&baz&quux=123")

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

	err := dec.Decode(rc, in, validator)
	assert.Equal(t, rest.ValidationErrors{"query:quux": []string{"unknown parameter with value 123"}}, err,
		fmt.Sprintf("%#v", err))
}

func req(url string) *fasthttp.RequestCtx {
	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI(url)

	return rc
}
