package nethttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
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

	u.Interactor = usecase.Interact(func(_ context.Context, input, output interface{}) error {
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

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	validatorCalled := false
	h := nethttp.NewHandler(u,
		func(h *nethttp.Handler) {
			h.ReqValidator = rest.ValidatorFunc(func(_ rest.ParamIn, _ map[string]interface{}) error {
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
		func(r *http.Request, input interface{}, validator rest.Validator) error {
			assert.Equal(t, req, r)

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
	w := nethttp.UseCaseMiddlewares(usecase.MiddlewareFunc(func(next usecase.Interactor) usecase.Interactor {
		return usecase.Interact(func(ctx context.Context, input, output interface{}) error {
			umwCalled = true

			return next.Interact(ctx, input, output)
		})
	}))
	hh := w(h)

	assert.True(t, nethttp.MiddlewareIsWrapper(w))

	rw := httptest.NewRecorder()
	hh.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusAccepted, rw.Code)
	assert.Equal(t, `{"value":"abc"}`+"\n", rw.Body.String())
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

	u.Interactor = usecase.Interact(func(_ context.Context, _, _ interface{}) error {
		assert.Fail(t, "should not be called")

		return nil
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	uh := nethttp.NewHandler(u)
	uh.SetRequestDecoder(request.DecoderFunc(
		func(_ *http.Request, _ interface{}, _ rest.Validator) error {
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

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusBadRequest, rw.Code)
	assert.Equal(t, `{"status":"INVALID_ARGUMENT","error":"invalid argument: failed to decode request"}`+"\n",
		rw.Body.String())
	assert.True(t, umwCalled)
}

func TestHandler_ServeHTTP_emptyPorts(t *testing.T) {
	u := struct {
		usecase.Interactor
	}{}

	u.Interactor = usecase.Interact(func(_ context.Context, input, output interface{}) error {
		assert.Nil(t, input)
		assert.Nil(t, output)

		return nil
	})

	h := nethttp.NewHandler(u)
	h.SetResponseEncoder(&response.Encoder{})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusNoContent, rw.Code)
	assert.Equal(t, "", rw.Body.String())
}

func TestHandler_ServeHTTP_customErrResp(t *testing.T) {
	u := struct {
		usecase.Interactor
		usecase.OutputWithNoContent
	}{}

	u.Interactor = usecase.Interact(func(_ context.Context, input, output interface{}) error {
		assert.Nil(t, input)
		assert.Nil(t, output)

		return errors.New("use case failed")
	})

	h := nethttp.NewHandler(u)
	h.MakeErrResp = func(_ context.Context, err error) (int, interface{}) {
		return http.StatusExpectationFailed, struct {
			Custom string `json:"custom"`
		}{
			Custom: err.Error(),
		}
	}
	h.SetResponseEncoder(&response.Encoder{})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusExpectationFailed, rw.Code)
	assert.Equal(t, `{"custom":"use case failed"}`+"\n", rw.Body.String())
}

func TestHandlerWithRouteMiddleware(t *testing.T) {
	called := false

	var h http.Handler
	h = http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	h = nethttp.HandlerWithRouteMiddleware(http.MethodPost, "/test/")(h)
	hr, ok := h.(rest.HandlerWithRoute)
	require.True(t, ok)
	assert.Equal(t, http.MethodPost, hr.RouteMethod())
	assert.Equal(t, "/test/", hr.RoutePattern())

	h.ServeHTTP(nil, nil)
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

	u.Interactor = usecase.Interact(func(_ context.Context, input, output interface{}) error {
		in, ok := input.(*reqWithBody)
		assert.True(t, ok)
		assert.Equal(t, 123, in.ID)
		assert.Nil(t, output)

		return nil
	})

	h := nethttp.NewHandler(u)
	h.SetRequestDecoder(request.NewDecoderFactory().MakeDecoder(http.MethodGet, new(reqWithBody), nil))
	h.SetResponseEncoder(&response.Encoder{})

	req, err := http.NewRequest(http.MethodGet, "/test", strings.NewReader(`{"id":123}`))
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusNoContent, rw.Code)
	assert.Equal(t, ``, rw.Body.String())
}

func TestHandler_ServeHTTP_customMapping(t *testing.T) {
	u := &struct {
		usecase.Interactor
		usecase.WithInput
		usecase.OutputWithNoContent
	}{}

	u.Input = new(Input)
	u.Interactor = usecase.Interact(func(_ context.Context, input, _ interface{}) error {
		in, ok := input.(*Input)
		assert.True(t, ok)
		assert.Equal(t, 123, in.ID)

		return nil
	})

	uh := nethttp.NewHandler(u)
	uh.ReqMapping = rest.RequestMapping{
		rest.ParamInQuery: map[string]string{"ID": "ident"},
	}

	ws := []func(handler http.Handler) http.Handler{
		request.DecoderMiddleware(request.NewDecoderFactory()),
		nethttp.HandlerWithRouteMiddleware(http.MethodGet, "/test"),
		response.EncoderMiddleware,
	}

	h := nethttp.WrapHandler(uh, ws...)

	for i, w := range ws {
		assert.True(t, nethttp.MiddlewareIsWrapper(w), i)
	}

	req, err := http.NewRequest(http.MethodGet, "/test?ident=123", nil)
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusNoContent, rw.Code)
	assert.Equal(t, "", rw.Body.String())
}

func TestOptionsMiddleware(t *testing.T) {
	u := usecase.NewIOI(nil, nil, func(_ context.Context, _, _ interface{}) error {
		return errors.New("failed")
	})
	h := nethttp.NewHandler(u, func(h *nethttp.Handler) {
		h.MakeErrResp = func(_ context.Context, err error) (int, interface{}) {
			return http.StatusExpectationFailed, struct {
				Foo string `json:"foo"`
			}{Foo: err.Error()}
		}
	})
	h.SetResponseEncoder(&response.Encoder{})

	var loggedErr error

	rw := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	oh := nethttp.OptionsMiddleware(func(h *nethttp.Handler) {
		handleErrResponse := h.HandleErrResponse
		h.HandleErrResponse = func(w http.ResponseWriter, r *http.Request, err error) {
			assert.Equal(t, req, r)
			handleErrResponse(w, r, err)

			loggedErr = err
		}
	})(h)

	oh.ServeHTTP(rw, req)

	assert.EqualError(t, loggedErr, "failed")
	assert.Equal(t, `{"foo":"failed"}`+"\n", rw.Body.String())
}
