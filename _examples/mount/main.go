package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v4emb"
	"github.com/swaggest/usecase"
)

func mul() usecase.Interactor {
	return usecase.NewInteractor(func(ctx context.Context, input []int, output *int) error {
		*output = 1

		for _, v := range input {
			*output *= v
		}

		return nil
	})
}

func sum() usecase.Interactor {
	return usecase.NewInteractor(func(ctx context.Context, input []int, output *int) error {
		for _, v := range input {
			*output += v
		}

		return nil
	})
}

func main() {
	service := web.DefaultService()
	service.OpenAPI.Info.Title = "Security and Mount Example"

	service.Post("/mul", mul())
	service.Post("/sum", sum())

	sub := web.DefaultService()

	sub.Wrap(
		middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"}),
		nethttp.HTTPBasicSecurityMiddleware(service.OpenAPICollector, "Admin", "Admin access"),
	)

	sub.Post("/sum2", sum())
	sub.Post("/mul2", mul())

	service.Mount("/restricted", sub)

	service.Docs("/docs", swgui.New)

	fmt.Println("Swagger UI at http://localhost:8010/docs.")
	if err := http.ListenAndServe("localhost:8010", service); err != nil {
		log.Fatal(err)
	}
}
