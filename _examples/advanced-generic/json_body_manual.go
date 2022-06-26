//go:build go1.18

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
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

func (i *inputWithJSON) LoadFromHTTPRequest(r *http.Request) (err error) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %s", err.Error())
		}
	}()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	if err = json.Unmarshal(b, i); err != nil {
		return fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	i.Header = r.Header.Get("X-Header")
	if err := i.Query.UnmarshalText([]byte(r.URL.Query().Get("in_query"))); err != nil {
		return fmt.Errorf("failed to decode in_query %q: %w", r.URL.Query().Get("in_query"), err)
	}

	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		i.Path = routeCtx.URLParam("in-path")
	} else {
		return errors.New("missing path params in context")
	}

	return nil
}
