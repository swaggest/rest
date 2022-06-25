package fhttp

import (
	"net/http"
	"time"

	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
	"github.com/swaggest/openapi-go/openapi3"
	rest2 "github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra/log"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra/schema"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra/service"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/usecase"
	"github.com/swaggest/rest-fasthttp/chirouter"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest-fasthttp/request"
	"github.com/swaggest/rest-fasthttp/response"
	"github.com/swaggest/rest/jsonschema"
	swgui "github.com/swaggest/swgui/v4emb"
)

// NewRouter creates HTTP router.
func NewRouter(locator *service.Locator) fchi.Handler {
	apiSchema := schema.NewOpenAPICollector()
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.SetDecoderFunc(rest2.ParamInPath, chirouter.PathToURLValues)

	r := chirouter.NewWrapper(fchi.NewRouter())

	r.Wrap(
		middleware.Recoverer,                              // Panic recovery.
		fhttp.UseCaseMiddlewares(log.UseCaseMiddleware()), // Sample use case middleware.
		fhttp.OpenAPIMiddleware(apiSchema),                // Documentation collector.
		request.DecoderMiddleware(decoderFactory),         // Decoder setup.
		request.ValidatorMiddleware(validatorFactory),     // Validator setup.
		response.EncoderMiddleware,                        // Encoder setup.
	)

	adminAuth := middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"})
	userAuth := middleware.BasicAuth("User Access", map[string]string{"user": "user"})

	r.Wrap(
		middleware.NoCache,
		middleware.Timeout(time.Second),
	)

	ff := func(h *fhttp.Handler) {
		h.ReqMapping = rest2.RequestMapping{rest2.ParamInPath: map[string]string{"ID": "id"}}
	}

	// Unrestricted access.
	r.Route("/dev", func(r fchi.Router) {
		r.Use(fhttp.AnnotateOpenAPI(apiSchema, func(op *openapi3.Operation) error {
			op.Tags = []string{"Dev Mode"}

			return nil
		}))
		r.Group(func(r fchi.Router) {
			r.Method(http.MethodPost, "/tasks", fhttp.NewHandler(usecase.CreateTask(locator),
				fhttp.SuccessStatus(http.StatusCreated)))
			r.Method(http.MethodPut, "/tasks/{id}", fhttp.NewHandler(usecase.UpdateTask(locator), ff))
			r.Method(http.MethodGet, "/tasks/{id}", fhttp.NewHandler(usecase.FindTask(locator), ff))
			r.Method(http.MethodGet, "/tasks", fhttp.NewHandler(usecase.FindTasks(locator)))
			r.Method(http.MethodDelete, "/tasks/{id}", fhttp.NewHandler(usecase.FinishTask(locator), ff))
		})
	})

	// Endpoints with admin access.
	r.Route("/admin", func(r fchi.Router) {
		r.Group(func(r fchi.Router) {
			r.Use(fhttp.AnnotateOpenAPI(apiSchema, func(op *openapi3.Operation) error {
				op.Tags = []string{"Admin Mode"}

				return nil
			}))
			r.Use(adminAuth, fhttp.HTTPBasicSecurityMiddleware(apiSchema, "Admin", "Admin access"))
			r.Method(http.MethodPut, "/tasks/{id}", fhttp.NewHandler(usecase.UpdateTask(locator), ff))
		})
	})

	// Endpoints with user access.
	r.Route("/user", func(r fchi.Router) {
		r.Group(func(r fchi.Router) {
			r.Use(userAuth, fhttp.HTTPBasicSecurityMiddleware(apiSchema, "User", "User access"))
			r.Method(http.MethodPost, "/tasks", fhttp.NewHandler(usecase.CreateTask(locator),
				fhttp.SuccessStatus(http.StatusCreated)))
		})
	})

	// Swagger UI endpoint at /docs.
	r.Method(http.MethodGet, "/docs/openapi.json", fchi.Adapt(apiSchema))
	r.Mount("/docs", fchi.Adapt(swgui.NewHandler(apiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs")))

	r.Mount("/debug", middleware.Profiler())

	return r
}
