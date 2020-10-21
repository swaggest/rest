// Package schema instruments OpenAPI schema.
package schema

import (
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/openapi"
)

// NewOpenAPICollector creates API documentation collector.
func NewOpenAPICollector() *openapi.Collector {
	apiSchema := openapi.Collector{}
	serviceInfo := openapi3.Info{}
	serviceInfo.
		WithTitle("Tasks Service").
		WithDescription("This example service manages tasks.").
		WithVersion("1.2.3")

	apiSchema.Reflector().SpecEns().WithInfo(serviceInfo)

	return &apiSchema
}
