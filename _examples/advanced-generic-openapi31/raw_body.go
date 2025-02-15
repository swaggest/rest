package main

import (
	"context"
	"github.com/swaggest/usecase"
)

func rawBody() usecase.Interactor {
	type rawBody struct {
		TextBody string `contentType:"text/plain"`
		CSVBody  string `contentType:"text/csv"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input rawBody, output *rawBody) error {
		*output = input

		return nil
	})

	u.SetTitle("Request/response With Raw Body")
	u.SetDescription("The `contentType` tag acts as a discriminator of where to read/write the body.")
	u.SetTags("Request", "Response")

	return u
}
