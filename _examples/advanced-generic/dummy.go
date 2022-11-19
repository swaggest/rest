package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func dummy() usecase.Interactor {
	return usecase.NewInteractor(func(ctx context.Context, input struct{}, output *struct{}) error {
		return nil
	})
}
