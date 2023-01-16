package web_test

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
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
	Locale string  `query:"locale"`
}

func postAlbums() usecase.Interactor {
	u := usecase.NewIOI(new(album), new(album), func(ctx context.Context, input, output any) error {
		log.Println("Creating album")

		return nil
	})
	u.SetTags("Album")

	return u
}

func ExampleDefaultService() {
	// Service initializes router with required middlewares.
	service := web.DefaultService()

	// It allows OpenAPI configuration.
	service.OpenAPI.Info.Title = "Albums API"
	service.OpenAPI.Info.WithDescription("This service provides API to manage albums.")
	service.OpenAPI.Info.Version = "v1.0.0"

	// Additional middlewares can be added.
	service.Use(
		middleware.StripSlashes,

		// cors.AllowAll().Handler, // "github.com/rs/cors", 3rd-party CORS middleware can also be configured here.
	)

	service.Wrap()

	// Use cases can be mounted using short syntax .<Method>(...).
	service.Post("/albums", postAlbums(), nethttp.SuccessStatus(http.StatusCreated))

	log.Println("Starting service at http://localhost:8080")

	//#nosec:G114
	if err := http.ListenAndServe("localhost:8080", service); err != nil {
		log.Fatal(err)
	}
}
