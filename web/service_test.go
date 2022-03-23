package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
)

func TestDefaultService(t *testing.T) {
	service := web.DefaultService()

	service.OpenAPI.Info.Title = "Albums API"
	service.OpenAPI.Info.WithDescription("This service provides API to manage albums.")
	service.OpenAPI.Info.Version = "v1.0.0"

	service.Delete("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
	service.Head("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
	service.Get("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))
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
	assertjson.EqualMarshal(t, rw.Body.Bytes(), service.OpenAPI)
}
