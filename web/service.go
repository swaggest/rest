// Package web provides default facades for web service bootstrap.
package web

import (
	"net/http"
	"strings"

	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
	"github.com/swaggest/openapi-go/openapi3"
	rest2 "github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp/chirouter"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest-fasthttp/request"
	"github.com/swaggest/rest-fasthttp/response"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/usecase"
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
		s.OpenAPICollector = &openapi.Collector{}
		s.OpenAPICollector.Reflector().Spec = s.OpenAPI
	}

	if s.Wrapper == nil {
		s.Wrapper = chirouter.NewWrapper(fchi.NewRouter())
	}

	if s.DecoderFactory == nil {
		decoderFactory := request.NewDecoderFactory()
		decoderFactory.ApplyDefaults = true
		decoderFactory.SetDecoderFunc(rest2.ParamInPath, chirouter.PathToURLValues)

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
		fhttp.OpenAPIMiddleware(s.OpenAPICollector),   // Documentation collector.
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

	PanicRecoveryMiddleware func(handler fchi.Handler) fchi.Handler // Default is middleware.Recoverer.
	OpenAPI                 *openapi3.Spec
	OpenAPICollector        *openapi.Collector
	DecoderFactory          *request.DecoderFactory

	// Response validation is not enabled by default for its less justifiable performance impact.
	// This field is populated so that response.ValidatorMiddleware(s.ResponseValidatorFactory) can be
	// added to service via Wrap.
	ResponseValidatorFactory rest2.ResponseValidatorFactory
}

// Delete adds the route `pattern` that matches a DELETE http method to invoke use case interactor.
func (s *Service) Delete(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodDelete, pattern, fhttp.NewHandler(uc, options...))
}

// Get adds the route `pattern` that matches a GET http method to invoke use case interactor.
func (s *Service) Get(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodGet, pattern, fhttp.NewHandler(uc, options...))
}

// Head adds the route `pattern` that matches a HEAD http method to invoke use case interactor.
func (s *Service) Head(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodHead, pattern, fhttp.NewHandler(uc, options...))
}

// Options adds the route `pattern` that matches a OPTIONS http method to invoke use case interactor.
func (s *Service) Options(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodOptions, pattern, fhttp.NewHandler(uc, options...))
}

// Patch adds the route `pattern` that matches a PATCH http method to invoke use case interactor.
func (s *Service) Patch(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodPatch, pattern, fhttp.NewHandler(uc, options...))
}

// Post adds the route `pattern` that matches a POST http method to invoke use case interactor.
func (s *Service) Post(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodPost, pattern, fhttp.NewHandler(uc, options...))
}

// Put adds the route `pattern` that matches a PUT http method to invoke use case interactor.
func (s *Service) Put(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodPut, pattern, fhttp.NewHandler(uc, options...))
}

// Trace adds the route `pattern` that matches a TRACE http method to invoke use case interactor.
func (s *Service) Trace(pattern string, uc usecase.Interactor, options ...func(h *fhttp.Handler)) {
	s.Method(http.MethodTrace, pattern, fhttp.NewHandler(uc, options...))
}

// Docs adds the route `pattern` that serves API documentation with Swagger UI.
//
// Swagger UI should be provided by `swgui` handler constructor, you can use one of these functions
//   github.com/swaggest/swgui/v4emb.New
//   github.com/swaggest/swgui/v4cdn.New
//   github.com/swaggest/swgui/v4.New
//   github.com/swaggest/swgui/v3emb.New
//   github.com/swaggest/swgui/v3cdn.New
//   github.com/swaggest/swgui/v3.New
// or create your own.
func (s *Service) Docs(pattern string, swgui func(title, schemaURL, basePath string) http.Handler) {
	pattern = strings.TrimRight(pattern, "/")
	s.Method(http.MethodGet, pattern+"/openapi.json", fchi.Adapt(s.OpenAPICollector))
	s.Mount(pattern, fchi.Adapt(swgui(s.OpenAPI.Info.Title, pattern+"/openapi.json", pattern)))
}
