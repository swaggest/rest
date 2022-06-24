package response_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest-fasthttp/response"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

func TestValidatorMiddleware(t *testing.T) {
	u := struct {
		usecase.Interactor
		usecase.WithOutput
	}{}

	type outputPort struct {
		Name  string   `header:"X-Name" minLength:"3" json:"-"`
		Items []string `json:"items" minItems:"3"`
	}

	invalidOut := outputPort{
		Name:  "Ja",
		Items: []string{"one"},
	}

	u.Output = new(outputPort)
	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		out, ok := output.(*outputPort)
		require.True(t, ok)

		*out = invalidOut

		return nil
	})

	h := fhttp.NewHandler(u)

	apiSchema := &openapi.Collector{}
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)
	wh := fhttp.WrapHandler(h, response.EncoderMiddleware, response.ValidatorMiddleware(validatorFactory))

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	wh.ServeHTTP(rc, rc)
	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	assert.Equal(t, `{"status":"INTERNAL","error":"internal: bad response: validation failed",`+
		`"context":{"header:X-Name":["#: length must be >= 3, but got 2"]}}`+"\n", string(rc.Response.Body()))

	invalidOut.Name = "Jane"
	rc.Response = fasthttp.Response{}

	wh.ServeHTTP(rc, rc)
	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	assert.Equal(t, `{"status":"INTERNAL","error":"internal: bad response: validation failed",`+
		`"context":{"body":["#/items: minimum 3 items allowed, but found 1 items"]}}`+"\n", string(rc.Response.Body()))
}
