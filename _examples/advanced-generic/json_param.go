//go:build go1.18

package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/swaggest/usecase"
)

func jsonParam() usecase.Interactor {
	type JSONPayload struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type inputWithJSON struct {
		Header   string      `header:"X-Header" description:"Simple scalar value in header."`
		Query    int         `query:"in_query" description:"Simple scalar value in query."`
		Path     string      `path:"in-path" description:"Simple scalar value in path"`
		Cookie   uuid.UUID   `cookie:"in_cookie" description:"UUID in cookie."`
		Identity JSONPayload `query:"identity" description:"JSON value in query"`
	}

	type outputWithJSON struct {
		Header string `json:"inHeader"`
		Query  int    `json:"inQuery"`
		Path   string `json:"inPath"`
		JSONPayload
	}

	u := usecase.NewInteractor(func(ctx context.Context, in inputWithJSON, out *outputWithJSON) (err error) {
		out.Query = in.Query
		out.Header = in.Header
		out.Path = in.Path
		out.JSONPayload = in.Identity

		return nil
	})

	u.SetTitle("Request With JSON Query Parameter")
	u.SetDescription("Request with JSON body and query/header/path params, response with JSON body and data from request.")

	return u
}
