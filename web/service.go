// Package web provides default facades for web service bootstrap.
package web

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
)

// NewService initializes router and other basic components of web service.
func NewService(refl oapi.Reflector, options ...func(s *Service)) *Service {
	s := Service{}

	for _, option := range options {
		option(&s)
	}

	// Init API documentation schema.
	if s.OpenAPICollector == nil {
		c := openapi.NewCollector(refl)

		c.DefaultSuccessResponseContentType = response.DefaultSuccessResponseContentType
		c.DefaultErrorResponseContentType = response.DefaultErrorResponseContentType

		s.OpenAPICollector = c
	}

	if s.Wrapper == nil {
		s.Wrapper = chirouter.NewWrapper(chi.NewRouter())
	}

	if s.DecoderFactory == nil {
		decoderFactory := request.NewDecoderFactory()
		decoderFactory.ApplyDefaults = true
		decoderFactory.JSONSchemaReflector = s.OpenAPICollector.Refl().JSONSchemaReflector()
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

	return &s
}

// DefaultService initializes router and other basic components of web service.
//
// Provided functional options are invoked twice, before and after initialization.
//
// Deprecated: use NewService.
func DefaultService(options ...func(s *Service, initialized bool)) *Service {
	s := NewService(openapi3.NewReflector(), func(s *Service) {
		for _, o := range options {
			o(s, false)
		}
	})

	if r3, ok := s.OpenAPIReflector().(*openapi3.Reflector); ok && s.OpenAPI == nil {
		s.OpenAPI = r3.Spec
	}

	for _, o := range options {
		o(s, true)
	}

	return s
}

// Service keeps instrumented router and documentation collector.
type Service struct {
	*chirouter.Wrapper

	PanicRecoveryMiddleware func(handler http.Handler) http.Handler // Default is middleware.Recoverer.

	// Deprecated: use OpenAPISchema().
	OpenAPI *openapi3.Spec

	OpenAPICollector *openapi.Collector
	DecoderFactory   *request.DecoderFactory

	// Response validation is not enabled by default for its less justifiable performance impact.
	// This field is populated so that response.ValidatorMiddleware(s.ResponseValidatorFactory) can be
	// added to service via Wrap.
	ResponseValidatorFactory rest.ResponseValidatorFactory

	// AddHeadToGet is an option to enable HEAD method for each usecase added with Service.Get.
	AddHeadToGet bool
}

// OpenAPISchema returns OpenAPI schema.
//
// Returned value can be type asserted to *openapi3.Spec, *openapi31.Spec or marshaled.
func (s *Service) OpenAPISchema() oapi.SpecSchema {
	return s.OpenAPICollector.SpecSchema()
}

// OpenAPIReflector returns OpenAPI structure reflector for customizations.
func (s *Service) OpenAPIReflector() oapi.Reflector {
	return s.OpenAPICollector.Refl()
}

// Delete adds the route `pattern` that matches a DELETE http method to invoke use case interactor.
func (s *Service) Delete(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.Method(http.MethodDelete, pattern, nethttp.NewHandler(uc, options...))
}

// Get adds the route `pattern` that matches a GET http method to invoke use case interactor.
//
// If Service.AddHeadToGet is enabled, it also adds a HEAD method.
func (s *Service) Get(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	if s.AddHeadToGet {
		s.HeadGet(pattern, uc, options...)

		return
	}

	h := nethttp.NewHandler(uc, options...)
	s.Method(http.MethodGet, pattern, h)
}

// Head adds the route `pattern` that matches a HEAD http method to invoke use case interactor.
func (s *Service) Head(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.Method(http.MethodHead, pattern, nethttp.NewHandler(uc, options...))
}

// HeadGet adds the route `pattern` that matches a HEAD/GET http method to invoke use case interactor.
//
// Response body is automatically ignored.
func (s *Service) HeadGet(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	h := nethttp.NewHandler(uc, options...)
	s.Method(http.MethodGet, pattern, h)
	s.Method(http.MethodHead, pattern, h)
}

// Options adds the route `pattern` that matches a OPTIONS http method to invoke use case interactor.
func (s *Service) Options(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.Method(http.MethodOptions, pattern, nethttp.NewHandler(uc, options...))
}

// Patch adds the route `pattern` that matches a PATCH http method to invoke use case interactor.
func (s *Service) Patch(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.Method(http.MethodPatch, pattern, nethttp.NewHandler(uc, options...))
}

// Post adds the route `pattern` that matches a POST http method to invoke use case interactor.
func (s *Service) Post(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.Method(http.MethodPost, pattern, nethttp.NewHandler(uc, options...))
}

// Put adds the route `pattern` that matches a PUT http method to invoke use case interactor.
func (s *Service) Put(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.Method(http.MethodPut, pattern, nethttp.NewHandler(uc, options...))
}

// Trace adds the route `pattern` that matches a TRACE http method to invoke use case interactor.
func (s *Service) Trace(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.Method(http.MethodTrace, pattern, nethttp.NewHandler(uc, options...))
}

// OnNotFound registers usecase interactor as a handler for not found conditions.
func (s *Service) OnNotFound(uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.NotFound(s.HandlerFunc(nethttp.NewHandler(uc, options...)))
}

// OnMethodNotAllowed registers usecase interactor as a handler for method not allowed conditions.
func (s *Service) OnMethodNotAllowed(uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	s.MethodNotAllowed(s.HandlerFunc(nethttp.NewHandler(uc, options...)))
}

// Docs adds the route `pattern` that serves API documentation with Swagger UI.
//
// Swagger UI should be provided by `swgui` handler constructor, you can use one of these functions
//
//	github.com/swaggest/swgui/v5emb.New
//	github.com/swaggest/swgui/v5cdn.New
//	github.com/swaggest/swgui/v5.New
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
	s.Mount(pattern, swgui(s.OpenAPISchema().Title(), pattern+"/openapi.json", pattern))
}
