package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func dummy() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *struct{}) error {
		return nil
	})
	u.SetTags("Other")

	return u
}
