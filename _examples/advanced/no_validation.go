package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func noValidation() usecase.Interactor {
	type inputPort struct {
		Header int  `header:"X-Input"`
		Query  bool `query:"q"`
		Data   struct {
			Value string `json:"value"`
		} `json:"data"`
	}

	type outputPort struct {
		Header        int  `header:"X-Output" json:"-"`
		AnotherHeader bool `header:"X-Query" json:"-"`
		Data          struct {
			Value string `json:"value"`
		} `json:"data"`
	}

	u := usecase.NewIOI(new(inputPort), new(outputPort), func(ctx context.Context, input, output interface{}) (err error) {
		in := input.(*inputPort)
		out := output.(*outputPort)

		out.Header = in.Header
		out.AnotherHeader = in.Query
		out.Data.Value = in.Data.Value

		return nil
	})

	u.SetTitle("No Validation")
	u.SetDescription("Input/Output without validation.")

	return u
}
