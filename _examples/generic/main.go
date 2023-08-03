//go:build go1.18
// +build go1.18

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v5emb"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func main() {
	s := web.DefaultService()

	// Init API documentation schema.
	s.OpenAPISchema().SetTitle("Basic Example")
	s.OpenAPISchema().SetDescription("This app showcases a trivial REST API.")
	s.OpenAPISchema().SetVersion("v1.2.3")

	// Setup middlewares.
	s.Wrap(
		gzip.Middleware, // Response compression with support for direct gzip pass through.
	)

	// Declare input port type.
	type helloInput struct {
		Locale string `query:"locale" default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$" enum:"ru-RU,en-US"`
		Name   string `path:"name" minLength:"3"` // Field tags define parameter location and JSON schema constraints.

		// Field tags of unnamed fields are applied to parent schema.
		// they are optional and can be used to disallow unknown parameters.
		// For non-body params, name tag must be provided explicitly.
		// E.g. here no unknown `query` and `cookie` parameters allowed,
		// unknown `header` params are ok.
		_ struct{} `query:"_" cookie:"_" additionalProperties:"false"`
	}

	// Declare output port type.
	type helloOutput struct {
		Now     time.Time `header:"X-Now" json:"-"`
		Message string    `json:"message"`
	}

	messages := map[string]string{
		"en-US": "Hello, %s!",
		"ru-RU": "Привет, %s!",
	}

	// Create use case interactor with references to input/output types and interaction function.
	u := usecase.NewInteractor(func(ctx context.Context, input helloInput, output *helloOutput) error {
		msg, available := messages[input.Locale]
		if !available {
			return status.Wrap(errors.New("unknown locale"), status.InvalidArgument)
		}

		output.Message = fmt.Sprintf(msg, input.Name)
		output.Now = time.Now()

		return nil
	})

	// Describe use case interactor.
	u.SetTitle("Greeter")
	u.SetDescription("Greeter greets you.")

	u.SetExpectedErrors(status.InvalidArgument)

	// Add use case handler to router.
	s.Get("/hello/{name}", u)

	// Swagger UI endpoint at /docs.
	s.Docs("/docs", swgui.New)

	// Start server.
	log.Println("http://localhost:8011/docs")
	if err := http.ListenAndServe("localhost:8011", s); err != nil {
		log.Fatal(err)
	}
}
