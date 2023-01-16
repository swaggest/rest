package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func validation() usecase.Interactor {
	type inputPort struct {
		Header int  `header:"X-Input" minimum:"10" description:"Request minimum: 10, response maximum: 20."`
		Query  bool `query:"q" description:"This parameter will bypass explicit validation as it does not have constraints."`
		Data   struct {
			Value string `json:"value" minLength:"3" description:"Request minLength: 3, response maxLength: 7"`
		} `json:"data" required:"true"`
	}

	type outputPort struct {
		Header        int  `header:"X-Output" json:"-" maximum:"20"`
		AnotherHeader bool `header:"X-Query" json:"-" description:"This header bypasses validation as it does not have constraints."`
		Data          struct {
			Value string `json:"value" maxLength:"7"`
		} `json:"data" required:"true"`
	}

	u := usecase.NewIOI(new(inputPort), new(outputPort), func(ctx context.Context, input, output any) (err error) {
		in := input.(*inputPort)
		out := output.(*outputPort)

		out.Header = in.Header
		out.AnotherHeader = in.Query
		out.Data.Value = in.Data.Value

		return nil
	})

	u.SetTitle("Validation")
	u.SetDescription("Input/Output with validation.")

	return u
}
