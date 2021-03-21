package nethttp_test

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/usecase"
)

func ExampleSecurityMiddleware() {
	// Create router.
	r := chirouter.NewWrapper(chi.NewRouter())

	// Init API documentation schema.
	apiSchema := &openapi.Collector{}

	// Setup middlewares (non-documentary middlewares omitted for brevity).
	r.Use(
		nethttp.OpenAPIMiddleware(apiSchema), // Documentation collector.
	)

	// Configure an actual security middleware.
	serviceTokenAuth := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") != "<secret>" {
				http.Error(w, "Authentication failed.", http.StatusUnauthorized)

				return
			}

			h.ServeHTTP(w, req)
		})
	}

	// Configure documentation middleware to describe actual security middleware.
	serviceTokenDoc := nethttp.SecurityMiddleware(apiSchema, "serviceToken", openapi3.SecurityScheme{
		APIKeySecurityScheme: &openapi3.APIKeySecurityScheme{
			Name: "Authorization",
			In:   openapi3.APIKeySecuritySchemeInHeader,
		},
	})

	u := usecase.NewIOI(nil, nil, func(ctx context.Context, input, output interface{}) error {
		// Do something.
		return nil
	})

	// Add use case handler to router with security middleware.
	r.
		With(serviceTokenAuth, serviceTokenDoc). // Apply a pair of middlewares: actual security and documentation.
		Method(http.MethodGet, "/foo", nethttp.NewHandler(u))
}
