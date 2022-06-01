package chirouter

import (
	"net/http"
	"strings"

	"github.com/swaggest/fchi"
	"github.com/swaggest/rest/nethttp"
)

// NewWrapper creates router wrapper to upgrade middlewares processing.
func NewWrapper(r fchi.Router) *Wrapper {
	return &Wrapper{
		Router: r,
	}
}

// Wrapper wraps chi.Router to enable unwrappable handlers in middlewares.
//
// Middlewares can call nethttp.HandlerAs to inspect wrapped handlers.
type Wrapper struct {
	fchi.Router
	name        string
	basePattern string

	middlewares []func(fchi.Handler) fchi.Handler
	wraps       []func(fchi.Handler) fchi.Handler
}

var _ fchi.Router = &Wrapper{}

func (r *Wrapper) copy(router fchi.Router, pattern string) *Wrapper {
	return &Wrapper{
		Router:      router,
		name:        r.name,
		basePattern: r.basePattern + pattern,
		middlewares: r.middlewares,
		wraps:       r.wraps,
	}
}

// Wrap appends one or more wrappers that will be applied to handler before adding to Router.
// It is different from middleware in the sense that it is handler-centric, rather than request-centric.
// Wraps are invoked once for each added handler, they are not invoked for http requests.
// Wraps can leverage nethttp.HandlerAs to inspect and access deeper layers.
// For most cases Wrap can be safely used instead of Use, Use is mandatory for middlewares
// that affect routing (such as middleware.StripSlashes for example).
func (r *Wrapper) Wrap(wraps ...func(handler fchi.Handler) fchi.Handler) {
	r.wraps = append(r.wraps, wraps...)
}

// Use appends one of more middlewares onto the Router stack.
func (r *Wrapper) Use(middlewares ...func(fchi.Handler) fchi.Handler) {
	r.Router.Use(middlewares...)
	r.middlewares = append(r.middlewares, middlewares...)
}

// With adds inline middlewares for an endpoint handler.
func (r Wrapper) With(middlewares ...func(fchi.Handler) fchi.Handler) fchi.Router {
	c := r.copy(r.Router.With(middlewares...), "")
	c.middlewares = append(c.middlewares, middlewares...)

	return c
}

// Group adds a new inline-router along the current routing path, with a fresh middleware stack for the inline-router.
func (r *Wrapper) Group(fn func(r fchi.Router)) fchi.Router {
	im := r.With()

	if fn != nil {
		fn(im)
	}

	return im
}

// Route mounts a sub-router along a `basePattern` string.
func (r *Wrapper) Route(pattern string, fn func(r fchi.Router)) fchi.Router {
	subRouter := r.copy(fchi.NewRouter(), pattern)

	if fn != nil {
		fn(subRouter)
	}

	r.Router.Mount(pattern, subRouter)

	return subRouter
}

// Mount attaches another Handler along "./basePattern/*".
func (r *Wrapper) Mount(pattern string, h fchi.Handler) {
	h = r.prepareHandler("", pattern, h)
	r.captureHandler(h)
	r.Router.Mount(pattern, h)
}

// Handle adds routes for `basePattern` that matches all HTTP methods.
func (r *Wrapper) Handle(pattern string, h fchi.Handler) {
	h = r.prepareHandler("", pattern, h)
	r.captureHandler(h)
	r.Router.Handle(pattern, h)
}

// Method adds routes for `basePattern` that matches the `method` HTTP method.
func (r *Wrapper) Method(method, pattern string, h fchi.Handler) {
	h = r.prepareHandler(method, pattern, h)
	r.captureHandler(h)
	r.Router.Method(method, pattern, h)
}

// Connect adds the route `pattern` that matches a CONNECT http method to execute the `h` fchi.Handler.
func (r *Wrapper) Connect(pattern string, h fchi.Handler) {
	r.Method(http.MethodConnect, pattern, h)
}

// Delete adds the route `pattern` that matches a DELETE http method to execute the `h` fchi.Handler.
func (r *Wrapper) Delete(pattern string, h fchi.Handler) {
	r.Method(http.MethodDelete, pattern, h)
}

// Get adds the route `pattern` that matches a GET http method to execute the `h` fchi.Handler.
func (r *Wrapper) Get(pattern string, h fchi.Handler) {
	r.Method(http.MethodGet, pattern, h)
}

// Head adds the route `pattern` that matches a HEAD http method to execute the `h` fchi.Handler.
func (r *Wrapper) Head(pattern string, h fchi.Handler) {
	r.Method(http.MethodHead, pattern, h)
}

// Options adds the route `pattern` that matches a OPTIONS http method to execute the `h` fchi.Handler.
func (r *Wrapper) Options(pattern string, h fchi.Handler) {
	r.Method(http.MethodOptions, pattern, h)
}

// Patch adds the route `pattern` that matches a PATCH http method to execute the `h` fchi.Handler.
func (r *Wrapper) Patch(pattern string, h fchi.Handler) {
	r.Method(http.MethodPatch, pattern, h)
}

// Post adds the route `pattern` that matches a POST http method to execute the `h` fchi.Handler.
func (r *Wrapper) Post(pattern string, h fchi.Handler) {
	r.Method(http.MethodPost, pattern, h)
}

// Put adds the route `pattern` that matches a PUT http method to execute the `h` fchi.Handler.
func (r *Wrapper) Put(pattern string, h fchi.Handler) {
	r.Method(http.MethodPut, pattern, h)
}

// Trace adds the route `pattern` that matches a TRACE http method to execute the `h` fchi.Handler.
func (r *Wrapper) Trace(pattern string, h fchi.Handler) {
	r.Method(http.MethodTrace, pattern, h)
}

func (r *Wrapper) resolvePattern(pattern string) string {
	return r.basePattern + strings.ReplaceAll(pattern, "/*/", "/")
}

func (r *Wrapper) captureHandler(h fchi.Handler) {
	nethttp.WrapHandler(h, r.middlewares...)
}

func (r *Wrapper) prepareHandler(method, pattern string, h fchi.Handler) fchi.Handler {
	mw := r.wraps
	mw = append(mw, nethttp.HandlerWithRouteMiddleware(method, r.resolvePattern(pattern)))
	h = nethttp.WrapHandler(h, mw...)

	return h
}
