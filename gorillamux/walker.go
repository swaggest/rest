package gorillamux

import (
	"net/http"

	"github.com/gorilla/mux"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
)

type DocsCollector struct {
	OnError        func(err error)
	Collector      *openapi.Collector
	DefaultMethods []string
}

func NewOpenAPICollector(r oapi.Reflector) *DocsCollector {
	c := openapi.NewCollector(r)

	return &DocsCollector{
		Collector: c,
		DefaultMethods: []string{
			http.MethodHead, http.MethodGet, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete,
		},
	}
}

type OperationPreparer interface {
	SetupOpenAPIOperation(oc oapi.OperationContext) error
}

func (dc *DocsCollector) Walker(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
	handler := route.GetHandler()
	// Path is critical info, skipping route if there is a problem with path.
	path, err := route.GetPathTemplate()
	if err != nil {
		dc.onError(err)

		return nil
	}

	methods, err := route.GetMethods()
	dc.onError(err)

	if len(methods) == 0 {
		methods = dc.DefaultMethods
	}

	var openAPIPreparer OperationPreparer

	nethttp.HandlerAs(handler, &openAPIPreparer)

	for _, method := range methods {
		if err := dc.Collector.CollectOperation(method, path, func(oc oapi.OperationContext) error {
			if openAPIPreparer != nil {
				return openAPIPreparer.SetupOpenAPIOperation(oc)
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

			oc.AddRespStructure(nil, func(cu *oapi.ContentUnit) {
				cu.ContentType = "text/html"
			})

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (dc *DocsCollector) onError(err error) {
	if dc.OnError != nil && err != nil {
		dc.OnError(err)
	}
}