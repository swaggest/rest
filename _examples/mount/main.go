package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v5emb"
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
	service.OpenAPISchema().SetTitle("Security and Mount Example")

	apiV1 := web.DefaultService()

	apiV1.Wrap(
		middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"}),
		nethttp.HTTPBasicSecurityMiddleware(service.OpenAPICollector, "Admin", "Admin access"),
	)

	apiV1.Post("/sum", sum())
	apiV1.Post("/mul", mul())

	service.Mount("/api/v1", apiV1)
	service.Docs("/api/v1/docs", swgui.New)

	// Blanket handler, for example to serve static content.
	service.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("blanket handler got a request: " + r.URL.String()))
	}))

	fmt.Println("Swagger UI at http://localhost:8010/api/v1/docs.")
	if err := http.ListenAndServe("localhost:8010", service); err != nil {
		log.Fatal(err)
	}
}
