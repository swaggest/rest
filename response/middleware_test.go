package response_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
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
	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		output.(*outputPort).Name = "Jane"
		output.(*outputPort).Items = []string{"one", "two", "three"}

		return nil
	})

	h := nethttp.NewHandler(u)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	response.EncoderMiddleware(h).ServeHTTP(rc, rc)
	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, `{"items":["one","two","three"]}`+"\n", string(rc.Response.Body()))
	assert.Equal(t, "Jane", string(rc.Response.Header.Peek("X-Name")))
}
