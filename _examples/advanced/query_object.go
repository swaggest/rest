package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func queryObject() usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.SetTitle("Request With Object As Query Parameter")

	type inputQueryObject struct {
		Query map[int]float64 `query:"in_query" description:"Object value in query."`
	}

	type outputQueryObject struct {
		Query map[int]float64 `json:"inQuery"`
	}

	u.Input = new(inputQueryObject)
	u.Output = new(outputQueryObject)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) (err error) {
		var (
			in  = input.(*inputQueryObject)
			out = output.(*outputQueryObject)
		)

		out.Query = in.Query

		return nil
	})

	return u
}
