package jsonschema_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
)

// BenchmarkRequestValidator_ValidateRequestData-4   	  634356	      1761 ns/op	    2496 B/op	       8 allocs/op.
func BenchmarkRequestValidator_ValidateRequestData(b *testing.B) {
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, new(struct {
			Cookie string `cookie:"in_cookie" minLength:"3" required:"true"`
		}), nil)

	b.ResetTimer()
	b.ReportAllocs()

	value := map[string]interface{}{
		"in_cookie": "abc",
	}

	for i := 0; i < b.N; i++ {
		err := validator.ValidateData(rest.ParamInCookie, value)
		if err != nil {
			b.Fail()
		}
	}
}

func TestRequestValidator_ValidateData(t *testing.T) {
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, new(struct {
			Cookie   string `cookie:"in_cookie" minimum:"100" required:"true"`
			Query    string `query:"in_query_ignored" minLength:"3"`
			FormData string `formData:"inFormDataIgnored" minLength:"3"`
		}), rest.RequestMapping{
			rest.ParamInQuery:    map[string]string{"Query": "in_query"},
			rest.ParamInFormData: map[string]string{"FormData": "inFormData"},
		})

	err := validator.ValidateData(rest.ParamInCookie, map[string]interface{}{"in_cookie": 123})
	assert.Equal(t, err, rest.ValidationErrors{"cookie:in_cookie": []string{"#: expected string, but got number"}})

	err = validator.ValidateData(rest.ParamInCookie, map[string]interface{}{})
	assert.Equal(t, err, rest.ValidationErrors{"cookie:in_cookie": []string{"missing value"}})

	err = validator.ValidateData(rest.ParamInQuery, map[string]interface{}{"in_query": 123})
	assert.Equal(t, err, rest.ValidationErrors{"query:in_query": []string{"#: expected string, but got number"}})

	err = validator.ValidateData(rest.ParamInQuery, map[string]interface{}{"in_query": "ab"})
	assert.Equal(t, err, rest.ValidationErrors{"query:in_query": []string{"#: length must be >= 3, but got 2"}})

	assert.NoError(t, validator.ValidateData(rest.ParamInQuery, map[string]interface{}{}))
	assert.NoError(t, validator.ValidateData(rest.ParamInQuery, map[string]interface{}{"unknown": 123}))
	assert.NoError(t, validator.ValidateData(rest.ParamInQuery, map[string]interface{}{"in_query_ignored": 123}))
	assert.NoError(t, validator.ValidateData("unknown", map[string]interface{}{}))
	assert.NoError(t, validator.ValidateData(rest.ParamInCookie, map[string]interface{}{"in_cookie": "abc"}))

	assert.NoError(t, validator.ValidateData(rest.ParamInFormData, map[string]interface{}{"inFormData": "abc"}))

	err = validator.ValidateData(rest.ParamInFormData, map[string]interface{}{"inFormData": "ab"})
	assert.Equal(t, err, rest.ValidationErrors{"formData:inFormData": []string{"#: length must be >= 3, but got 2"}})
}

func TestFactory_MakeResponseValidator(t *testing.T) {
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeResponseValidator(http.StatusOK, "application/json", new(struct {
			Name  string `json:"name" minLength:"1"`
			Trace string `maxLength:"3"`
		}), map[string]string{
			"Trace": "x-TrAcE",
		})

	assert.NoError(t, validator.ValidateJSONBody([]byte(`{"name":"John"}`)))
	assert.Error(t, validator.ValidateJSONBody([]byte(`{"name":""}`))) // minLength:"1" violated.
	assert.NoError(t, validator.ValidateData(rest.ParamInHeader, map[string]interface{}{
		"X-Trace": "abc",
	}))
	assert.Error(t, validator.ValidateData(rest.ParamInHeader, map[string]interface{}{
		"X-Trace": "abcd", // maxLength:"3" violated.
	}))
}

func TestNullableTime(t *testing.T) {
	type request struct {
		ExpiryDate *time.Time `json:"expiryDate"`
	}

	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodPost, new(request), nil)
	err := validator.ValidateJSONBody([]byte(`{"expiryDate":null}`))

	assert.NoError(t, err, "%+v", err)
}

func TestValidator_ForbidUnknownParams(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"/?foo=bar&baz=1", nil)
	assert.NoError(t, err)

	type input struct {
		Foo string `query:"foo"`

		_ struct{} `query:"_" additionalProperties:"false"`
	}

	in := new(input)

	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, in, nil)
	validator := jsonschema.NewFactory(&openapi.Collector{}, &openapi.Collector{}).
		MakeRequestValidator(http.MethodGet, in, nil)

	err = dec.Decode(req, in, validator)
	assert.Equal(t, rest.ValidationErrors{"query:baz": []string{"unknown parameter with value 1"}}, err,
		fmt.Sprintf("%#v", err))
}
