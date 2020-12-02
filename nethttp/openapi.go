package nethttp

import (
	"net/http"

	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/openapi"
)

// OpenAPIMiddleware reads info and adds validation to handler.
func OpenAPIMiddleware(s *openapi.Collector) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		var (
			withRoute rest.HandlerWithRoute
			handler   *Handler
		)

		if !HandlerAs(h, &withRoute) || !HandlerAs(h, &handler) {
			return h
		}

		err := s.Collect(
			withRoute.RouteMethod(),
			withRoute.RoutePattern(),
			handler.UseCase(),
			handler.HandlerTrait,
			handler.OperationAnnotations...,
		)
		if err != nil {
			panic(err)
		}

		return h
	}
}

// HTTPBasicSecurityMiddleware creates middleware to expose Basic Security schema.
func HTTPBasicSecurityMiddleware(c *openapi.Collector, name, description string) func(http.Handler) http.Handler {
	hss := openapi3.HTTPSecurityScheme{}

	hss.WithScheme("basic")

	if description != "" {
		hss.WithDescription(description)
	}

	c.Reflector().SpecEns().ComponentsEns().SecuritySchemesEns().WithMapOfSecuritySchemeOrRefValuesItem(
		name,
		openapi3.SecuritySchemeOrRef{
			SecurityScheme: &openapi3.SecurityScheme{
				HTTPSecurityScheme: &hss,
			},
		},
	)

	return securityMiddleware(c, name)
}

// HTTPBearerSecurityMiddleware creates middleware to expose HTTP Bearer security schema.
func HTTPBearerSecurityMiddleware(
	c *openapi.Collector, name, description, bearerFormat string,
) func(http.Handler) http.Handler {
	hss := openapi3.HTTPSecurityScheme{}

	hss.WithScheme("bearer")

	if bearerFormat != "" {
		hss.WithBearerFormat(bearerFormat)
	}

	if description != "" {
		hss.WithDescription(description)
	}

	c.Reflector().SpecEns().ComponentsEns().SecuritySchemesEns().WithMapOfSecuritySchemeOrRefValuesItem(
		name,
		openapi3.SecuritySchemeOrRef{
			SecurityScheme: &openapi3.SecurityScheme{
				HTTPSecurityScheme: &hss,
			},
		},
	)

	return securityMiddleware(c, name)
}

// AnnotateOpenAPI applies OpenAPI annotation to relevant handlers.
func AnnotateOpenAPI(
	s *openapi.Collector,
	setup ...func(op *openapi3.Operation) error,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		var withRoute rest.HandlerWithRoute

		if HandlerAs(next, &withRoute) {
			s.Annotate(
				withRoute.RouteMethod(),
				withRoute.RoutePattern(),
				setup...,
			)
		}

		return next
	}
}

func securityMiddleware(s *openapi.Collector, name string) func(http.Handler) http.Handler {
	return AnnotateOpenAPI(s, func(op *openapi3.Operation) error {
		op.Security = append(op.Security, map[string][]string{name: {}})

		return s.Reflector().SetJSONResponse(op, rest.ErrResponse{}, http.StatusUnauthorized)
	})
}
