package response

import (
	"net/http"

	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
)

type withRestHandler interface {
	RestHandler() *rest.HandlerTrait
}

// ValidatorMiddleware sets up response validator in suitable handlers.
func ValidatorMiddleware(factory rest.ResponseValidatorFactory) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		if nethttp.IsWrapperChecker(handler) {
			return handler
		}

		var (
			withUseCase       rest.HandlerWithUseCase
			handlerTrait      withRestHandler
			useCaseWithOutput usecase.HasOutputPort
		)

		if !nethttp.HandlerAs(handler, &handlerTrait) ||
			!nethttp.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithOutput) {
			return handler
		}

		rh := handlerTrait.RestHandler()

		statusCode := rh.SuccessStatus
		if statusCode == 0 {
			statusCode = http.StatusOK

			if rest.OutputHasNoContent(useCaseWithOutput.OutputPort()) {
				statusCode = http.StatusNoContent
			}
		}

		rh.RespValidator = factory.MakeResponseValidator(
			statusCode, rh.SuccessContentType, useCaseWithOutput.OutputPort(), rh.RespHeaderMapping,
		)

		return handler
	}
}
