// Package web provides default facades for web service bootstrap.
package web

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
)

// DefaultService initializes router and other basic components of web service.
//
// Provided functional options are invoked twice, before and after initialization.
func DefaultService(options ...func(s *Service, initialized bool)) *Service {
	s := Service{}

	for _, option := range options {
		option(&s, false)
	}

	if s.OpenAPI == nil {
		s.OpenAPI = &openapi3.Spec{Openapi: "3.0.3"}
	}

	// Init API documentation schema.
	if s.OpenAPICollector == nil {
		c := &openapi.Collector{}

		c.DefaultSuccessResponseContentType = response.DefaultSuccessResponseContentType
		c.DefaultErrorResponseContentType = response.DefaultErrorResponseContentType

		s.OpenAPICollector = c
		s.OpenAPICollector.Reflector().Spec = s.OpenAPI
	}

	if s.Wrapper == nil {
		s.Wrapper = chirouter.NewWrapper(chi.NewRouter())
	}

	if s.DecoderFactory == nil {
		decoderFactory := request.NewDecoderFactory()
		decoderFactory.ApplyDefaults = true
		decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

		s.DecoderFactory = decoderFactory
	}

	validatorFactory := jsonschema.NewFactory(s.OpenAPICollector, s.OpenAPICollector)
	s.ResponseValidatorFactory = validatorFactory

	if s.PanicRecoveryMiddleware == nil {
		s.PanicRecoveryMiddleware = middleware.Recoverer
	}

	// Setup middlewares.
	s.Wrapper.Wrap(
		s.PanicRecoveryMiddleware,                     // Panic recovery.
		nethttp.OpenAPIMiddleware(s.OpenAPICollector), // Documentation collector.
		request.DecoderMiddleware(s.DecoderFactory),   // Request decoder setup.
		request.ValidatorMiddleware(validatorFactory), // Request validator setup.
		response.EncoderMiddleware,                    // Response encoder setup.
	)

	for _, option := range options {
		option(&s, true)
	}

	return &s
}

// Service keeps instrumented router and documentation collector.
type Service struct {
	*chirouter.Wrapper

	PanicRecoveryMiddleware func(handler http.Handler) http.Handler // Default is middleware.Recoverer.
	OpenAPI                 *openapi3.Spec
	OpenAPICollector        *openapi.Collector
	DecoderFactory          *request.DecoderFactory

	// Response validation is not enabled by default for its less justifiable performance impact.
	// This field is populated so that response.ValidatorMiddleware(s.ResponseValidatorFactory) can be
	// added to service via Wrap.
	ResponseValidatorFactory rest.ResponseValidatorFactory
}

// Docs adds the route `pattern` that serves API documentation with Swagger UI.
//
// Swagger UI should be provided by `swgui` handler constructor, you can use one of these functions
//
//	github.com/swaggest/swgui/v4emb.New
//	github.com/swaggest/swgui/v4cdn.New
//	github.com/swaggest/swgui/v4.New
//	github.com/swaggest/swgui/v3emb.New
//	github.com/swaggest/swgui/v3cdn.New
//	github.com/swaggest/swgui/v3.New
//
// or create your own.
func (s *Service) Docs(pattern string, swgui func(title, schemaURL, basePath string) http.Handler) {
	pattern = strings.TrimRight(pattern, "/")
	s.Method(http.MethodGet, pattern+"/openapi.json", s.OpenAPICollector)
	s.Mount(pattern, swgui(s.OpenAPI.Info.Title, pattern+"/openapi.json", pattern))
}
