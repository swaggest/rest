package gorillamux

import (
	"net/http"

	"github.com/gorilla/mux"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
)

// OpenAPICollector is a wrapper for openapi.Collector tailored to walk gorilla/mux router.
type OpenAPICollector struct {
	// Collector is an actual OpenAPI collector.
	Collector *openapi.Collector

	// DefaultMethods list is used when handler serves all methods.
	DefaultMethods []string

	// OperationExtractor allows flexible extraction of OpenAPI information.
	OperationExtractor func(h http.Handler) func(oc oapi.OperationContext) error

	// Host filters routes by host, gorilla/mux can serve different handlers at
	// same method, paths with different hosts. This can not be expressed with a single
	// OpenAPI document.
	Host string
}

// NewOpenAPICollector creates route walker for gorilla/mux, that collects OpenAPI operations.
func NewOpenAPICollector(r oapi.Reflector) *OpenAPICollector {
	c := openapi.NewCollector(r)

	return &OpenAPICollector{
		Collector: c,
		DefaultMethods: []string{
			http.MethodHead, http.MethodGet, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete,
		},
	}
}

// OpenAPIPreparer defines http.Handler with OpenAPI information.
type OpenAPIPreparer interface {
	SetupOpenAPIOperation(oc oapi.OperationContext) error
}

// Describe attaches OpenAPI documentation to a http.HandlerFunc. Unlike Collector.AnnotateOperation,
// which is keyed by a separately maintained method+pattern string that can drift from the actual
// route, Describe keeps the route registration itself as the single source of truth.
//
// The result implements http.Handler, e.g. router.Handle(pattern, c.Describe(h, setup)).Methods(http.MethodGet).
func (dc *OpenAPICollector) Describe(h http.HandlerFunc, setup func(oc oapi.OperationContext) error) http.Handler {
	return nethttp.Describe(h, setup)
}

// Walker walks route tree and collects OpenAPI information.
func (dc *OpenAPICollector) Walker(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
	handler := route.GetHandler()

	if handler == nil {
		return nil
	}

	// Path is critical info, skipping route if there is a problem with path.
	path, err := route.GetPathTemplate()
	if err != nil && path == "" {
		return nil
	}

	host, err := route.GetHostTemplate()
	if (err == nil && host != dc.Host) || // There is host, but different.
		(err != nil && dc.Host != "") { // There is no host, but should be.
		return nil
	}

	methods, err := route.GetMethods()
	if err != nil {
		methods = dc.DefaultMethods
	}

	preparer := nethttp.Preparer(handler, dc.OperationExtractor)

	for _, method := range methods {
		if err := nethttp.CollectRouteOperation(dc.Collector, method, path, preparer); err != nil {
			return err
		}
	}

	return nil
}
