package fhttp

import (
	"net/http"
	"time"

	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra/schema"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra/service"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/usecase"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest-fasthttp/web"
	swgui "github.com/swaggest/swgui/v4emb"
)

// NewRouter creates HTTP router.
func NewRouter(locator *service.Locator) fchi.Handler {
	s := web.DefaultService()

	schema.SetupOpenAPICollector(s.OpenAPICollector)

	adminAuth := middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"})
	userAuth := middleware.BasicAuth("User Access", map[string]string{"user": "user"})

	s.Wrap(
		middleware.NoCache,
		middleware.Timeout(time.Second),
	)

	ff := func(h *fhttp.Handler) {
		h.ReqMapping = rest.RequestMapping{rest.ParamInPath: map[string]string{"ID": "id"}}
	}

	// Unrestricted access.
	s.Route("/dev", func(r fchi.Router) {
		r.Use(fhttp.AnnotateOpenAPI(s.OpenAPICollector, func(op *openapi3.Operation) error {
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
	s.Route("/admin", func(r fchi.Router) {
		r.Group(func(r fchi.Router) {
			r.Use(fhttp.AnnotateOpenAPI(s.OpenAPICollector, func(op *openapi3.Operation) error {
				op.Tags = []string{"Admin Mode"}

				return nil
			}))
			r.Use(adminAuth, fhttp.HTTPBasicSecurityMiddleware(s.OpenAPICollector, "Admin", "Admin access"))
			r.Method(http.MethodPut, "/tasks/{id}", fhttp.NewHandler(usecase.UpdateTask(locator), ff))
		})
	})

	// Endpoints with user access.
	s.Route("/user", func(r fchi.Router) {
		r.Group(func(r fchi.Router) {
			r.Use(userAuth, fhttp.HTTPBasicSecurityMiddleware(s.OpenAPICollector, "User", "User access"))
			r.Method(http.MethodPost, "/tasks", fhttp.NewHandler(usecase.CreateTask(locator),
				fhttp.SuccessStatus(http.StatusCreated)))
		})
	})

	// Swagger UI endpoint at /docs.
	s.Docs("/docs", swgui.New)

	s.Mount("/debug", middleware.Profiler())

	return s
}
