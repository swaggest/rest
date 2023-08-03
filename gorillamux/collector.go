package gorillamux

import (
	"net/http"

	"github.com/gorilla/mux"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
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

type preparerFunc func(oc oapi.OperationContext) error

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

	var (
		openAPIPreparer OpenAPIPreparer
		preparer        preparerFunc
	)

	if nethttp.HandlerAs(handler, &openAPIPreparer) {
		preparer = openAPIPreparer.SetupOpenAPIOperation
	} else if dc.OperationExtractor != nil {
		preparer = dc.OperationExtractor(handler)
	}

	for _, method := range methods {
		if err := dc.Collector.CollectOperation(method, path, dc.collect(method, path, preparer)); err != nil {
			return err
		}
	}

	return nil
}

func (dc *OpenAPICollector) collect(method, path string, preparer preparerFunc) preparerFunc {
	return func(oc oapi.OperationContext) error {
		// Do not apply default parameters to not conflict with custom preparer.
		if preparer != nil {
			return preparer(oc)
		}

		// Do not apply default parameters to not conflict with custom annotation.
		if dc.Collector.HasAnnotation(method, path) {
			return nil
		}

		_, _, pathItems, err := oapi.SanitizeMethodPath(method, path)
		if err != nil {
			return err
		}

		if len(pathItems) > 0 {
			if o3, ok := oc.(openapi3.OperationExposer); ok {
				op := o3.Operation()

				for _, p := range pathItems {
					param := openapi3.ParameterOrRef{}
					param.WithParameter(openapi3.Parameter{
						Name: p,
						In:   openapi3.ParameterInPath,
					})

					op.Parameters = append(op.Parameters, param)
				}
			}
		}

		oc.SetDescription("Information about this operation was obtained using only HTTP method and path pattern. " +
			"It may be incomplete and/or inaccurate.")
		oc.SetTags("Incomplete")
		oc.AddRespStructure(nil, func(cu *oapi.ContentUnit) {
			cu.ContentType = "text/html"
		})

		return nil
	}
}
