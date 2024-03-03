// Package main implements an example where two versioned API revisions are mounted into root web service
// and are available through a service selector in Swagger UI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	swg "github.com/swaggest/swgui"
	swgui "github.com/swaggest/swgui/v5emb"
	"github.com/swaggest/usecase"
)

func main() {
	fmt.Println("Swagger UI at http://localhost:8010/api/docs.")
	if err := http.ListenAndServe("localhost:8010", service()); err != nil {
		log.Fatal(err)
	}
}

func service() *web.Service {
	// Creating root service, to host versioned APIs.
	s := web.NewService(openapi3.NewReflector())
	s.OpenAPISchema().SetTitle("Security and Mount Example")

	// Each versioned API is exposed with its own OpenAPI schema.
	v1r := openapi3.NewReflector()
	v1r.SpecEns().WithServers(openapi3.Server{URL: "/api/v1/"}).WithInfo(openapi3.Info{Title: "My API of version 1"})
	apiV1 := web.NewService(v1r)

	v2r := openapi3.NewReflector()
	v2r.SpecEns().WithServers(openapi3.Server{URL: "/api/v2/"})
	apiV2 := web.NewService(v2r)

	// Versioned APIs may or may not have their own middlewares and wraps.
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
	// Once all API use cases are added, schema can be served too.
	apiV1.Method(http.MethodGet, "/openapi.json", specHandler(apiV1.OpenAPICollector.SpecSchema()))

	apiV2.Post("/summarization", sum())
	apiV2.Post("/multiplication", mul())
	apiV2.Method(http.MethodGet, "/openapi.json", specHandler(apiV2.OpenAPICollector.SpecSchema()))

	// Prepared versioned API services are mounted with their base URLs into root service.
	s.Mount("/api/v1", apiV1)
	s.Mount("/api/v2", apiV2)

	// Root docs needs a bit of hackery to expose versioned APIs as separate services.
	s.Docs("/api/docs", swgui.NewWithConfig(swg.Config{
		ShowTopBar: true,
		SettingsUI: map[string]string{
			// When "urls" are configured, Swagger UI ignores "url" and switches to multi API mode.
			"urls": `[
	{"url": "/api/v1/openapi.json", "name": "APIv1"}, 
	{"url": "/api/v2/openapi.json", "name": "APIv2"}
]`,
			`"urls.primaryName"`: `"APIv2"`, // Using APIv2 as default.
		},
	}))

	// Blanket handler, for example to serve static content.
	s.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("blanket handler got a request: " + r.URL.String()))
	}))

	return s
}

func specHandler(s openapi.SpecSchema) http.Handler {
	j, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(j)
	})
}

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
