package response_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
)

func TestEncoderMiddleware(t *testing.T) {
	u := struct {
		usecase.Interactor
		usecase.WithOutput
	}{}

	type outputPort struct {
		Name  string   `header:"X-Name" json:"-"`
		Items []string `json:"items"`
	}

	u.Output = new(outputPort)
	u.Interactor = usecase.Interact(func(_ context.Context, _, output interface{}) error {
		output.(*outputPort).Name = "Jane"
		output.(*outputPort).Items = []string{"one", "two", "three"}

		return nil
	})

	h := nethttp.NewHandler(u)

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	em := response.EncoderMiddleware
	em(h).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"items":["one","two","three"]}`+"\n", w.Body.String())
	assert.Equal(t, "Jane", w.Header().Get("X-Name"))
	assert.True(t, nethttp.MiddlewareIsWrapper(em))
}
