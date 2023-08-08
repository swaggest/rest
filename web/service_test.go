package web_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
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

	service := web.NewService(
		openapi3.NewReflector(),
		func(s *web.Service) { l = append(l, "one") },
		func(s *web.Service) { l = append(l, "two") },
	)

	service.OpenAPISchema().SetTitle("Albums API")
	service.OpenAPISchema().SetDescription("This service provides API to manage albums.")
	service.OpenAPISchema().SetVersion("v1.0.0")

	service.Delete("/albums/{id}", albumByID())
	service.Head("/albums/{id}", albumByID())
	service.Get("/albums/{id}", albumByID())
	service.Post("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
	service.Patch("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
	service.Put("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
	service.Trace("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
	service.Options("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
	service.Docs("/docs", func(title, schemaURL, basePath string) http.Handler {
		// Mount github.com/swaggest/swgui/v4emb.New here.
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {})
	})

	rw := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "http://localhost/docs/openapi.json", nil)
	require.NoError(t, err)
	service.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
	assertjson.EqualMarshal(t, rw.Body.Bytes(), service.OpenAPISchema())

	expected, err := ioutil.ReadFile("_testdata/openapi.json")
	require.NoError(t, err)
	assertjson.EqualMarshal(t, expected, service.OpenAPISchema())

	assert.Equal(t, []string{"one", "two"}, l)
}
