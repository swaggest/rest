//go:build go1.18

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/fchi"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

func jsonBodyManual() usecase.Interactor {
	type outputWithJSON struct {
		Header string          `json:"inHeader"`
		Query  jsonschema.Date `json:"inQuery" deprecated:"true"`
		Path   string          `json:"inPath"`
		JSONPayload
	}

	u := usecase.NewInteractor(func(ctx context.Context, in inputWithJSON, out *outputWithJSON) (err error) {
		out.Query = in.Query
		out.Header = in.Header
		out.Path = in.Path
		out.JSONPayload = in.JSONPayload

		return nil
	})

	u.SetTitle("Request With JSON Body and manual decoder")
	u.SetDescription("Request with JSON body and query/header/path params, response with JSON body and data from request.")

	return u
}

type JSONPayload struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type inputWithJSON struct {
	Header      string          `header:"X-Header" description:"Simple scalar value in header."`
	Query       jsonschema.Date `query:"in_query" description:"Simple scalar value in query."`
	Path        string          `path:"in-path" description:"Simple scalar value in path"`
	NamedStruct JSONPayload     `json:"namedStruct" deprecated:"true"`
	JSONPayload
}

var _ request.Loader = &inputWithJSON{}

func (i *inputWithJSON) LoadFromFastHTTPRequest(rc *fasthttp.RequestCtx) (err error) {
	if err = json.Unmarshal(rc.Request.Body(), i); err != nil {
		return fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	i.Header = string(rc.Request.Header.Peek("X-Header"))

	rc.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		if string(key) == "in_query" {
			if err = i.Query.UnmarshalText(value); err != nil {
				err = fmt.Errorf("failed to decode in_query %q: %w", string(value), err)
			}
		}
	})
	if err != nil {
		return err
	}

	if routeCtx := fchi.RouteContext(rc); routeCtx != nil {
		i.Path = routeCtx.URLParam("in-path")
	} else {
		return errors.New("missing path params in context")
	}

	return nil
}
