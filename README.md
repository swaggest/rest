# REST with Clean Architecture for Go

[![Build Status](https://github.com/swaggest/rest/workflows/test/badge.svg)](https://github.com/swaggest/rest/actions?query=branch%3Amaster+workflow%3Atest)
[![Coverage Status](https://codecov.io/gh/swaggest/rest/branch/master/graph/badge.svg)](https://codecov.io/gh/swaggest/rest)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/swaggest/rest)
[![Time Tracker](https://wakatime.com/badge/github/swaggest/rest.svg)](https://wakatime.com/badge/github/swaggest/rest)
![Code lines](https://sloc.xyz/github/swaggest/rest/?category=code)
![Comments](https://sloc.xyz/github/swaggest/rest/?category=comments)

This module implements HTTP transport level for [`github.com/swaggest/usecase`](https://github.com/swaggest/usecase) 
to build REST services.

## Goals

* Maintain single source of truth for documentation, validation and input/output of HTTP API.
* Avoid dependency on compile time code generation.
* Improve productivity and reliability by abstracting HTTP details with simple API for common case.
* Allow low-level customizations for advanced cases.
* Maintain reasonable performance with low GC impact.

## Non-Goals

* Support for legacy documentation schemas like Swagger 2.0 or RAML.
* Zero allocations.
* Explicit support for XML in request or response bodies.

## Features

* Compatible with `net/http`.
* Built with [`github.com/go-chi/chi`](https://github.com/go-chi/chi) router.
* Modular flexible structure.
* HTTP [request mapping](#request-decoder) into Go value based on field tags.
* Decoupled business logic with Clean Architecture use cases.
* Automatic type-safe OpenAPI 3 documentation with [`github.com/swaggest/openapi-go`](https://github.com/swaggest/openapi-go).
* Single source of truth for the documentation and endpoint interface.
* Automatic request/response JSON schema validation with [`github.com/santhosh-tekuri/jsonschema`](https://github.com/santhosh-tekuri/jsonschema).
* Dynamic gzip compression and fast pass through mode.
* Optimized performance.
* Embedded [Swagger UI](https://swagger.io/tools/swagger-ui/).
* Integration test helpers.
* Generic interface for [use case interactors](https://pkg.go.dev/github.com/swaggest/usecase#NewInteractor). 

## Usage

### Request Decoder

Go struct with optional field tags defines input port. 
Request decoder populates field values from `http.Request` data before use case interactor is invoked. 

```go
// Declare input port type.
type helloInput struct {
    Locale string `query:"locale" default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$"`
    Name   string `path:"name" minLength:"3"` // Field tags define parameter location and JSON schema constraints.
	_      string `additionalProperties:"false"` // Field tags of unnamed fields are applied to parent schema.
}
```

Input data can be located in:
* `path` parameter in request URI, e.g. `/users/{name}`,
* `query` parameter in request URI, e.g. `/users?locale=en-US`,
* `formData` parameter in request body with `application/x-www-form-urlencoded` or `multipart/form-data` content,
* `json` parameter in request body with `application/json` content,
* `cookie` parameter in request cookie,
* `header` parameter in request header.

For more explicit separation of concerns between use case and transport it is possible to provide request mapping 
separately when initializing handler (please note, such mapping is [not applied](https://github.com/swaggest/rest/issues/61#issuecomment-1059851553) to `json` body).

```go
// Declare input port type.
type helloInput struct {
    Locale string `default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$"`
    Name   string `minLength:"3"` // Field tags define parameter location and JSON schema constraints.
}
```

```go
// Add use case handler with custom input mapping to router.
r.Method(http.MethodGet, "/hello/{name}", nethttp.NewHandler(u,
    nethttp.RequestMapping(new(struct {
       Locale string `query:"locale"`
       Name   string `path:"name"` // Field tags define parameter location and JSON schema constraints.
    })),
))
```

Additional field tags describe JSON schema constraints, please check 
[documentation](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Reflector.Reflect).

More schema customizations are possible with [`github.com/swaggest/jsonschema-go interfaces`](https://github.com/swaggest/jsonschema-go#implementing-interfaces-on-a-type).

By default `default` tags are only contributing to documentation, 
if [`request.DecoderFactory.ApplyDefaults`](https://pkg.go.dev/github.com/swaggest/rest/request#DecoderFactory) is 
set to `true`, fields of request structure that don't have explicit value but have `default` will be populated with 
default value.

If input structure implements [`request.Loader`](https://pkg.go.dev/github.com/swaggest/rest/request#Loader),  
then `LoadFromHTTPRequest(r *http.Request) error` method will be invoked to populate input structure instead 
of automatic decoding. This allows low level control for cases that need it.

### Response Encoder

Go struct with field tags defines output port.
Response encoder writes data from output to `http.ResponseWriter` after use case interactor invocation finishes.

```go
// Declare output port type.
type helloOutput struct {
    Now     time.Time `header:"X-Now" json:"-"`
    Message string    `json:"message"`
}
```

Output data can be located in:
* `json` for response body with `application/json` content,
* `header` for values in response header.

For more explicit separation of concerns between use case and transport it is possible to provide response header mapping 
separately when initializing handler.

```go
// Declare output port type.
type helloOutput struct {
    Now     time.Time `json:"-"`
    Message string    `json:"message"`
}
```

```go
// Add use case handler with custom output headers mapping to router.
r.Method(http.MethodGet, "/hello/{name}", nethttp.NewHandler(u,
    nethttp.ResponseHeaderMapping(new(struct {
        Now     time.Time `header:"X-Now"`
    })),
))
```

Additional field tags describe JSON schema constraints, please check 
[documentation](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Reflector.Reflect).

### Creating Use Case Interactor

HTTP transport is decoupled from business logic by adapting
[use case interactors](https://pkg.go.dev/github.com/swaggest/usecase#Interactor).

Use case interactor can define input and output ports that are used to map data between Go values and transport.
It can provide information about itself that will be exposed in generated documentation.

```go
// Create use case interactor with references to input/output types and interaction function.
u := usecase.NewIOI(new(helloInput), new(helloOutput), func(ctx context.Context, input, output interface{}) error {
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
```

For modularity particular use case interactor instance can be assembled by embedding relevant traits in a struct,
for example you can skip adding `usecase.WithInput` if your use case does not imply any input.

```go
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
u.Input = new(helloInput)
u.Output = new(helloOutput)
u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
    // Do something about input to prepare output.
    return nil
})
```

### Adding use case to router

```go
// Add use case handler to router.
r.Method(http.MethodGet, "/hello/{name}", nethttp.NewHandler(u))
```

## API Schema Collector

OpenAPI schema should be initialized with general information about REST API.

It uses [type-safe mapping](https://github.com/swaggest/openapi-go) for the configuration, 
so any IDE will help with available fields. 

```go
// Init API documentation schema.
apiSchema := &openapi.Collector{}
apiSchema.Reflector().SpecEns().Info.Title = "Basic Example"
apiSchema.Reflector().SpecEns().Info.WithDescription("This app showcases a trivial REST API.")
apiSchema.Reflector().SpecEns().Info.Version = "v1.2.3"
```

## Router Setup

REST router is based on [`github.com/go-chi/chi`](https://github.com/go-chi/chi), wrapper allows unwrapping instrumented
handler in middleware.

These middlewares are required:
* `nethttp.OpenAPIMiddleware(apiSchema)`, 
* `request.DecoderMiddleware(decoderFactory)`,
* `response.EncoderMiddleware`.

Optionally you can add more middlewares with some performance impact:
* `request.ValidatorMiddleware(validatorFactory)` (request validation, recommended)
* `response.ValidatorMiddleware(validatorFactory)`
* `gzip.Middleware`

You can also add any other 3rd party middlewares compatible with `net/http` at your discretion.

```go
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
```

Register Swagger UI to serve documentation at `/docs`.

```go
// Swagger UI endpoint at /docs.
r.Method(http.MethodGet, "/docs/openapi.json", apiSchema)
r.Mount("/docs", v3cdn.NewHandler(apiSchema.Reflector().Spec.Info.Title,
    "/docs/openapi.json", "/docs"))
```

## Security Setup

```go
// Prepare middleware with suitable security schema.
// It will perform actual security check for every relevant request.
adminAuth := middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"})

// Prepare API schema updater middleware.
// It will annotate handler documentation with security schema.
adminSecuritySchema := nethttp.HTTPBasicSecurityMiddleware(apiSchema, "Admin", "Admin access")

// Endpoints with admin access.
r.Route("/admin", func(r chi.Router) {
    r.Group(func(r chi.Router) {
        r.Use(adminAuth, adminSecuritySchema) // Add both middlewares to routing group to enforce and document security.
        r.Method(http.MethodPut, "/hello/{name}", nethttp.NewHandler(u))
    })
})
```

See [example](./_examples/task-api/internal/infra/nethttp/router.go).

## Handler Setup

Handler is a generalized adapter for use case interactor, so usually setup is trivial.

```go
// Add use case handler to router.
r.Method(http.MethodGet, "/hello/{name}", nethttp.NewHandler(u))
```

## Example

For non-generic use case, see another [example](./_examples/basic/main.go).

```go
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
	swgui "github.com/swaggest/swgui/v4"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func main() {
	s := web.DefaultService()

	// Init API documentation schema.
	s.OpenAPI.Info.Title = "Basic Example"
	s.OpenAPI.Info.WithDescription("This app showcases a trivial REST API.")
	s.OpenAPI.Info.Version = "v1.2.3"

	// Setup middlewares.
	s.Use(
		gzip.Middleware, // Response compression with support for direct gzip pass through.
	)

	// Declare input port type.
	type helloInput struct {
		Locale string `query:"locale" default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$" enum:"ru-RU,en-US"`
		Name   string `path:"name" minLength:"3"` // Field tags define parameter location and JSON schema constraints.
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
	if err := http.ListenAndServe(":8011", s); err != nil {
		log.Fatal(err)
	}
}

```

![Documentation Page](./_examples/basic/screen.png)

## Versioning

This project adheres to [Semantic Versioning](https://semver.org/#semantic-versioning-200).

Before version `1.0.0`, breaking changes are tagged with `MINOR` bump, features and fixes are tagged with `PATCH` bump.
After version `1.0.0`, breaking changes are tagged with `MAJOR` bump.

Breaking changes are described in [UPGRADE.md](./UPGRADE.md).
