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
// adapters can implement the same collection flow with RouteDocs and CollectRouteOperation.
type RoutePreparer interface {
	SetupOpenAPIOperation(oc oapi.OperationContext) error
}

// RoutePreparerFunc sets up OpenAPI operation.
type RoutePreparerFunc func(oc oapi.OperationContext) error

// RouteDocs holds handler-identity-keyed OpenAPI documentation, registered via Describe.
type RouteDocs struct {
	byHandler map[uintptr]RoutePreparerFunc
}

// Describe attaches OpenAPI documentation to a http.HandlerFunc, keyed by the handler's own
// identity rather than by its route. Unlike Collector.AnnotateOperation, which is keyed by a
// separately maintained method+pattern string that can drift from the actual route, Describe
// keeps the route registration itself as the single source of truth.
func (d *RouteDocs) Describe(h http.HandlerFunc, setup RoutePreparerFunc) http.HandlerFunc {
	if d.byHandler == nil {
		d.byHandler = make(map[uintptr]RoutePreparerFunc)
	}

	d.byHandler[reflect.ValueOf(h).Pointer()] = setup

	return h
}

// Preparer resolves the preparer func for handler: interface implementation first,
// then identity-based Describe docs, then the extractor as a last resort.
func (d *RouteDocs) Preparer(
	handler http.Handler,
	extractor func(h http.Handler) func(oc oapi.OperationContext) error,
) RoutePreparerFunc {
	var (
		routePreparer RoutePreparer
		preparer      RoutePreparerFunc
	)

	if HandlerAs(handler, &routePreparer) {
		preparer = routePreparer.SetupOpenAPIOperation
	} else if hf, ok := handler.(http.HandlerFunc); ok {
		preparer = d.byHandler[reflect.ValueOf(hf).Pointer()]
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
