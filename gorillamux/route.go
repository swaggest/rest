package gorillamux

import (
	"net/http"

	"github.com/gorilla/mux"
)

type WrappedRoute struct {
	*mux.Route
}

func (r *WrappedRoute) Name(name string) *WrappedRoute {
	r.Route.Name(name)

	return r
}

func (r *WrappedRoute) Handler(handler http.Handler) *WrappedRoute {
	r.Route.Handler(handler)

	return r
}

func (r *WrappedRoute) Path(path string) *WrappedRoute {
	r.Route.Path(path)

	return r
}

func (r *WrappedRoute) HandlerFunc(f func(http.ResponseWriter, *http.Request)) *WrappedRoute {
	r.Route.HandlerFunc(f)

	return r
}

func (r *WrappedRoute) Headers(pairs ...string) *WrappedRoute {
	r.Route.Headers(pairs...)

	return r
}

func (r *WrappedRoute) Host(tpl string) *WrappedRoute {
	r.Route.Host(tpl)

	return r
}

func (r *WrappedRoute) PathPrefix(tpl string) *WrappedRoute {
	r.Route.PathPrefix(tpl)

	return r
}

func (r *WrappedRoute) MatcherFunc(f mux.MatcherFunc) *WrappedRoute {
	r.Route.MatcherFunc(f)

	return r
}

func (r *WrappedRoute) Queries(pairs ...string) *WrappedRoute {
	r.Route.Queries(pairs...)

	return r
}

func (r *WrappedRoute) Methods(methods ...string) *WrappedRoute {
	r.Route.Methods(methods...)

	return r
}

func (r *WrappedRoute) Schemes(schemes ...string) *WrappedRoute {
	r.Route.Schemes(schemes...)

	return r
}

func (r *WrappedRoute) BuildVarsFunc(f mux.BuildVarsFunc) *WrappedRoute {
	r.Route.BuildVarsFunc(f)

	return r
}
