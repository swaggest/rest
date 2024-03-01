package nethttp

import (
	"context"
	"net/http"

	"github.com/swaggest/usecase"
)

// UseCaseMiddlewares applies use case middlewares to Handler.
func UseCaseMiddlewares(mw ...usecase.Middleware) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		if IsWrapperChecker(handler) {
			return handler
		}

		var uh *Handler
		if !HandlerAs(handler, &uh) {
			return handler
		}

		u := uh.UseCase()
		fu := usecase.Wrap(u, usecase.MiddlewareFunc(func(_ usecase.Interactor) usecase.Interactor {
			return usecase.Interact(func(ctx context.Context, _, _ interface{}) error {
				return ctx.Value(decodeErrCtxKey{}).(error)
			})
		}))

		uh.SetUseCase(usecase.Wrap(u, mw...))
		uh.failingUseCase = usecase.Wrap(fu, mw...)

		return handler
	}
}
