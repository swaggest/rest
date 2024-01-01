package chirouter

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/swaggest/rest"
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
	wraps       []func(http.Handler) http.Handler
	handlers    []http.Handler
}

var _ chi.Router = &Wrapper{}

func (r *Wrapper) copy(router chi.Router, pattern string) *Wrapper {
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
func (r *Wrapper) Wrap(wraps ...func(handler http.Handler) http.Handler) {
	r.wraps = append(r.wraps, wraps...)
}

// Use appends one of more middlewares onto the Router stack.
func (r *Wrapper) Use(middlewares ...func(http.Handler) http.Handler) {
	var mws []func(http.Handler) http.Handler

	for _, mw := range middlewares {
		if nethttp.MiddlewareIsWrapper(mw) {
			r.wraps = append(r.wraps, mw)
		} else {
			mws = append(mws, mw)
		}
	}

	r.Router.Use(mws...)
	r.middlewares = append(r.middlewares, mws...)
}

// With adds inline middlewares for an endpoint handler.
func (r Wrapper) With(middlewares ...func(http.Handler) http.Handler) chi.Router {
	var mws, ws []func(http.Handler) http.Handler

	for _, mw := range middlewares {
		if nethttp.MiddlewareIsWrapper(mw) {
			ws = append(ws, mw)
		} else {
			mws = append(mws, mw)
		}
	}

	c := r.copy(r.Router.With(mws...), "")
	c.wraps = append(c.wraps, ws...)

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
	if hr, ok := h.(interface {
		handlersWithRoute() []http.Handler
		handlerWraps() []func(http.Handler) http.Handler
	}); ok {
		pattern = strings.TrimSuffix(pattern, "/")

		for _, h := range hr.handlersWithRoute() {
			var rh rest.HandlerWithRoute
			if nethttp.HandlerAs(h, &rh) {
				m := rh.RouteMethod()
				p := r.resolvePattern(pattern + rh.RoutePattern())
				h := nethttp.WrapHandler(h, nethttp.HandlerWithRouteMiddleware(m, p))
				h = nethttp.WrapHandler(h, hr.handlerWraps()...)
				_ = nethttp.WrapHandler(h, r.wraps...)
			}
		}
	} else {
		h = r.prepareHandler("", pattern, h)
		r.captureHandler(h)
	}

	r.Router.Mount(pattern, h)
}

// Handle adds routes for `basePattern` that matches all HTTP methods.
func (r *Wrapper) Handle(pattern string, h http.Handler) {
	h = r.prepareHandler("", pattern, h)
	r.captureHandler(h)
	r.Router.Handle(pattern, h)
}

// Method adds routes for `basePattern` that matches the `method` HTTP method.
func (r *Wrapper) Method(method, pattern string, h http.Handler) {
	h = r.prepareHandler(method, pattern, h)
	r.captureHandler(h)
	r.Router.Method(method, pattern, h)
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

// HandlerFunc prepares handler and returns its function.
//
// Can be used as input for NotFound, MethodNotAllowed.
func (r *Wrapper) HandlerFunc(h http.Handler) http.HandlerFunc {
	h = nethttp.WrapHandler(h, r.wraps...)

	return h.ServeHTTP
}

func (r *Wrapper) resolvePattern(pattern string) string {
	return r.basePattern + strings.ReplaceAll(pattern, "/*/", "/")
}

func (r *Wrapper) captureHandler(h http.Handler) {
	nethttp.WrapHandler(h, r.middlewares...)
}

func (r *Wrapper) prepareHandler(method, pattern string, h http.Handler) http.Handler {
	mw := nethttp.HandlerWithRouteMiddleware(method, r.resolvePattern(pattern))
	h = nethttp.WrapHandler(h, mw)
	r.handlers = append(r.handlers, h)
	h = nethttp.WrapHandler(h, r.wraps...)

	return h
}

func (r *Wrapper) handlersWithRoute() []http.Handler {
	return r.handlers
}

func (r *Wrapper) handlerWraps() []func(http.Handler) http.Handler {
	return r.wraps
}
