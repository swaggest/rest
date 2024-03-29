package nethttp

import (
	"net/http"

	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/openapi"
)

// OpenAPIMiddleware reads info and adds validation to handler.
func OpenAPIMiddleware(s *openapi.Collector) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		if IsWrapperChecker(h) {
			return h
		}

		var (
			withRoute rest.HandlerWithRoute
			handler   *Handler
		)

		if !HandlerAs(h, &withRoute) || !HandlerAs(h, &handler) {
			return h
		}

		var methods []string

		method := withRoute.RouteMethod()

		if method == "" {
			methods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}
		} else {
			methods = []string{method}
		}

		for _, m := range methods {
			err := s.CollectUseCase(
				m,
				withRoute.RoutePattern(),
				handler.UseCase(),
				handler.HandlerTrait,
			)
			if err != nil {
				panic(err)
			}
		}

		return h
	}
}

// AuthMiddleware creates middleware to expose security scheme.
func AuthMiddleware(
	c *openapi.Collector,
	name string,
	options ...func(*MiddlewareConfig),
) func(http.Handler) http.Handler {
	cfg := MiddlewareConfig{}

	for _, o := range options {
		o(&cfg)
	}

	return securityMiddleware(c, name, cfg)
}

// SecurityMiddleware creates middleware to expose security scheme.
//
// Deprecated: use AuthMiddleware.
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

// APIKeySecurityMiddleware creates middleware to expose API Key security schema.
func APIKeySecurityMiddleware(
	c *openapi.Collector,
	name string, fieldName string, fieldIn oapi.In, description string,
	options ...func(*MiddlewareConfig),
) func(http.Handler) http.Handler {
	c.SpecSchema().SetAPIKeySecurity(name, fieldName, fieldIn, description)

	return AuthMiddleware(c, name, options...)
}

// HTTPBasicSecurityMiddleware creates middleware to expose HTTP Basic security schema.
func HTTPBasicSecurityMiddleware(
	c *openapi.Collector,
	name, description string,
	options ...func(*MiddlewareConfig),
) func(http.Handler) http.Handler {
	c.SpecSchema().SetHTTPBasicSecurity(name, description)

	return AuthMiddleware(c, name, options...)
}

// HTTPBearerSecurityMiddleware creates middleware to expose HTTP Bearer security schema.
func HTTPBearerSecurityMiddleware(
	c *openapi.Collector,
	name, description, bearerFormat string,
	options ...func(*MiddlewareConfig),
) func(http.Handler) http.Handler {
	c.SpecSchema().SetHTTPBearerTokenSecurity(name, bearerFormat, description)

	return AuthMiddleware(c, name, options...)
}

// AnnotateOpenAPI applies OpenAPI annotation to relevant handlers.
//
// Deprecated: use OpenAPIAnnotationsMiddleware.
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

// OpenAPIAnnotationsMiddleware applies OpenAPI annotations to handlers.
func OpenAPIAnnotationsMiddleware(
	s *openapi.Collector,
	annotations ...func(oc oapi.OperationContext) error,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if IsWrapperChecker(next) {
			return next
		}

		var withRoute rest.HandlerWithRoute

		if HandlerAs(next, &withRoute) {
			method := withRoute.RouteMethod()
			pattern := withRoute.RoutePattern()

			s.AnnotateOperation(
				method,
				pattern,
				annotations...,
			)
		}

		return next
	}
}

func securityMiddleware(s *openapi.Collector, name string, cfg MiddlewareConfig) func(http.Handler) http.Handler {
	return OpenAPIAnnotationsMiddleware(s, func(oc oapi.OperationContext) error {
		oc.AddSecurity(name)

		if cfg.ResponseStatus == 0 {
			cfg.ResponseStatus = http.StatusUnauthorized
		}

		if cfg.ResponseStructure == nil {
			cfg.ResponseStructure = rest.ErrResponse{}
		}

		oc.AddRespStructure(cfg.ResponseStructure, func(cu *oapi.ContentUnit) {
			cu.HTTPStatus = cfg.ResponseStatus
		})

		return nil
	})
}
