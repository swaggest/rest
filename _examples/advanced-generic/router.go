//go:build go1.18
// +build go1.18

package main

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/response/gzip"
	v4 "github.com/swaggest/swgui/v4"
)

func NewRouter() http.Handler {
	apiSchema := &openapi.Collector{}
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	apiSchema.Reflector().SpecEns().Info.Title = "Advanced Example"
	apiSchema.Reflector().SpecEns().Info.WithDescription("This app showcases a variety of features.")
	apiSchema.Reflector().SpecEns().Info.Version = "v1.2.3"
	apiSchema.Reflector().InterceptDefName(func(t reflect.Type, defaultDefName string) string {
		return strings.ReplaceAll(defaultDefName, "Generic", "")
	})

	r := chirouter.NewWrapper(chi.NewRouter())

	r.Use(
		middleware.Recoverer,                          // Panic recovery.
		nethttp.OpenAPIMiddleware(apiSchema),          // Documentation collector.
		request.DecoderMiddleware(decoderFactory),     // Request decoder setup.
		request.ValidatorMiddleware(validatorFactory), // Request validator setup.
		response.EncoderMiddleware,                    // Response encoder setup.

		// Response validator setup.
		//
		// It might be a good idea to disable this middleware in production to save performance,
		// but keep it enabled in dev/test/staging environments to catch logical issues.
		response.ValidatorMiddleware(validatorFactory),
		gzip.Middleware, // Response compression with support for direct gzip pass through.
	)

	// Annotations can be used to alter documentation of operation identified by method and path.
	apiSchema.Annotate(http.MethodPost, "/validation", func(op *openapi3.Operation) error {
		if op.Description != nil {
			*op.Description = *op.Description + " Custom annotation."
		}

		return nil
	})

	r.Method(http.MethodGet, "/query-object", nethttp.NewHandler(queryObject()))

	r.Method(http.MethodPost, "/file-upload", nethttp.NewHandler(fileUploader()))
	r.Method(http.MethodPost, "/file-multi-upload", nethttp.NewHandler(fileMultiUploader()))
	r.Method(http.MethodGet, "/json-param/{in-path}", nethttp.NewHandler(jsonParam()))
	r.Method(http.MethodPost, "/json-body/{in-path}", nethttp.NewHandler(jsonBody(),
		nethttp.SuccessStatus(http.StatusCreated)))
	r.Method(http.MethodPost, "/json-body-validation/{in-path}", nethttp.NewHandler(jsonBodyValidation()))
	r.Method(http.MethodPost, "/json-slice-body", nethttp.NewHandler(jsonSliceBody()))

	r.Method(http.MethodPost, "/json-map-body", nethttp.NewHandler(jsonMapBody(),
		// Annotate operation to add post processing if necessary.
		nethttp.AnnotateOperation(func(op *openapi3.Operation) error {
			op.WithDescription("Request with JSON object (map) body.")

			return nil
		})))

	r.Method(http.MethodGet, "/output-headers", nethttp.NewHandler(outputHeaders()))
	r.Method(http.MethodHead, "/output-headers", nethttp.NewHandler(outputHeaders()))
	r.Method(http.MethodGet, "/output-csv-writer", nethttp.NewHandler(outputCSVWriter(),
		nethttp.SuccessfulResponseContentType("text/csv; charset=utf-8")))

	r.Method(http.MethodPost, "/req-resp-mapping", nethttp.NewHandler(reqRespMapping(),
		nethttp.RequestMapping(new(struct {
			Val1 string `header:"X-Header"`
			Val2 int    `formData:"val2"`
		})),
		nethttp.ResponseHeaderMapping(new(struct {
			Val1 string `header:"X-Value-1"`
			Val2 int    `header:"X-Value-2"`
		})),
	))

	r.Method(http.MethodPost, "/validation", nethttp.NewHandler(validation()))

	// Type mapping is necessary to pass interface as structure into documentation.
	apiSchema.Reflector().AddTypeMapping(new(gzipPassThroughOutput), new(gzipPassThroughStruct))
	r.Method(http.MethodGet, "/gzip-pass-through", nethttp.NewHandler(directGzip()))
	r.Method(http.MethodHead, "/gzip-pass-through", nethttp.NewHandler(directGzip()))

	// Swagger UI endpoint at /docs.
	r.Method(http.MethodGet, "/docs/openapi.json", apiSchema)
	r.Mount("/docs", v4.NewHandler(apiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))

	return r
}
