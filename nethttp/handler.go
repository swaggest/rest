package nethttp

import (
	"log"
	"net/http"
	"reflect"

	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

var _ http.Handler = &Handler{}

// NewHandler creates use case http handler.
func NewHandler(useCase usecase.Interactor, options ...func(h *Handler)) *Handler {
	h := &Handler{
		options: options,
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
	rest.HandlerTrait

	// OperationAnnotations are called after operation setup and before adding operation to documentation.
	OperationAnnotations []func(op *openapi3.Operation) error

	// requestDecoder maps data from http.Request into structured Go input value.
	requestDecoder RequestDecoder

	options []func(h *Handler)

	// failingUseCase allows to pass input decoding error through use case middlewares.
	failingUseCase usecase.Interactor

	useCase usecase.Interactor

	inputBufferType reflect.Type

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

// ServeHTTP serves http inputPort with use case interactor.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		input, output interface{}
		err           error
	)

	if h.inputBufferType != nil {
		if h.requestDecoder == nil {
			panic("request decoder is not initialized, please use SetRequestDecoder")
		}

		input = reflect.New(h.inputBufferType).Interface()
		err = h.requestDecoder.Decode(r, input, h.ReqValidator)

		if r.MultipartForm != nil {
			defer closeMultipartForm(r)
		}

		if err != nil {
			h.handleDecodeError(w, r, err)

			return
		}
	}

	if h.responseEncoder == nil {
		panic("response encoder is not initialized, please use SetResponseEncoder")
	}

	output = h.responseEncoder.MakeOutput(w, h.HandlerTrait)

	err = h.useCase.Interact(r.Context(), input, output)

	if err != nil {
		h.handleErrResponse(w, r, err)

		return
	}

	h.responseEncoder.WriteSuccessfulResponse(w, r, output, h.HandlerTrait)
}

func (h *Handler) handleErrResponse(w http.ResponseWriter, r *http.Request, err error) {
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

func closeMultipartForm(r *http.Request) {
	if err := r.MultipartForm.RemoveAll(); err != nil {
		log.Println(err)
	}
}

func (h *Handler) handleDecodeError(w http.ResponseWriter, r *http.Request, err error) {
	err = status.Wrap(err, status.InvalidArgument)

	if h.failingUseCase != nil {
		err = h.failingUseCase.Interact(r.Context(), "decoding failed", err)
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
		return handlerWithRoute{
			Handler:     handler,
			pathPattern: pathPattern,
			method:      method,
		}
	}
}

// RequestDecoder maps data from http.Request into structured Go input value.
type RequestDecoder interface {
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
