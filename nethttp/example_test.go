package nethttp_test

import (
	"bytes"
	"context"
	"net/http"

	"github.com/swaggest/fchi"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

func ExampleSecurityMiddleware() {
	// Create router.
	r := chirouter.NewWrapper(fchi.NewRouter())

	// Init API documentation schema.
	apiSchema := &openapi.Collector{}

	// Setup middlewares (non-documentary middlewares omitted for brevity).
	r.Wrap(
		nethttp.OpenAPIMiddleware(apiSchema), // Documentation collector.
	)

	// Configure an actual security middleware.
	serviceTokenAuth := func(h fchi.Handler) fchi.Handler {
		return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			if !bytes.Equal(rc.Request.Header.Peek("Authorization"), []byte("<secret>")) {
				fchi.Error(rc, "Authentication failed.", http.StatusUnauthorized)

				return
			}

			h.ServeHTTP(ctx, rc)
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
