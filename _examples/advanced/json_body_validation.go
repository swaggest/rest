package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func jsonBodyValidation() usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.SetTitle("Request With JSON Body and non-trivial validation")
	u.SetDescription("Request with JSON body and query/header/path params, response with JSON body and data from request.")

	type JSONPayload struct {
		ID   int    `json:"id" minimum:"100"`
		Name string `json:"name" minLength:"3"`
	}

	type inputWithJSON struct {
		Header string `header:"X-Header" description:"Simple scalar value in header." minLength:"3"`
		Query  int    `query:"in_query" description:"Simple scalar value in query." minimum:"100"`
		Path   string `path:"in-path" description:"Simple scalar value in path" minLength:"3"`
		JSONPayload
	}

	type outputWithJSON struct {
		Header string `json:"inHeader" minLength:"3"`
		Query  int    `json:"inQuery" minimum:"3"`
		Path   string `json:"inPath" minLength:"3"`
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
