package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/swaggest/jsonschema-go"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v5emb"
)

func NewRouter() http.Handler {
	response.DefaultErrorResponseContentType = "application/problem+json"
	response.DefaultSuccessResponseContentType = "application/dummy+json"

	s := web.NewService(openapi3.NewReflector())

	s.OpenAPISchema().SetTitle("Advanced Example")
	s.OpenAPISchema().SetDescription("This app showcases a variety of features.")
	s.OpenAPISchema().SetVersion("v1.2.3")

	// An example of global schema override to disable additionalProperties for all object schemas.
	jsr := s.OpenAPIReflector().JSONSchemaReflector()
	jsr.DefaultOptions = append(jsr.DefaultOptions,
		jsonschema.InterceptSchema(func(params jsonschema.InterceptSchemaParams) (stop bool, err error) {
			// Allow unknown request headers and skip response.
			if oc, ok := oapi.OperationCtx(params.Context); !params.Processed || !ok ||
				oc.IsProcessingResponse() || oc.ProcessingIn() == oapi.InHeader {
				return false, nil
			}

			schema := params.Schema

			if schema.HasType(jsonschema.Object) && len(schema.Properties) > 0 && schema.AdditionalProperties == nil {
				schema.AdditionalProperties = (&jsonschema.SchemaOrBool{}).WithTypeBoolean(false)
			}

			return false, nil
		}),
	)

	// Create custom schema mapping for 3rd party type.
	uuidDef := jsonschema.Schema{}
	uuidDef.AddType(jsonschema.String)
	uuidDef.WithFormat("uuid")
	uuidDef.WithExamples("248df4b7-aa70-47b8-a036-33ac447e668d")
	jsr.AddTypeMapping(uuid.UUID{}, uuidDef)

	s.OpenAPICollector.CombineErrors = "anyOf"

	s.Wrap(
		// Response validator setup.
		//
		// It might be a good idea to disable this middleware in production to save performance,
		// but keep it enabled in dev/test/staging environments to catch logical issues.
		response.ValidatorMiddleware(s.ResponseValidatorFactory),
		gzip.Middleware, // Response compression with support for direct gzip pass through.

		// Example middleware to setup custom error responses.
		func(handler http.Handler) http.Handler {
			var h *nethttp.Handler
			if nethttp.HandlerAs(handler, &h) {
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
			}

			return handler
		},
	)

	// Annotations can be used to alter documentation of operation identified by method and path.
	s.OpenAPICollector.AnnotateOperation(http.MethodPost, "/validation", func(oc oapi.OperationContext) error {
		o3, ok := oc.(openapi3.OperationExposer)
		if !ok {
			return nil
		}

		op := o3.Operation()

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
		nethttp.SuccessStatus(http.StatusCreated))
	s.Post("/json-body-validation/{in-path}", jsonBodyValidation())
	s.Post("/json-slice-body", jsonSliceBody())

	s.Post("/json-map-body", jsonMapBody(),
		// Annotate operation to add post-processing if necessary.
		nethttp.AnnotateOpenAPIOperation(func(oc oapi.OperationContext) error {
			oc.SetDescription("Request with JSON object (map) body.")

			return nil
		}))

	s.Get("/output-headers", outputHeaders())
	s.Head("/output-headers", outputHeaders())
	s.Get("/output-csv-writer", outputCSVWriter(),
		nethttp.SuccessfulResponseContentType("text/csv; charset=utf-8"))

	s.Post("/req-resp-mapping", reqRespMapping(),
		nethttp.RequestMapping(new(struct {
			Val1 string `header:"X-Header"`
			Val2 int    `formData:"val2"`
		})),
		nethttp.ResponseHeaderMapping(new(struct {
			Val1 string `header:"X-Value-1"`
			Val2 int    `header:"X-Value-2"`
		})),
	)

	s.Post("/validation", validation())
	s.Post("/no-validation", noValidation())

	// Type mapping is necessary to pass interface as structure into documentation.
	jsr.AddTypeMapping(new(gzipPassThroughOutput), new(gzipPassThroughStruct))
	s.Get("/gzip-pass-through", directGzip())
	s.Head("/gzip-pass-through", directGzip())

	s.Get("/error-response", errorResponse())
	s.Get("/dynamic-schema", dynamicSchema())

	// Security middlewares.
	//  - sessMW is the actual request-level processor,
	//  - sessDoc is a handler-level wrapper to expose docs.
	sessMW := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c, err := r.Cookie("sessid"); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), "sessionID", c.Value))
			}

			handler.ServeHTTP(w, r)
		})
	}

	sessDoc := nethttp.APIKeySecurityMiddleware(s.OpenAPICollector, "User",
		"sessid", oapi.InCookie, "Session cookie.")

	// Security schema is configured for a single top-level route.
	s.With(sessMW, sessDoc).Method(http.MethodGet, "/root-with-session", nethttp.NewHandler(dummy()))

	// Security schema is configured on a sub-router.
	s.Route("/deeper-with-session", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(sessMW, sessDoc)

			r.Method(http.MethodGet, "/one", nethttp.NewHandler(dummy()))
			r.Method(http.MethodGet, "/two", nethttp.NewHandler(dummy()))
		})
	})

	// Swagger UI endpoint at /docs.
	s.Docs("/docs", swgui.New)

	return s
}
