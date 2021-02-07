package main

import (
	"context"

	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/usecase"
)

func jsonBody() usecase.Interactor {
	u := usecase.IOInteractor{}

	u.SetTitle("Request With JSON Body")
	u.SetDescription("Request with JSON body and query/header/path params, response with JSON body and data from request.")

	type JSONPayload struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type inputWithJSON struct {
		Header string          `header:"X-Header" description:"Simple scalar value in header."`
		Query  jsonschema.Date `query:"in_query" description:"Simple scalar value in query."`
		Path   string          `path:"in-path" description:"Simple scalar value in path"`
		JSONPayload
	}

	type outputWithJSON struct {
		Header string          `json:"inHeader"`
		Query  jsonschema.Date `json:"inQuery"`
		Path   string          `json:"inPath"`
		JSONPayload
	}

	u.Input = new(inputWithJSON)
	u.Output = new(outputWithJSON)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) (err error) {
		var (
			in  = input.(*inputWithJSON)
			out = output.(*outputWithJSON)
		)

		out.Query = in.Query
		out.Header = in.Header
		out.Path = in.Path
		out.JSONPayload = in.JSONPayload

		return nil
	})

	return u
}
