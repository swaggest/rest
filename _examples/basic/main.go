package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/swgui/v3cdn"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func main() {
	// Init API documentation schema.
	apiSchema := &openapi.Collector{}
	apiSchema.Reflector().SpecEns().Info.Title = "Basic Example"
	apiSchema.Reflector().SpecEns().Info.WithDescription("This app showcases a trivial REST API.")
	apiSchema.Reflector().SpecEns().Info.Version = "v1.2.3"

	// Setup request decoder and validator.
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	// Create router.
	r := chirouter.NewWrapper(chi.NewRouter())

	// Setup middlewares.
	r.Use(
		middleware.Recoverer,                          // Panic recovery.
		nethttp.OpenAPIMiddleware(apiSchema),          // Documentation collector.
		request.DecoderMiddleware(decoderFactory),     // Request decoder setup.
		request.ValidatorMiddleware(validatorFactory), // Request validator setup.
		response.EncoderMiddleware,                    // Response encoder setup.
		gzip.Middleware,                               // Response compression with support for direct gzip pass through.
	)

	// Create use case interactor.
	u := struct {
		usecase.Info
		usecase.Interactor
		usecase.WithInput
		usecase.WithOutput
	}{}

	// Describe use case interactor.
	u.SetTitle("Greeter")
	u.SetDescription("Greeter greets you.")

	// Declare input port type.
	type helloInput struct {
		Locale string `query:"locale" default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$" enum:"ru-RU,en-US"`
		Name   string `path:"name" minLength:"3"` // Field tags define parameter location and JSON schema constraints.
	}
	u.Input = new(helloInput)

	// Declare output port type.
	type helloOutput struct {
		Now     time.Time `header:"X-Now" json:"-"`
		Message string    `json:"message"`
	}
	u.Output = new(helloOutput)

	u.SetExpectedErrors(status.InvalidArgument)
	messages := map[string]string{
		"en-US": "Hello, %s!",
		"ru-RU": "Привет, %s!",
	}
	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		var (
			in  = input.(*helloInput)
			out = output.(*helloOutput)
		)

		msg, available := messages[in.Locale]
		if !available {
			return status.Wrap(errors.New("unknown locale"), status.InvalidArgument)
		}

		out.Message = fmt.Sprintf(msg, in.Name)
		out.Now = time.Now()

		return nil
	})

	// Add use case handler to router.
	r.Method(http.MethodGet, "/hello/{name}", nethttp.NewHandler(u))

	// Swagger UI endpoint at /docs.
	r.Method(http.MethodGet, "/docs/openapi.json", apiSchema)
	r.Mount("/docs", v3cdn.NewHandler(apiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))

	// Start server.
	log.Println("http://localhost:8011/docs")
	if err := http.ListenAndServe(":8011", r); err != nil {
		log.Fatal(err)
	}
}
