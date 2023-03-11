package chirouter

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
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
func (r Wrapper) With(middlewares ...func(http.Handler) http.Handler) *Wrapper {
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
func (r *Wrapper) Group(fn func(r *Wrapper)) *Wrapper {
	im := r.With()

	if fn != nil {
		fn(im)
	}

	return im
}

// Route mounts a sub-router along a `basePattern` string.
func (r *Wrapper) Route(pattern string, fn func(r *Wrapper)) *Wrapper {
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

func (r *Wrapper) Delete(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodDelete, pattern, nethttp.NewHandler(uc, options...))
}

// Get adds the route `pattern` that matches a GET http method to invoke use case interactor.
func (r *Wrapper) Get(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodGet, pattern, nethttp.NewHandler(uc, options...))
}

// Head adds the route `pattern` that matches a HEAD http method to invoke use case interactor.
func (r *Wrapper) Head(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodHead, pattern, nethttp.NewHandler(uc, options...))
}

// Options adds the route `pattern` that matches a OPTIONS http method to invoke use case interactor.
func (r *Wrapper) Options(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodOptions, pattern, nethttp.NewHandler(uc, options...))
}

// Patch adds the route `pattern` that matches a PATCH http method to invoke use case interactor.
func (r *Wrapper) Patch(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodPatch, pattern, nethttp.NewHandler(uc, options...))
}

// Post adds the route `pattern` that matches a POST http method to invoke use case interactor.
func (r *Wrapper) Post(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodPost, pattern, nethttp.NewHandler(uc, options...))
}

// Put adds the route `pattern` that matches a PUT http method to invoke use case interactor.
func (r *Wrapper) Put(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodPut, pattern, nethttp.NewHandler(uc, options...))
}

// Trace adds the route `pattern` that matches a TRACE http method to invoke use case interactor.
func (r *Wrapper) Trace(pattern string, uc usecase.Interactor, options ...func(h *nethttp.Handler)) {
	r.Method(http.MethodTrace, pattern, nethttp.NewHandler(uc, options...))
}

func (r *Wrapper) resolvePattern(pattern string) string {
	return r.basePattern + strings.ReplaceAll(pattern, "/*/", "/")
}

func (r *Wrapper) captureHandler(h http.Handler) {
	nethttp.WrapHandler(h, r.middlewares...)
}

func (r *Wrapper) prepareHandler(method, pattern string, h http.Handler) http.Handler {
	h = nethttp.WrapHandler(h, nethttp.HandlerWithRouteMiddleware(method, r.resolvePattern(pattern)))
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
