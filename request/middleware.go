package request

import (
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

type requestDecoderSetter interface {
	SetRequestDecoder(fhttp.RequestDecoder)
}

type requestMapping interface {
	RequestMapping() rest.RequestMapping
}

// DecoderMiddleware sets up request decoder in suitable handlers.
func DecoderMiddleware(factory DecoderMaker) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		var (
			withRoute          rest.HandlerWithRoute
			withUseCase        rest.HandlerWithUseCase
			withRequestMapping requestMapping
			setRequestDecoder  requestDecoderSetter
			useCaseWithInput   usecase.HasInputPort
		)

		if !fhttp.HandlerAs(handler, &setRequestDecoder) ||
			!fhttp.HandlerAs(handler, &withRoute) ||
			!fhttp.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithInput) {
			return handler
		}

		var customMapping rest.RequestMapping
		if fhttp.HandlerAs(handler, &withRequestMapping) {
			customMapping = withRequestMapping.RequestMapping()
		}

		input := useCaseWithInput.InputPort()
		if input != nil {
			setRequestDecoder.SetRequestDecoder(
				factory.MakeDecoder(withRoute.RouteMethod(), useCaseWithInput.InputPort(), customMapping),
			)
		}

		return handler
	}
}

type withRestHandler interface {
	RestHandler() *rest.HandlerTrait
}

// ValidatorMiddleware sets up request validator in suitable handlers.
func ValidatorMiddleware(factory rest.RequestValidatorFactory) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		var (
			withRoute        rest.HandlerWithRoute
			withUseCase      rest.HandlerWithUseCase
			handlerTrait     withRestHandler
			useCaseWithInput usecase.HasInputPort
		)

		if !fhttp.HandlerAs(handler, &handlerTrait) ||
			!fhttp.HandlerAs(handler, &withRoute) ||
			!fhttp.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithInput) {
			return handler
		}

		rh := handlerTrait.RestHandler()

		rh.ReqValidator = factory.MakeRequestValidator(
			withRoute.RouteMethod(), useCaseWithInput.InputPort(), rh.ReqMapping)

		return handler
	}
}

var _ fhttp.RequestDecoder = DecoderFunc(nil)

// DecoderFunc implements RequestDecoder with a func.
type DecoderFunc func(rc *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error

// Decode implements RequestDecoder.
func (df DecoderFunc) Decode(rc *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error {
	return df(rc, input, validator)
}

// DecoderMaker creates request decoder for particular structured Go input value.
type DecoderMaker interface {
	MakeDecoder(method string, input interface{}, customMapping rest.RequestMapping) fhttp.RequestDecoder
}
