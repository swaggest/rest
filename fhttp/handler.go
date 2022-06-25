package fhttp

import (
	"context"
	"net/http"
	"reflect"

	"github.com/swaggest/fchi"
	rest2 "github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/valyala/fasthttp"
)

var _ fchi.Handler = &Handler{}

// NewHandler creates use case http handler.
func NewHandler(useCase usecase.Interactor, options ...func(h *Handler)) *Handler {
	if useCase == nil {
		panic("usecase interactor is nil")
	}

	h := &Handler{
		options: options,
	}
	h.HandleErrResponse = h.handleErrResponseDefault
	h.SetUseCase(useCase)

	return h
}

// UseCase returns use case interactor.
func (h *Handler) UseCase() usecase.Interactor {
	return h.useCase
}

// SetUseCase prepares handler for a use case.
func (h *Handler) SetUseCase(useCase usecase.Interactor) {
	h.useCase = useCase

	for _, option := range h.options {
		option(h)
	}

	h.setupInputBuffer()
	h.setupOutputBuffer()
}

// Handler is a use case http handler with documentation and inputPort validation.
//
// Please use NewHandler to create instance.
type Handler struct {
	rest2.HandlerTrait

	// HandleErrResponse allows control of error response processing.
	HandleErrResponse func(ctx context.Context, rc *fasthttp.RequestCtx, err error)

	// requestDecoder maps data from http.Request into structured Go input value.
	requestDecoder RequestDecoder

	options []func(h *Handler)

	// failingUseCase allows to pass input decoding error through use case middlewares.
	failingUseCase usecase.Interactor

	useCase usecase.Interactor

	inputBufferType reflect.Type
	inputIsPtr      bool

	responseEncoder ResponseEncoder
}

// SetResponseEncoder sets response encoder.
func (h *Handler) SetResponseEncoder(responseEncoder ResponseEncoder) {
	h.responseEncoder = responseEncoder

	h.setupOutputBuffer()
}

// SetRequestDecoder sets request decoder.
func (h *Handler) SetRequestDecoder(requestDecoder RequestDecoder) {
	h.requestDecoder = requestDecoder
}

func (h *Handler) decodeRequest(rc *fasthttp.RequestCtx) (interface{}, error) {
	if h.requestDecoder == nil {
		panic("request decoder is not initialized, please use SetRequestDecoder")
	}

	iv := reflect.New(h.inputBufferType)
	err := h.requestDecoder.Decode(rc, iv.Interface(), h.ReqValidator)

	if !h.inputIsPtr {
		return iv.Elem().Interface(), err
	}

	return iv.Interface(), err
}

// ServeHTTP serves http inputPort with use case interactor.
func (h *Handler) ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	var (
		input, output interface{}
		err           error
	)

	if h.responseEncoder == nil {
		panic("response encoder is not initialized, please use SetResponseEncoder")
	}

	output = h.responseEncoder.MakeOutput(rc, h.HandlerTrait)

	if h.inputBufferType != nil {
		input, err = h.decodeRequest(rc)

		if err != nil {
			h.handleDecodeError(ctx, rc, err, input, output)

			return
		}
	}

	err = h.useCase.Interact(ctx, input, output)

	if err != nil {
		h.handleErrResponse(ctx, rc, err)

		return
	}

	h.responseEncoder.WriteSuccessfulResponse(rc, output, h.HandlerTrait)
}

func (h *Handler) handleErrResponseDefault(ctx context.Context, rc *fasthttp.RequestCtx, err error) {
	var (
		code int
		er   interface{}
	)

	if h.MakeErrResp != nil {
		code, er = h.MakeErrResp(ctx, err)
	} else {
		code, er = rest.Err(err)
	}

	h.responseEncoder.WriteErrResponse(rc, code, er)
}

func (h *Handler) handleErrResponse(ctx context.Context, rc *fasthttp.RequestCtx, err error) {
	if h.HandleErrResponse != nil {
		h.HandleErrResponse(ctx, rc, err)

		return
	}

	h.handleErrResponseDefault(ctx, rc, err)
}

type decodeErrCtxKey struct{}

func (h *Handler) handleDecodeError(ctx context.Context, rc *fasthttp.RequestCtx, err error, input, output interface{}) {
	err = status.Wrap(err, status.InvalidArgument)

	if h.failingUseCase != nil {
		err = h.failingUseCase.Interact(context.WithValue(ctx, decodeErrCtxKey{}, err), input, output)
	}

	h.handleErrResponse(ctx, rc, err)
}

func (h *Handler) setupInputBuffer() {
	h.inputBufferType = nil

	var withInput usecase.HasInputPort
	if !usecase.As(h.useCase, &withInput) {
		return
	}

	h.inputBufferType = reflect.TypeOf(withInput.InputPort())
	if h.inputBufferType != nil {
		if h.inputBufferType.Kind() == reflect.Ptr {
			h.inputBufferType = h.inputBufferType.Elem()
			h.inputIsPtr = true
		}
	}
}

func (h *Handler) setupOutputBuffer() {
	var (
		withOutput usecase.HasOutputPort
		output     interface{}
	)

	if usecase.As(h.useCase, &withOutput) && reflect.TypeOf(withOutput.OutputPort()) != nil {
		output = withOutput.OutputPort()
	} else if h.SuccessStatus == 0 {
		h.SuccessStatus = http.StatusNoContent
	}

	if h.responseEncoder != nil {
		h.responseEncoder.SetupOutput(output, &h.HandlerTrait)
	}
}

type handlerWithRoute struct {
	fchi.Handler
	method      string
	pathPattern string
}

func (h handlerWithRoute) RouteMethod() string {
	return h.method
}

func (h handlerWithRoute) RoutePattern() string {
	return h.pathPattern
}

// HandlerWithRouteMiddleware wraps handler with routing information.
func HandlerWithRouteMiddleware(method, pathPattern string) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		return handlerWithRoute{
			Handler:     handler,
			pathPattern: pathPattern,
			method:      method,
		}
	}
}

// RequestDecoder maps data from http.Request into structured Go input value.
type RequestDecoder interface {
	Decode(rc *fasthttp.RequestCtx, input interface{}, validator rest2.Validator) error
}

// ResponseEncoder writes data from use case output/error into http.ResponseWriter.
type ResponseEncoder interface {
	WriteErrResponse(rc *fasthttp.RequestCtx, statusCode int, response interface{})
	WriteSuccessfulResponse(
		rc *fasthttp.RequestCtx,
		output interface{},
		ht rest2.HandlerTrait,
	)
	SetupOutput(output interface{}, ht *rest2.HandlerTrait)
	MakeOutput(rc *fasthttp.RequestCtx, ht rest2.HandlerTrait) interface{}
}
