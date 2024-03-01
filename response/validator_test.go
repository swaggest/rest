package response_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
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
	u.Interactor = usecase.Interact(func(_ context.Context, _, output interface{}) error {
		out, ok := output.(*outputPort)
		require.True(t, ok)

		*out = invalidOut

		return nil
	})

	h := nethttp.NewHandler(u)

	apiSchema := &openapi.Collector{}
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)
	wh := nethttp.WrapHandler(h, response.EncoderMiddleware, response.ValidatorMiddleware(validatorFactory))

	assert.True(t, nethttp.MiddlewareIsWrapper(response.ValidatorMiddleware(validatorFactory)))

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	wh.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"status":"INTERNAL","error":"internal: bad response: validation failed",`+
		`"context":{"header:X-Name":["#: length must be >= 3, but got 2"]}}`+"\n", w.Body.String())

	invalidOut.Name = "Jane"
	w = httptest.NewRecorder()

	wh.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"status":"INTERNAL","error":"internal: bad response: validation failed",`+
		`"context":{"body":["#/items: minimum 3 items allowed, but found 1 items"]}}`+"\n", w.Body.String())
}
