# Advanced Usage and Fine-tuning

In most cases you would not need to touch these APIs directly, and instead you may find `web.Service` sufficient.

If that's not the case, this document covers internal components.

## Creating Modular Use Case Interactor

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

## Adding use case to router

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
apiSchema := openapi.NewCollector(openapi31.NewReflector())
apiSchema.SpecSchema().SetTitle("Basic Example")
apiSchema.SpecSchema().SetDescription("This app showcases a trivial REST API.")
apiSchema.SpecSchema().SetVersion("v1.2.3")
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
