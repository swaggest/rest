package web_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest-fasthttp/web"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

type albumID struct {
	ID     int    `path:"id"`
	Locale string `query:"locale"`
}

func albumByID() usecase.Interactor {
	u := usecase.NewIOI(new(albumID), new(album), func(ctx context.Context, input, output interface{}) error {
		return nil
	})
	u.SetTags("Album")

	return u
}

func TestDefaultService(t *testing.T) {
	var l []string

	service := web.DefaultService(
		func(s *web.Service, initialized bool) {
			l = append(l, fmt.Sprintf("one:%v", initialized))
		},
		func(s *web.Service, initialized bool) {
			l = append(l, fmt.Sprintf("two:%v", initialized))
		},
	)

	service.OpenAPI.Info.Title = "Albums API"
	service.OpenAPI.Info.WithDescription("This service provides API to manage albums.")
	service.OpenAPI.Info.Version = "v1.0.0"

	service.Delete("/albums/{id}", albumByID())
	service.Head("/albums/{id}", albumByID())
	service.Get("/albums/{id}", albumByID())
	service.Post("/albums", postAlbums(), fhttp.SuccessStatus(http.StatusCreated))
	service.Patch("/albums", postAlbums(), fhttp.SuccessStatus(http.StatusCreated))
	service.Put("/albums", postAlbums(), fhttp.SuccessStatus(http.StatusCreated))
	service.Trace("/albums", postAlbums(), fhttp.SuccessStatus(http.StatusCreated))
	service.Options("/albums", postAlbums(), fhttp.SuccessStatus(http.StatusCreated))
	service.Docs("/docs", func(title, schemaURL, basePath string) http.Handler {
		// Mount github.com/swaggest/swgui/v4emb.New here.
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {})
	})

	rc := &fasthttp.RequestCtx{
		Request:  fasthttp.Request{},
		Response: fasthttp.Response{},
	}
	rc.Request.SetRequestURI("http://localhost/docs/openapi.json")
	service.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assertjson.EqualMarshal(t, rc.Response.Body(), service.OpenAPI)

	expected, err := ioutil.ReadFile("_testdata/openapi.json")
	require.NoError(t, err)
	assertjson.EqualMarshal(t, expected, service.OpenAPI)

	assert.Equal(t, []string{"one:false", "two:false", "one:true", "two:true"}, l)
}
