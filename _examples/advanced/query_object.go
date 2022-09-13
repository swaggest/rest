package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func queryObject() usecase.Interactor {
	type inputQueryObject struct {
		Query map[int]float64 `query:"in_query" description:"Object value in query."`
	}

	type outputQueryObject struct {
		Query map[int]float64 `json:"inQuery"`
	}

	u := usecase.NewIOI(new(inputQueryObject), new(outputQueryObject),
		func(ctx context.Context, input, output any) (err error) {
			var (
				in  = input.(*inputQueryObject)
				out = output.(*outputQueryObject)
			)

			out.Query = in.Query

			return nil
		})

	u.SetTitle("Request With Object As Query Parameter")

	return u
}
