package response

import (
	"net/http"

	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
)

type responseEncoderSetter interface {
	SetResponseEncoder(responseWriter nethttp.ResponseEncoder)
}

// EncoderMiddleware instruments qualifying http.Handler with Encoder.
func EncoderMiddleware(handler http.Handler) http.Handler {
	if nethttp.IsWrapperChecker(handler) {
		return handler
	}

	var (
		withUseCase        rest.HandlerWithUseCase
		setResponseEncoder responseEncoderSetter
		useCaseWithOutput  usecase.HasOutputPort
		restHandler        withRestHandler
	)

	if !nethttp.HandlerAs(handler, &setResponseEncoder) {
		return handler
	}

	responseEncoder := Encoder{}

	if nethttp.HandlerAs(handler, &withUseCase) &&
		nethttp.HandlerAs(handler, &restHandler) &&
		usecase.As(withUseCase.UseCase(), &useCaseWithOutput) {
		responseEncoder.SetupOutput(useCaseWithOutput.OutputPort(), restHandler.RestHandler())
	}

	setResponseEncoder.SetResponseEncoder(&responseEncoder)

	return handler
}
