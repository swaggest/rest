package main

import (
	"context"
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

type jsonFilter struct {
	Foo    string `json:"foo"`
	Bar    int    `json:"bar"`
	Deeper struct {
		Val string `json:"val"`
	} `json:"deeper"`
}

type deepObjectFilter struct {
	Baz    bool    `query:"baz"`
	Quux   float64 `query:"quux"`
	Deeper struct {
		Val string `query:"val"`
	} `query:"deeper"`
}

// Declare input port type.
type helloInput struct {
	Filter        jsonFilter       `query:"filter"`
	NonJSONFilter deepObjectFilter `query:"non_json_filter"`
	Other         string           `query:"other"`
	// Locale        string           `query:"locale" default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$" enum:"ru-RU,en-US"`
	// Name          string           `path:"name" minLength:"3"` // Field tags define parameter location and JSON schema constraints.
}

func (h *helloInput) SetRequest(r *http.Request) {
	q := r.URL.Query()

	println(q)
}

// Declare output port type.
type helloOutput struct {
	Now     time.Time `header:"X-Now" json:"-"`
	Message string    `json:"message"`
}

func main() {
	s := web.DefaultService()

	// Init API documentation schema.
	s.OpenAPI.Info.Title = "Basic Example"
	s.OpenAPI.Info.WithDescription("This app showcases a trivial REST API.")
	s.OpenAPI.Info.Version = "v1.2.3"

	// Setup middlewares.
	s.Wrap(
		gzip.Middleware, // Response compression with support for direct gzip pass through.
	)

	//messages := map[string]string{
	//	"en-US": "Hello, %s!",
	//	"ru-RU": "Привет, %s!",
	//}

	// Create use case interactor with references to input/output types and interaction function.
	u := usecase.NewIOI(new(helloInput), new(helloOutput), func(ctx context.Context, input, output interface{}) error {
		//var (
		//	in  = input.(*helloInput)
		//	out = output.(*helloOutput)
		//)

		//msg, available := messages[in.Locale]
		//if !available {
		//	return status.Wrap(errors.New("unknown locale"), status.InvalidArgument)
		//}

		fmt.Printf("%#v\n", input)
		//
		//out.Message = fmt.Sprintf(msg, in.Name)
		//out.Now = time.Now()

		return nil
	})

	// Describe use case interactor.
	u.SetTitle("Greeter")
	u.SetDescription("Greeter greets you.")

	u.SetExpectedErrors(status.InvalidArgument)

	// Add use case handler to router.
	s.Get("/hello/name", u)

	// Swagger UI endpoint at /docs.
	s.Docs("/docs", swgui.New)

	// Start server.
	log.Println("http://localhost:8011/docs")
	if err := http.ListenAndServe("localhost:8011", s); err != nil {
		log.Fatal(err)
	}
}
