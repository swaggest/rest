package nethttp_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

type Input struct {
	ID int
}

type Output struct {
	Value string `json:"value"`
}

func TestHandler_ServeHTTP(t *testing.T) {
	u := &struct {
		usecase.Interactor
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.Input = new(Input)
	u.Output = new(Output)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		in, ok := input.(*Input)
		require.True(t, ok)
		require.NotNil(t, in)
		assert.Equal(t, 123, in.ID)

		out, ok := output.(*Output)
		require.True(t, ok)
		require.NotNil(t, out)

		out.Value = "abc"

		return nil
	})

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/test")

	validatorCalled := false
	h := nethttp.NewHandler(u,
		func(h *nethttp.Handler) {
			h.ReqValidator = rest.ValidatorFunc(func(in rest.ParamIn, namedData map[string]interface{}) error {
				validatorCalled = true

				return nil
			})
		},
		func(h *nethttp.Handler) {
			h.SuccessStatus = http.StatusAccepted
		},
	)
	h.SetResponseEncoder(&response.Encoder{})

	h.SetRequestDecoder(request.DecoderFunc(
		func(r *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error {
			assert.Equal(t, rc, r)
			in, ok := input.(*Input)
			require.True(t, ok)
			require.NotNil(t, in)

			in.ID = 123

			assert.NoError(t, validator.ValidateData("", nil))

			return nil
		},
	))

	assert.Equal(t, u, h.UseCase())

	umwCalled := false
	hh := nethttp.UseCaseMiddlewares(usecase.MiddlewareFunc(func(next usecase.Interactor) usecase.Interactor {
		return usecase.Interact(func(ctx context.Context, input, output interface{}) error {
			umwCalled = true

			return next.Interact(ctx, input, output)
		})
	}))(h)

	hh.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusAccepted, rc.Response.StatusCode())
	assert.Equal(t, `{"value":"abc"}`+"\n", string(rc.Response.Body()))
	assert.True(t, validatorCalled)
	assert.True(t, umwCalled)
}

func TestHandler_ServeHTTP_decodeErr(t *testing.T) {
	u := &struct {
		usecase.Interactor
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.Input = new(Input)
	u.Output = new(Output)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		assert.Fail(t, "should not be called")

		return nil
	})

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/test")

	uh := nethttp.NewHandler(u)
	uh.SetRequestDecoder(request.DecoderFunc(
		func(r *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error {
			return errors.New("failed to decode request")
		},
	))
	uh.SetResponseEncoder(&response.Encoder{})

	umwCalled := false
	h := nethttp.UseCaseMiddlewares(usecase.MiddlewareFunc(func(next usecase.Interactor) usecase.Interactor {
		return usecase.Interact(func(ctx context.Context, input, output interface{}) error {
			umwCalled = true

			return next.Interact(ctx, input, output)
		})
	}))(uh)

	h.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	assert.Equal(t, `{"status":"INVALID_ARGUMENT","error":"invalid argument: failed to decode request"}`+"\n",
		string(rc.Response.Body()))
	assert.True(t, umwCalled)
}

func TestHandler_ServeHTTP_emptyPorts(t *testing.T) {
	u := struct {
		usecase.Interactor
	}{}

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		assert.Nil(t, input)
		assert.Nil(t, output)

		return nil
	})

	h := nethttp.NewHandler(u)
	h.SetResponseEncoder(&response.Encoder{})

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/test")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusNoContent, rc.Response.StatusCode())
	assert.Equal(t, "", string(rc.Response.Body()))
}

func TestHandler_ServeHTTP_customErrResp(t *testing.T) {
	u := struct {
		usecase.Interactor
		usecase.OutputWithNoContent
	}{}

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		assert.Nil(t, input)
		assert.Nil(t, output)

		return errors.New("use case failed")
	})

	h := nethttp.NewHandler(u)
	h.MakeErrResp = func(ctx context.Context, err error) (int, interface{}) {
		return http.StatusExpectationFailed, struct {
			Custom string `json:"custom"`
		}{
			Custom: err.Error(),
		}
	}
	h.SetResponseEncoder(&response.Encoder{})

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/test")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusExpectationFailed, rc.Response.StatusCode())
	assert.Equal(t, `{"custom":"use case failed"}`+"\n", string(rc.Response.Body()))
}

