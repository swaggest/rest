package chirouter

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/swaggest/rest/nethttp"
)

// NewWrapper creates router wrapper to upgrade middlewares processing.
func NewWrapper(r chi.Router) *Wrapper {
	return &Wrapper{
		Router: r,
	}
}

// Wrapper wraps chi.Router to enable unwrappable handlers in middlewares.
//
// Middlewares can call nethttp.HandlerAs to inspect wrapped handlers.
type Wrapper struct {
	chi.Router
	name        string
	basePattern string

	middlewares []func(http.Handler) http.Handler
}

var _ chi.Router = &Wrapper{}

func (r *Wrapper) copy(router chi.Router, pattern string) *Wrapper {
	return &Wrapper{
		Router:      router,
		name:        r.name,
		basePattern: r.basePattern + pattern,
		middlewares: r.middlewares,
	}
}

// Use appends one of more middlewares onto the Router stack.
func (r *Wrapper) Use(middlewares ...func(http.Handler) http.Handler) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// With adds inline middlewares for an endpoint handler.
func (r Wrapper) With(middlewares ...func(http.Handler) http.Handler) chi.Router {
	c := r.copy(r.Router, "")
	c.Use(middlewares...)

	return c
}

// Group adds a new inline-router along the current routing path, with a fresh middleware stack for the inline-router.
func (r *Wrapper) Group(fn func(r chi.Router)) chi.Router {
	im := r.With()

	if fn != nil {
		fn(im)
	}

	return im
}

// Route mounts a sub-router along a `basePattern` string.
func (r *Wrapper) Route(pattern string, fn func(r chi.Router)) chi.Router {
	subRouter := r.copy(chi.NewRouter(), pattern)

	if fn != nil {
		fn(subRouter)
	}

	r.Router.Mount(pattern, subRouter)

	return subRouter
}

// Mount attaches another http.Handler along "./basePattern/*".
func (r *Wrapper) Mount(pattern string, h http.Handler) {
	p := r.prepareHandler("", pattern, h)
	r.Router.Mount(pattern, p)
}

// Handle adds routes for `basePattern` that matches all HTTP methods.
func (r *Wrapper) Handle(pattern string, h http.Handler) {
	r.Router.Handle(pattern, r.prepareHandler("", pattern, h))
}

// Method adds routes for `basePattern` that matches the `method` HTTP method.
func (r *Wrapper) Method(method, pattern string, h http.Handler) {
	r.Router.Method(method, pattern, r.prepareHandler(method, pattern, h))
}

// MethodFunc adds the route `pattern` that matches `method` http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) MethodFunc(method, pattern string, handlerFn http.HandlerFunc) {
	r.Method(method, pattern, handlerFn)
}

// Connect adds the route `pattern` that matches a CONNECT http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Connect(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodConnect, pattern, handlerFn)
}

// Delete adds the route `pattern` that matches a DELETE http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Delete(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodDelete, pattern, handlerFn)
}

// Get adds the route `pattern` that matches a GET http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Get(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodGet, pattern, handlerFn)
}

// Head adds the route `pattern` that matches a HEAD http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Head(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodHead, pattern, handlerFn)
}

// Options adds the route `pattern` that matches a OPTIONS http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Options(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodOptions, pattern, handlerFn)
}

// Patch adds the route `pattern` that matches a PATCH http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Patch(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodPatch, pattern, handlerFn)
}

// Post adds the route `pattern` that matches a POST http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Post(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodPost, pattern, handlerFn)
}

// Put adds the route `pattern` that matches a PUT http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Put(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodPut, pattern, handlerFn)
}

// Trace adds the route `pattern` that matches a TRACE http method to execute the `handlerFn` http.HandlerFunc.
func (r *Wrapper) Trace(pattern string, handlerFn http.HandlerFunc) {
	r.Method(http.MethodTrace, pattern, handlerFn)
}

func (r *Wrapper) resolvePattern(pattern string) string {
	return r.basePattern + strings.ReplaceAll(pattern, "/*/", "/")
}

func (r *Wrapper) prepareHandler(method, pattern string, h http.Handler) http.Handler {
	mw := append(r.middlewares, nethttp.HandlerWithRouteMiddleware(method, r.resolvePattern(pattern)))
	h = nethttp.WrapHandler(h, mw...)

	return h
}
