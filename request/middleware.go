package request

import (
	"net/http"

	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
)

type requestDecoderSetter interface {
	SetRequestDecoder(rd nethttp.RequestDecoder)
}

type requestMapping interface {
	RequestMapping() rest.RequestMapping
}

// DecoderMiddleware sets up request decoder in suitable handlers.
func DecoderMiddleware(factory DecoderMaker) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		if nethttp.IsWrapperChecker(handler) {
			return handler
		}

		var (
			withRoute          rest.HandlerWithRoute
			withUseCase        rest.HandlerWithUseCase
			withRequestMapping requestMapping
			setRequestDecoder  requestDecoderSetter
			useCaseWithInput   usecase.HasInputPort
		)

		if !nethttp.HandlerAs(handler, &setRequestDecoder) ||
			!nethttp.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithInput) {
			return handler
		}

		var customMapping rest.RequestMapping
		if nethttp.HandlerAs(handler, &withRequestMapping) {
			customMapping = withRequestMapping.RequestMapping()
		}

		input := useCaseWithInput.InputPort()
		if input != nil {
			method := http.MethodPost // Default for handlers without method (for example NotFound handler).
			if nethttp.HandlerAs(handler, &withRoute) {
				method = withRoute.RouteMethod()
			}

			dec := factory.MakeDecoder(method, input, customMapping)
			setRequestDecoder.SetRequestDecoder(dec)
		}

		return handler
	}
}

type withRestHandler interface {
	RestHandler() *rest.HandlerTrait
}

// ValidatorMiddleware sets up request validator in suitable handlers.
func ValidatorMiddleware(factory rest.RequestValidatorFactory) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		if nethttp.IsWrapperChecker(handler) {
			return handler
		}

		var (
			withRoute        rest.HandlerWithRoute
			withUseCase      rest.HandlerWithUseCase
			handlerTrait     withRestHandler
			useCaseWithInput usecase.HasInputPort
		)

		if !nethttp.HandlerAs(handler, &handlerTrait) ||
			!nethttp.HandlerAs(handler, &withRoute) ||
			!nethttp.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithInput) {
			return handler
		}

		rh := handlerTrait.RestHandler()

		rh.ReqValidator = factory.MakeRequestValidator(
			withRoute.RouteMethod(), useCaseWithInput.InputPort(), rh.ReqMapping)

		return handler
	}
}

var _ nethttp.RequestDecoder = DecoderFunc(nil)

// DecoderFunc implements RequestDecoder with a func.
type DecoderFunc func(r *http.Request, input interface{}, validator rest.Validator) error

// Decode implements RequestDecoder.
func (df DecoderFunc) Decode(r *http.Request, input interface{}, validator rest.Validator) error {
	return df(r, input, validator)
}

// DecoderMaker creates request decoder for particular structured Go input value.
type DecoderMaker interface {
	MakeDecoder(method string, input interface{}, customMapping rest.RequestMapping) nethttp.RequestDecoder
}
