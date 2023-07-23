//go:build go1.18

package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func queryObject() usecase.Interactor {
	type jsonFilter struct {
		Foo string `json:"foo" maxLength:"5"`
	}

	type deepObjectFilter struct {
		Bar string `query:"bar" minLength:"3"`
	}

	type inputQueryObject struct {
		Query            map[int]float64  `query:"in_query" description:"Object value in query."`
		JSONFilter       jsonFilter       `query:"json_filter" description:"JSON object value in query."`
		DeepObjectFilter deepObjectFilter `query:"deep_object_filter" description:"Deep object value in query params."`
	}

	type outputQueryObject struct {
		Query            map[int]float64  `json:"inQuery"`
		JSONFilter       jsonFilter       `json:"jsonFilter"`
		DeepObjectFilter deepObjectFilter `json:"deepObjectFilter"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in inputQueryObject, out *outputQueryObject) (err error) {
		out.Query = in.Query
		out.JSONFilter = in.JSONFilter
		out.DeepObjectFilter = in.DeepObjectFilter

		return nil
	})

	u.SetTitle("Request With Object As Query Parameter")
	u.SetTags("Request")

	return u
}
