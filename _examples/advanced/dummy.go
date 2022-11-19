package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func dummy() usecase.Interactor {
	return usecase.NewIOI(nil, nil, func(ctx context.Context, input, output interface{}) error {
		return nil
	})
}
