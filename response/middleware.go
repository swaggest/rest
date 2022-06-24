package response

import (
	"github.com/swaggest/fchi"
	rest2 "github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/usecase"
)

type responseEncoderSetter interface {
	SetResponseEncoder(responseWriter fhttp.ResponseEncoder)
}

// EncoderMiddleware instruments qualifying fchi.Handler with Encoder.
func EncoderMiddleware(handler fchi.Handler) fchi.Handler {
	var (
		withUseCase        rest2.HandlerWithUseCase
		setResponseEncoder responseEncoderSetter
		useCaseWithOutput  usecase.HasOutputPort
		restHandler        withRestHandler
	)

	if !fhttp.HandlerAs(handler, &setResponseEncoder) {
		return handler
	}

	responseEncoder := Encoder{}

	if fhttp.HandlerAs(handler, &withUseCase) &&
		fhttp.HandlerAs(handler, &restHandler) &&
		usecase.As(withUseCase.UseCase(), &useCaseWithOutput) {
		responseEncoder.SetupOutput(useCaseWithOutput.OutputPort(), restHandler.RestHandler())
	}

	setResponseEncoder.SetResponseEncoder(&responseEncoder)

	return handler
}
