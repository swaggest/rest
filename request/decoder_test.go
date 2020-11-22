package request_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
)

// BenchmarkDecoder_Decode-4   	 1314788	       857 ns/op	     448 B/op	       4 allocs/op.
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
	Header   int    `header:"X-In-Header" required:"true"`
	Cookie   string `cookie:"in_cookie"`
	Query    string `query:"in_query"`
	Path     string `path:"in_path"`
	FormData string `formData:"inFormData"`
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
	req.Header.Set("X-In-Header", "123")

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
	assert.Equal(t, rest.ValidationErrors{"header:X-In-Header": []string{"missing value"}}, err)
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
		"#: Invalid Integer Value 'c' Type 'int' Namespace 'in_query'",
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
