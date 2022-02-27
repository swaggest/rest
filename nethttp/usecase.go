package nethttp

import (
	"context"

	"github.com/swaggest/fchi"
	"github.com/swaggest/usecase"
)

// UseCaseMiddlewares applies use case middlewares to Handler.
func UseCaseMiddlewares(mw ...usecase.Middleware) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		var uh *Handler
		if !HandlerAs(handler, &uh) {
			return handler
		}

		u := uh.UseCase()
		fu := usecase.Wrap(u, usecase.MiddlewareFunc(func(next usecase.Interactor) usecase.Interactor {
			return usecase.Interact(func(ctx context.Context, input, output interface{}) error {
				return ctx.Value(decodeErrCtxKey{}).(error)
			})
		}))

		uh.SetUseCase(usecase.Wrap(u, mw...))
		uh.failingUseCase = usecase.Wrap(fu, mw...)

		return handler
	}
}
