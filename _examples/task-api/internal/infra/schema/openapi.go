// Package schema instruments OpenAPI schema.
package schema

import (
	"github.com/swaggest/rest/openapi"
)

// SetupOpenAPICollector sets up API documentation collector.
func SetupOpenAPICollector(apiSchema *openapi.Collector) {
	apiSchema.SpecSchema().SetTitle("Tasks Service")
	apiSchema.SpecSchema().SetDescription("This example service manages tasks.")
	apiSchema.SpecSchema().SetVersion("1.2.3")
}
