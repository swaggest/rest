//go:build go1.18

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

	u := usecase.NewInteractor(func(ctx context.Context, in inputQueryObject, out *outputQueryObject) (err error) {
		out.Query = in.Query

		return nil
	})

	u.SetTitle("Request With Object As Query Parameter")
	u.SetTags("Request")

	return u
}
