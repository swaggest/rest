//go:build go1.18

package main

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/swaggest/fchi"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest-fasthttp/response"
	"github.com/swaggest/rest-fasthttp/response/gzip"
	"github.com/swaggest/rest-fasthttp/web"
	swgui "github.com/swaggest/swgui/v4emb"
)

func NewRouter() fchi.Handler {
	s := web.DefaultService()

	s.OpenAPI.Info.Title = "Advanced Example"
	s.OpenAPI.Info.WithDescription("This app showcases a variety of features.")
	s.OpenAPI.Info.Version = "v1.2.3"
	s.OpenAPICollector.Reflector().InterceptDefName(func(t reflect.Type, defaultDefName string) string {
		return strings.ReplaceAll(defaultDefName, "Generic", "")
	})

	// An example of global schema override to disable additionalProperties for all object schemas.
	s.OpenAPICollector.Reflector().DefaultOptions = append(s.OpenAPICollector.Reflector().DefaultOptions, func(rc *jsonschema.ReflectContext) {
		it := rc.InterceptType
		rc.InterceptType = func(value reflect.Value, schema *jsonschema.Schema) (bool, error) {
			stop, err := it(value, schema)
			if err != nil {
				return stop, err
			}

			// Allow unknown request headers and skip response.
			if oc, ok := openapi3.OperationCtx(rc); !ok ||
				oc.ProcessingResponse || oc.ProcessingIn == string(rest.ParamInHeader) {
				return stop, nil
			}

			if schema.HasType(jsonschema.Object) && len(schema.Properties) > 0 && schema.AdditionalProperties == nil {
				schema.AdditionalProperties = (&jsonschema.SchemaOrBool{}).WithTypeBoolean(false)
			}

			return stop, nil
		}
	})

	// When multiple structures can be returned with the same HTTP status code, it is possible to combine them into a
	// single schema with such configuration.
	s.OpenAPICollector.CombineErrors = "anyOf"

	s.Wrap(
		// Example middleware to set up custom error responses.
		func(handler fchi.Handler) fchi.Handler {
			var h *fhttp.Handler
			if fhttp.HandlerAs(handler, &h) {
				h.MakeErrResp = func(ctx context.Context, err error) (int, interface{}) {
					code, er := rest.Err(err)

					var ae anotherErr

					if errors.As(err, &ae) {
						return http.StatusBadRequest, ae
					}

					return code, customErr{
						Message: er.ErrorText,
						Details: er.Context,
					}
				}

				var hr rest.HandlerWithRoute
				if h.RespValidator != nil &&
					fhttp.HandlerAs(handler, &hr) {
					if hr.RoutePattern() == "/json-body-manual/{in-path}" || hr.RoutePattern() == "/json-body/{in-path}" {
						h.RespValidator = nil
					}
				}
			}

			return handler
		},

		// Response validator setup.
		//
		// It might be a good idea to disable this middleware in production to save performance,
		// but keep it enabled in dev/test/staging environments to catch logical issues.
		response.ValidatorMiddleware(s.ResponseValidatorFactory),
		gzip.Middleware, // Response compression with support for direct gzip pass through.
	)

	// Annotations can be used to alter documentation of operation identified by method and path.
	s.OpenAPICollector.Annotate(http.MethodPost, "/validation", func(op *openapi3.Operation) error {
		if op.Description != nil {
			*op.Description = *op.Description + " Custom annotation."
		}

		return nil
	})

	s.Get("/query-object", queryObject())

	s.Post("/file-upload", fileUploader())
	s.Post("/file-multi-upload", fileMultiUploader())
	s.Get("/json-param/{in-path}", jsonParam())
	s.Post("/json-body/{in-path}", jsonBody(),
		fhttp.SuccessStatus(http.StatusCreated))
	s.Post("/json-body-manual/{in-path}", jsonBodyManual(),
		fhttp.SuccessStatus(http.StatusCreated))
	s.Post("/json-body-validation/{in-path}", jsonBodyValidation())
	s.Post("/json-slice-body", jsonSliceBody())

	s.Post("/json-map-body", jsonMapBody(),
		// Annotate operation to add post-processing if necessary.
		fhttp.AnnotateOperation(func(op *openapi3.Operation) error {
			op.WithDescription("Request with JSON object (map) body.")

			return nil
		}))

	s.Get("/output-headers", outputHeaders())
	s.Head("/output-headers", outputHeaders())
	s.Get("/output-csv-writer", outputCSVWriter(),
		fhttp.SuccessfulResponseContentType("text/csv; charset=utf-8"))

	s.Post("/req-resp-mapping", reqRespMapping(),
		fhttp.RequestMapping(new(struct {
			Val1 string `header:"X-Header"`
			Val2 int    `formData:"val2"`
		})),
		fhttp.ResponseHeaderMapping(new(struct {
			Val1 string `header:"X-Value-1"`
			Val2 int    `header:"X-Value-2"`
		})),
	)

	s.Post("/validation", validation())
	s.Post("/no-validation", noValidation())

	// Type mapping is necessary to pass interface as structure into documentation.
	s.OpenAPICollector.Reflector().AddTypeMapping(new(gzipPassThroughOutput), new(gzipPassThroughStruct))
	s.Get("/gzip-pass-through", directGzip())
	s.Head("/gzip-pass-through", directGzip())

	s.Get("/error-response", errorResponse())

	// Swagger UI endpoint at /docs.
	s.Docs("/docs", swgui.New)

	return s
}
