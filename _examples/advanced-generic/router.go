//go:build go1.18

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"github.com/swaggest/jsonschema-go"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v4emb"
	"github.com/swaggest/usecase"
)

func NewRouter() http.Handler {
	s := web.DefaultService()

	s.OpenAPISchema().SetTitle("Advanced Example")
	s.OpenAPISchema().SetDescription("This app showcases a variety of features.")
	s.OpenAPISchema().SetVersion("v1.2.3")

	jsr := s.OpenAPIReflector().JSONSchemaReflector()

	jsr.DefaultOptions = append(jsr.DefaultOptions, jsonschema.InterceptDefName(
		func(t reflect.Type, defaultDefName string) string {
			return strings.ReplaceAll(defaultDefName, "Generic", "")
		},
	))

	// Usecase middlewares can be added to web.Service or chirouter.Wrapper.
	s.Wrap(nethttp.UseCaseMiddlewares(usecase.MiddlewareFunc(func(next usecase.Interactor) usecase.Interactor {
		var (
			hasName usecase.HasName
			name    = "unknown"
		)

		if usecase.As(next, &hasName) {
			name = hasName.Name()
		}

		return usecase.Interact(func(ctx context.Context, input, output interface{}) error {
			err := next.Interact(ctx, input, output)
			if err != nil && !errors.Is(err, rest.HTTPCodeAsError(http.StatusNotModified)) {
				log.Printf("usecase %s request (%+v) failed: %v\n", name, input, err)
			}

			return err
		})
	})))

	// An example of global schema override to disable additionalProperties for all object schemas.
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

	// When multiple structures can be returned with the same HTTP status code, it is possible to combine them into a
	// single schema with such configuration.
	s.OpenAPICollector.CombineErrors = "anyOf"

	s.Wrap(
		// Example middleware to set up custom error responses and disable response validation for particular handlers.
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

				var hr rest.HandlerWithRoute
				if h.RespValidator != nil &&
					nethttp.HandlerAs(handler, &hr) {
					if hr.RoutePattern() == "/json-body-manual/{in-path}" || hr.RoutePattern() == "/json-body/{in-path}" {
						h.RespValidator = nil
					}
				}
			}

			return handler
		},

		// Example middleware to set up CORS headers.
		// See https://pkg.go.dev/github.com/rs/cors for more details.
		cors.AllowAll().Handler,

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

	//s.OpenAPICollector.AnnotateOperation(http.MethodPost, "/validation", func(oc oapi.OperationContext) error {
	//	o3, ok := oc.(openapi3.OperationExposer)
	//	if !ok {
	//		return nil
	//	}
	//
	//	op := o3.Operation()
	//
	//	if op.Description != nil {
	//		*op.Description = *op.Description + " Custom annotation."
	//	}
	//
	//	return nil
	//})

	s.Get("/query-object", queryObject())
	s.Post("/form", form())

	s.Post("/file-upload", fileUploader())
	s.Post("/file-multi-upload", fileMultiUploader())
	s.Get("/json-param/{in-path}", jsonParam())
	s.Post("/json-body/{in-path}", jsonBody(),
		nethttp.SuccessStatus(http.StatusCreated))
	s.Post("/json-body-manual/{in-path}", jsonBodyManual(),
		nethttp.SuccessStatus(http.StatusCreated))
	s.Post("/json-body-validation/{in-path}", jsonBodyValidation())
	s.Post("/json-slice-body", jsonSliceBody())

	s.Post("/json-map-body", jsonMapBody(),
		// Annotate operation to add post-processing if necessary.
		nethttp.AnnotateOpenAPIOperation(func(oc oapi.OperationContext) error {
			oc.SetDescription("Request with JSON object (map) body.")

			return nil
		}))

	s.Get("/html-response/{id}", htmlResponse(), nethttp.SuccessfulResponseContentType("text/html"))

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
	s.Post("/text-req-body/{path}", textReqBody(), nethttp.RequestBodyContent("text/csv"))
	s.Post("/text-req-body-ptr/{path}", textReqBodyPtr(), nethttp.RequestBodyContent("text/csv"))

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

	sessDoc := nethttp.SecurityMiddleware(s.OpenAPICollector, "User", openapi3.SecurityScheme{
		APIKeySecurityScheme: &openapi3.APIKeySecurityScheme{
			In:   "cookie",
			Name: "sessid",
		},
	})

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
