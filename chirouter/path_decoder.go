package chirouter

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

// PathToURLValues is a decoder function for parameters in path.
func PathToURLValues(r *http.Request) (url.Values, error) {
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		params := make(url.Values, len(routeCtx.URLParams.Keys))

		for i, key := range routeCtx.URLParams.Keys {
			value, err := url.PathUnescape(routeCtx.URLParams.Values[i])
			if err != nil {
				return nil, err
			}

			params[key] = []string{value}
		}

		return params, nil
	}

	return nil, nil
}
