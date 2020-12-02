package nethttp

import (
	"github.com/swaggest/openapi-go/openapi3"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/log"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/schema"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/service"
	"github.com/swaggest/rest/_examples/task-api/internal/usecase"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/swgui/v3cdn"
)

// NewRouter creates HTTP router.
func NewRouter(locator *service.Locator) http.Handler {
	apiSchema := schema.NewOpenAPICollector()
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	r := chirouter.NewWrapper(chi.NewRouter())

	r.Use(
		middleware.Recoverer, // Panic recovery.
		nethttp.UseCaseMiddlewares(log.UseCaseMiddleware()), // Sample use case middleware.
		nethttp.OpenAPIMiddleware(apiSchema),                // Documentation collector.
		request.DecoderMiddleware(decoderFactory),           // Decoder setup.
		request.ValidatorMiddleware(validatorFactory),       // Validator setup.
		response.EncoderMiddleware,                          // Encoder setup.
	)

	adminAuth := middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"})
	userAuth := middleware.BasicAuth("User Access", map[string]string{"user": "user"})

	r.Use(
		middleware.NoCache,
		middleware.Timeout(time.Second),
	)

	ff := func(h *nethttp.Handler) {
		h.ReqMapping = rest.RequestMapping{rest.ParamInPath: map[string]string{"ID": "id"}}
	}

	// Unrestricted access.
	r.Route("/dev", func(r chi.Router) {
		r.Use(nethttp.AnnotateOpenAPI(apiSchema, func(op *openapi3.Operation) error {
			op.Tags = []string{"Dev Mode"}

			return nil
		}))
		r.Group(func(r chi.Router) {
			r.Method(http.MethodPost, "/tasks", nethttp.NewHandler(usecase.CreateTask(locator)))
			r.Method(http.MethodPut, "/tasks/{id}", nethttp.NewHandler(usecase.UpdateTask(locator), ff))
			r.Method(http.MethodGet, "/tasks/{id}", nethttp.NewHandler(usecase.FindTask(locator), ff))
			r.Method(http.MethodGet, "/tasks", nethttp.NewHandler(usecase.FindTasks(locator)))
			r.Method(http.MethodDelete, "/tasks/{id}", nethttp.NewHandler(usecase.FinishTask(locator), ff))
		})
	})

	// Endpoints with admin access.
	r.Route("/admin", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(nethttp.AnnotateOpenAPI(apiSchema, func(op *openapi3.Operation) error {
				op.Tags = []string{"Admin Mode"}

				return nil
			}))
			r.Use(adminAuth, nethttp.HTTPBasicSecurityMiddleware(apiSchema, "Admin", "Admin access"))
			r.Method(http.MethodPut, "/tasks/{id}", nethttp.NewHandler(usecase.UpdateTask(locator), ff))
		})
	})

	// Endpoints with user access.
	r.Route("/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(userAuth, nethttp.HTTPBasicSecurityMiddleware(apiSchema, "User", "User access"))
			r.Method(http.MethodPost, "/tasks", nethttp.NewHandler(usecase.CreateTask(locator)))
		})
	})

	// Swagger UI endpoint at /docs.
	r.Method(http.MethodGet, "/docs/openapi.json", apiSchema)
	r.Mount("/docs", v3cdn.NewHandler(apiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))

	r.Mount("/debug", middleware.Profiler())

	return r
}
