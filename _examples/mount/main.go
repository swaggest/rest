package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
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

func service() *web.Service {
	s := web.NewService(openapi3.NewReflector())
	s.OpenAPISchema().SetTitle("Security and Mount Example")

	apiV1 := web.NewService(openapi3.NewReflector())
	apiV2 := web.NewService(openapi3.NewReflector())

	apiV1.Wrap(
		middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"}),
		nethttp.HTTPBasicSecurityMiddleware(s.OpenAPICollector, "Admin", "Admin access"),
		nethttp.OpenAPIAnnotationsMiddleware(s.OpenAPICollector, func(oc openapi.OperationContext) error {
			oc.SetTags(append(oc.Tags(), "V1")...)
			return nil
		}),
	)
	apiV1.Post("/sum", sum())
	apiV1.Post("/mul", mul())

	apiV2.Wrap(
		// No auth for V2.

		nethttp.OpenAPIAnnotationsMiddleware(s.OpenAPICollector, func(oc openapi.OperationContext) error {
			oc.SetTags(append(oc.Tags(), "V2")...)
			return nil
		}),
	)
	apiV2.Post("/summarization", sum())
	apiV2.Post("/multiplication", mul())

	s.Mount("/api/v1", apiV1)
	s.Mount("/api/v2", apiV2)
	s.Docs("/api/docs", swgui.New)

	// Blanket handler, for example to serve static content.
	s.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("blanket handler got a request: " + r.URL.String()))
	}))

	return s
}

func main() {
	fmt.Println("Swagger UI at http://localhost:8010/api/docs.")
	if err := http.ListenAndServe("localhost:8010", service()); err != nil {
		log.Fatal(err)
	}
}