func TestHandlerWithRouteMiddleware(t *testing.T) {
	called := false

	var h fchi.Handler
	h = fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		called = true
	})

	h = nethttp.HandlerWithRouteMiddleware(http.MethodPost, "/test/")(h)
	hr, ok := h.(rest.HandlerWithRoute)
	require.True(t, ok)
	assert.Equal(t, http.MethodPost, hr.RouteMethod())
	assert.Equal(t, "/test/", hr.RoutePattern())

	h.ServeHTTP(context.Background(), nil)
	assert.True(t, called)
}

type reqWithBody struct {
	ID int `json:"id"`
}

func (*reqWithBody) ForceRequestBody() {}

func TestHandler_ServeHTTP_getWithBody(t *testing.T) {
	u := struct {
		usecase.Interactor
		usecase.WithInput
		usecase.OutputWithNoContent
	}{}

	u.Input = new(reqWithBody)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		in, ok := input.(*reqWithBody)
		assert.True(t, ok)
		assert.Equal(t, 123, in.ID)
		assert.Nil(t, output)

		return nil
	})

	h := nethttp.NewHandler(u)
	h.SetRequestDecoder(request.NewDecoderFactory().MakeDecoder(http.MethodGet, new(reqWithBody), nil))
	h.SetResponseEncoder(&response.Encoder{})

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/test")
	rc.Request.SetBody([]byte(`{"id":123}`))

	h.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusNoContent, rc.Response.StatusCode())
	assert.Equal(t, ``, string(rc.Response.Body()))
}

func TestHandler_ServeHTTP_customMapping(t *testing.T) {
	u := &struct {
		usecase.Interactor
		usecase.WithInput
		usecase.OutputWithNoContent
	}{}

	u.Input = new(Input)
	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		in, ok := input.(*Input)
		assert.True(t, ok)
		assert.Equal(t, 123, in.ID)

		return nil
	})

	uh := nethttp.NewHandler(u)
	uh.ReqMapping = rest.RequestMapping{
		rest.ParamInQuery: map[string]string{"ID": "ident"},
	}

	h := nethttp.WrapHandler(uh,
		request.DecoderMiddleware(request.NewDecoderFactory()),
		nethttp.HandlerWithRouteMiddleware(http.MethodGet, "/test"),
		response.EncoderMiddleware,
	)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/test?ident=123")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusNoContent, rc.Response.StatusCode())
	assert.Equal(t, "", string(rc.Response.Body()))
}

func TestOptionsMiddleware(t *testing.T) {
	u := usecase.NewIOI(nil, nil, func(ctx context.Context, input, output interface{}) error {
		return errors.New("failed")
	})
	h := nethttp.NewHandler(u, func(h *nethttp.Handler) {
		h.MakeErrResp = func(ctx context.Context, err error) (int, interface{}) {
			return http.StatusExpectationFailed, struct {
				Foo string `json:"foo"`
			}{Foo: err.Error()}
		}
	})
	h.SetResponseEncoder(&response.Encoder{})

	var loggedErr error

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	oh := nethttp.OptionsMiddleware(func(h *nethttp.Handler) {
		handleErrResponse := h.HandleErrResponse
		h.HandleErrResponse = func(ctx context.Context, r *fasthttp.RequestCtx, err error) {
			assert.Equal(t, rc, r)
			loggedErr = err
			handleErrResponse(ctx, r, err)
		}
	})(h)

	oh.ServeHTTP(rc, rc)

	assert.EqualError(t, loggedErr, "failed")
	assert.Equal(t, `{"foo":"failed"}`+"\n", string(rc.Response.Body()))
}
