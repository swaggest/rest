package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func outputHeaders() usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithOutput
	}{}

	u.SetTitle("Output With Headers")
	u.SetDescription("Output with headers.")

	type headerOutput struct {
		Header string `header:"X-Header" json:"-" description:"Sample response header."`
		InBody string `json:"inBody" deprecated:"true"`
	}

	u.Output = new(headerOutput)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output any) (err error) {
		out := output.(*headerOutput)

		out.Header = "abc"
		out.InBody = "def"

		return nil
	})

	return u
}
