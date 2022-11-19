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

// SecurityMiddleware creates middleware to expose security scheme.
func SecurityMiddleware(
	c *openapi.Collector,
	name string,
	scheme openapi3.SecurityScheme,
	options ...func(*MiddlewareConfig),
) func(http.Handler) http.Handler {
	c.Reflector().SpecEns().ComponentsEns().SecuritySchemesEns().WithMapOfSecuritySchemeOrRefValuesItem(
		name,
		openapi3.SecuritySchemeOrRef{
			SecurityScheme: &scheme,
		},
	)

	cfg := MiddlewareConfig{}

	for _, o := range options {
		o(&cfg)
	}

	return securityMiddleware(c, name, cfg)
}

// HTTPBasicSecurityMiddleware creates middleware to expose Basic Security schema.
func HTTPBasicSecurityMiddleware(
	c *openapi.Collector,
	name, description string,
	options ...func(*MiddlewareConfig),
) func(http.Handler) http.Handler {
	hss := openapi3.HTTPSecurityScheme{}

	hss.WithScheme("basic")

	if description != "" {
		hss.WithDescription(description)
	}

	return SecurityMiddleware(c, name, openapi3.SecurityScheme{
		HTTPSecurityScheme: &hss,
	}, options...)
}

// HTTPBearerSecurityMiddleware creates middleware to expose HTTP Bearer security schema.
func HTTPBearerSecurityMiddleware(
	c *openapi.Collector,
	name, description, bearerFormat string,
	options ...func(*MiddlewareConfig),
) func(http.Handler) http.Handler {
	hss := openapi3.HTTPSecurityScheme{}

	hss.WithScheme("bearer")

	if bearerFormat != "" {
		hss.WithBearerFormat(bearerFormat)
	}

	if description != "" {
		hss.WithDescription(description)
	}

	return SecurityMiddleware(c, name, openapi3.SecurityScheme{
		HTTPSecurityScheme: &hss,
	}, options...)
}

// AnnotateOpenAPI applies OpenAPI annotation to relevant handlers.
func AnnotateOpenAPI(
	s *openapi.Collector,
	setup ...func(op *openapi3.Operation) error,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if IsWrapperChecker(next) {
			return next
		}

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

// SecurityResponse is a security middleware option to customize response structure and status.
func SecurityResponse(structure interface{}, httpStatus int) func(config *MiddlewareConfig) {
	return func(config *MiddlewareConfig) {
		config.ResponseStructure = structure
		config.ResponseStatus = httpStatus
	}
}

// MiddlewareConfig defines security middleware options.
type MiddlewareConfig struct {
	// ResponseStructure declares structure that is used for unauthorized message, default rest.ErrResponse{}.
	ResponseStructure interface{}

	// ResponseStatus declares HTTP status code that is used for unauthorized message, default http.StatusUnauthorized.
	ResponseStatus int
}

func securityMiddleware(s *openapi.Collector, name string, cfg MiddlewareConfig) func(http.Handler) http.Handler {
	return AnnotateOpenAPI(s, func(op *openapi3.Operation) error {
		op.Security = append(op.Security, map[string][]string{name: {}})

		if cfg.ResponseStatus == 0 {
			cfg.ResponseStatus = http.StatusUnauthorized
		}

		if cfg.ResponseStructure == nil {
			cfg.ResponseStructure = rest.ErrResponse{}
		}

		return s.Reflector().SetJSONResponse(op, cfg.ResponseStructure, cfg.ResponseStatus)
	})
}
