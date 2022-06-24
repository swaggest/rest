package response

import (
	rest2 "github.com/swaggest/rest"
	"net/http"

	"github.com/swaggest/fchi"
	"github.com/swaggest/rest-fasthttp"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/usecase"
)

type withRestHandler interface {
	RestHandler() *rest2.HandlerTrait
}

// ValidatorMiddleware sets up response validator in suitable handlers.
func ValidatorMiddleware(factory rest2.ResponseValidatorFactory) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		var (
			withUseCase       rest.HandlerWithUseCase
			handlerTrait      withRestHandler
			useCaseWithOutput usecase.HasOutputPort
		)

		if !fhttp.HandlerAs(handler, &handlerTrait) ||
			!fhttp.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithOutput) {
			return handler
		}

		rh := handlerTrait.RestHandler()

		statusCode := rh.SuccessStatus
		if statusCode == 0 {
			statusCode = http.StatusOK

			if rest2.OutputHasNoContent(useCaseWithOutput.OutputPort()) {
				statusCode = http.StatusNoContent
			}
		}

		rh.RespValidator = factory.MakeResponseValidator(
			statusCode, rh.SuccessContentType, useCaseWithOutput.OutputPort(), rh.RespHeaderMapping,
		)

		return handler
	}
}
