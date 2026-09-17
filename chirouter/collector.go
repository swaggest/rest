package chirouter

import (
	"net/http"

	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
)

// OpenAPICollector is a wrapper for openapi.Collector tailored to walk chi router.
type OpenAPICollector struct {
	// Collector is an actual OpenAPI collector.
	Collector *openapi.Collector

	// OperationExtractor allows flexible extraction of OpenAPI information.
	OperationExtractor func(h http.Handler) func(oc oapi.OperationContext) error

	docs nethttp.RouteDocs
}

// NewOpenAPICollector creates route walker for chi, that collects OpenAPI operations.
func NewOpenAPICollector(r oapi.Reflector) *OpenAPICollector {
	return &OpenAPICollector{
		Collector: openapi.NewCollector(r),
	}
}

// Describe attaches OpenAPI documentation to a http.HandlerFunc, keyed by the handler's own
// identity rather than by its route. Unlike Collector.AnnotateOperation, which is keyed by a
// separately maintained method+pattern string that can drift from the actual route, Describe
// keeps the route registration itself as the single source of truth.
//
// Wrap the handler with it right where it is registered, e.g. router.Get(pattern, c.Describe(h, setup)).
func (dc *OpenAPICollector) Describe(h http.HandlerFunc, setup func(oc oapi.OperationContext) error) http.HandlerFunc {
	return dc.docs.Describe(h, setup)
}

// Walker walks chi route tree and collects OpenAPI information.
//
// Use it as chi.WalkFunc with chi.Walk.
func (dc *OpenAPICollector) Walker(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
	if handler == nil {
		return nil
	}

	preparer := dc.docs.Preparer(handler, dc.OperationExtractor)

	return nethttp.CollectRouteOperation(dc.Collector, method, route, preparer)
}
