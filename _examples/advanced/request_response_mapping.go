package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func reqRespMapping() usecase.Interactor {
	type inputPort struct {
		Val1 string `description:"Simple scalar value."`
		Val2 int    `description:"Simple scalar value."`
	}

	type outputPort struct {
		Val1 string `json:"-"`
		Val2 int    `json:"-"`
	}

	u := usecase.NewIOI(new(inputPort), new(outputPort), func(ctx context.Context, input, output interface{}) (err error) {
		var (
			in  = input.(*inputPort)
			out = output.(*outputPort)
		)

		out.Val1 = in.Val1
		out.Val2 = in.Val2

		return nil
	})

	u.SetTitle("Request Response Mapping")
	u.SetName("reqRespMapping")
	u.SetDescription("This use case has transport concerns fully decoupled with external req/resp mapping.")

	return u
}
