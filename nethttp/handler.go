package nethttp

import (
	"context"
	"log"
	"net/http"
	"reflect"

	"github.com/swaggest/rest"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

var _ http.Handler = &Handler{}

// NewHandler creates use case http handler.
func NewHandler(useCase usecase.Interactor, options ...func(h *Handler)) *Handler {
	if useCase == nil {
		panic("usecase interactor is nil")
	}

	h := &Handler{
		options: options,
	}
	h.HandleErrResponse = h.handleErrResponseDefault

	for _, option := range h.options {
		option(h)
	}

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

	h.setupInputBuffer()
	h.setupOutputBuffer()
}

// Handler is a use case http handler with documentation and inputPort validation.
//
// Please use NewHandler to create instance.
type Handler struct {
	rest.HandlerTrait

	// HandleErrResponse allows control of error response processing.
	HandleErrResponse func(w http.ResponseWriter, r *http.Request, err error)

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

func (h *Handler) decodeRequest(r *http.Request) (interface{}, error) {
	if h.requestDecoder == nil {
		panic("request decoder is not initialized, please use SetRequestDecoder")
	}

	iv := reflect.New(h.inputBufferType)
	err := h.requestDecoder.Decode(r, iv.Interface(), h.ReqValidator)

	if !h.inputIsPtr {
		return iv.Elem().Interface(), err
	}

	return iv.Interface(), err
}

// ServeHTTP serves http inputPort with use case interactor.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		input, output interface{}
		err           error
	)

	if h.responseEncoder == nil {
		panic("response encoder is not initialized, please use SetResponseEncoder")
	}

	output = h.responseEncoder.MakeOutput(w, h.HandlerTrait)

	if h.inputBufferType != nil {
		input, err = h.decodeRequest(r)

		if r.MultipartForm != nil {
			defer closeMultipartForm(r)
		}

		if err != nil {
			h.handleDecodeError(w, r, err, input, output)

			return
		}
	}

	err = h.useCase.Interact(r.Context(), input, output)
	if err != nil {
		h.handleErrResponse(w, r, err)

		return
	}

	h.responseEncoder.WriteSuccessfulResponse(w, r, output, h.HandlerTrait)
}

func (h *Handler) handleErrResponseDefault(w http.ResponseWriter, r *http.Request, err error) {
	var (
		code int
		er   interface{}
	)

	if h.MakeErrResp != nil {
		code, er = h.MakeErrResp(r.Context(), err)
	} else {
		code, er = rest.Err(err)
	}

	h.responseEncoder.WriteErrResponse(w, r, code, er)
}

func (h *Handler) handleErrResponse(w http.ResponseWriter, r *http.Request, err error) {
	if h.HandleErrResponse != nil {
		h.HandleErrResponse(w, r, err)

		return
	}

	h.handleErrResponseDefault(w, r, err)
}

func closeMultipartForm(r *http.Request) {
	if err := r.MultipartForm.RemoveAll(); err != nil {
		log.Println(err)
	}
}

type decodeErrCtxKey struct{}

func (h *Handler) handleDecodeError(w http.ResponseWriter, r *http.Request, err error, input, output interface{}) {
	err = status.Wrap(err, status.InvalidArgument)

	if h.failingUseCase != nil {
		err = h.failingUseCase.Interact(context.WithValue(r.Context(), decodeErrCtxKey{}, err), input, output)
	}

	h.handleErrResponse(w, r, err)
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
	http.Handler
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
func HandlerWithRouteMiddleware(method, pathPattern string) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		if IsWrapperChecker(handler) {
			return handler
		}

		return handlerWithRoute{
			Handler:     handler,
			pathPattern: pathPattern,
			method:      method,
		}
	}
}

// RequestDecoder maps data from http.Request into structured Go input value.
type RequestDecoder interface {
	// Decode fills input with data from request, input should be a pointer.
	Decode(r *http.Request, input interface{}, validator rest.Validator) error
}

// ResponseEncoder writes data from use case output/error into http.ResponseWriter.
type ResponseEncoder interface {
	WriteErrResponse(w http.ResponseWriter, r *http.Request, statusCode int, response interface{})
	WriteSuccessfulResponse(
		w http.ResponseWriter,
		r *http.Request,
		output interface{},
		ht rest.HandlerTrait,
	)
	SetupOutput(output interface{}, ht *rest.HandlerTrait)
	MakeOutput(w http.ResponseWriter, ht rest.HandlerTrait) interface{}
}
