package gorillamux

import (
	"net/http"

	"github.com/gorilla/mux"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
)

type DocsCollector struct {
	AllowIncomplete bool
	OnError         func(err error)
	Collector       *openapi.Collector
}

func NewOpenAPICollector() *DocsCollector {
	c := &openapi.Collector{}

	c.Reflector().SpecEns().Info.
		WithTitle("Test Server").
		WithVersion("v1.2.3").
		WithDescription("Provides API over HTTP")

	return &DocsCollector{
		Collector: c,
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
		methods = []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	}

	var openAPIPreparer OperationPreparer

	nethttp.HandlerAs(handler, &openAPIPreparer)

	for _, method := range methods {
		if err := dc.Collector.CollectOperation(method, path, func(oc oapi.OperationContext) error {
			if openAPIPreparer != nil {
				return openAPIPreparer.SetupOpenAPIOperation(oc)
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
