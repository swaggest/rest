package response

import (
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
)

type responseEncoderSetter interface {
	SetResponseEncoder(responseWriter nethttp.ResponseEncoder)
}

// EncoderMiddleware instruments qualifying fchi.Handler with Encoder.
func EncoderMiddleware(handler fchi.Handler) fchi.Handler {
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
