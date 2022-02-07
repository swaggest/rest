package web_test

import (
	"context"
	"log"
	"net/http"

	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
)

// album represents data about a record album.
type album struct {
	ID     int     `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

func postAlbums() usecase.Interactor {
	u := usecase.NewIOI(new(album), new(album), func(ctx context.Context, input, output interface{}) error {
		log.Println("Creating album")

		return nil
	})
	u.SetTags("Album")

	return u
}

func ExampleDefaultService() {
	service := web.DefaultService()

	service.OpenAPI.Info.Title = "Albums API"
	service.OpenAPI.Info.WithDescription("This service provides API to manage albums.")
	service.OpenAPI.Info.Version = "v1.0.0"

	service.Post("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))

	log.Println("Starting service at http://localhost:8080")

	if err := http.ListenAndServe("localhost:8080", service); err != nil {
		log.Fatal(err)
	}
}
