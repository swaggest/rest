package nethttp

import (
	"net/http"
	"reflect"

	"github.com/swaggest/jsonschema-go"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/rest/openapi"
)

// RoutePreparer defines http.Handler with OpenAPI information.
//
// It is used by router adapters (such as chirouter and gorillamux) that augment an existing
// http.Handler-based router instead of wrapping typed usecase interactors. Third-party router
// adapters can implement the same collection flow with Describe, Preparer and CollectRouteOperation.
type RoutePreparer interface {
	SetupOpenAPIOperation(oc oapi.OperationContext) error
}

// RoutePreparerFunc sets up OpenAPI operation.
type RoutePreparerFunc func(oc oapi.OperationContext) error

// describedHandler pairs a http.HandlerFunc with its own OpenAPI setup, so Preparer can
// recognize it the same way as a handler type that implements RoutePreparer directly.
//
// A map keyed by reflect.ValueOf(h).Pointer() would not work here: that pointer identifies the
// handler's compiled code, not a particular closure instance, so handlers built from the same
// func literal (e.g. in a loop, for programmatically registered routes) would collide and
// silently share one another's documentation.
type describedHandler struct {
	http.HandlerFunc

	setup RoutePreparerFunc
}

// SetupOpenAPIOperation implements RoutePreparer.
func (d describedHandler) SetupOpenAPIOperation(oc oapi.OperationContext) error {
	return d.setup(oc)
}

// Describe attaches OpenAPI documentation to a http.HandlerFunc. Unlike Collector.AnnotateOperation,
// which is keyed by a separately maintained method+pattern string that can drift from the actual
// route, Describe keeps the route registration itself as the single source of truth: it wraps the
// handler so Preparer recognizes it the same way as a handler that implements RoutePreparer
// directly.
//
// The result implements http.Handler, so register it with a method that accepts one
// (e.g. chi's Method/Handle, or gorilla/mux's Handle), not one that requires http.HandlerFunc.
func Describe(h http.HandlerFunc, setup RoutePreparerFunc) http.Handler {
	return describedHandler{HandlerFunc: h, setup: setup}
}

// Preparer resolves the preparer func for handler: interface implementation first (this is also
// how a handler wrapped with Describe is recognized), then the extractor as a last resort.
func Preparer(
	handler http.Handler,
	extractor func(h http.Handler) func(oc oapi.OperationContext) error,
) RoutePreparerFunc {
	var (
		routePreparer RoutePreparer
		preparer      RoutePreparerFunc
	)

	if HandlerAs(handler, &routePreparer) {
		preparer = routePreparer.SetupOpenAPIOperation
	}

	if preparer == nil && extractor != nil {
		preparer = extractor(handler)
	}

	return preparer
}

// CollectRouteOperation prepares and adds an OpenAPI operation, falling back to a best-effort
// "Incomplete" description (derived from method/path alone) when there is no preparer and no
// prior annotation.
func CollectRouteOperation(coll *openapi.Collector, method, path string, preparer RoutePreparerFunc) error {
	return coll.CollectOperation(method, path, func(oc oapi.OperationContext) error {
		// Do not apply default parameters to not conflict with custom preparer.
		if preparer != nil {
			return preparer(oc)
		}

		// Do not apply default parameters to not conflict with custom annotation.
		if coll.HasAnnotation(method, path) {
			return nil
		}

		_, _, pathItems, err := oapi.SanitizeMethodPath(method, path)
		if err != nil {
			return err
		}

		if len(pathItems) > 0 {
			req := jsonschema.Struct{}
			for _, p := range pathItems {
				req.Fields = append(req.Fields, jsonschema.Field{
					Name:  "F" + p,
					Tag:   reflect.StructTag(`path:"` + p + `"`),
					Value: "",
				})
			}

			oc.AddReqStructure(req)
		}

		oc.SetDescription("Information about this operation was obtained using only HTTP method and path pattern. " +
			"It may be incomplete and/or inaccurate.")
		oc.SetTags("Incomplete")
		oc.AddRespStructure(nil, func(cu *oapi.ContentUnit) {
			cu.ContentType = "text/html"
		})

		return nil
	})
}
