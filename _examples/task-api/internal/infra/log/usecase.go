// Package log provides logging helpers.
package log

import (
	"context"
	"io/ioutil"
	"log"

	"github.com/swaggest/usecase"
)

// UseCaseMiddleware creates logging use case middleware.
func UseCaseMiddleware() usecase.Middleware {
	return usecase.MiddlewareFunc(func(next usecase.Interactor) usecase.Interactor {
		if log.Writer() == ioutil.Discard {
			return next
		}

		var (
			hasName usecase.HasName
			name    = "unknown"
		)

		if usecase.As(next, &hasName) {
			name = hasName.Name()
		}

		return usecase.Interact(func(ctx context.Context, input, output any) error {
			err := next.Interact(ctx, input, output)
			if err != nil {
				log.Printf("usecase %s request (%v) failed: %v", name, input, err)
			}

			return err
		})
	})
}
