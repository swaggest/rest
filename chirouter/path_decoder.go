package chirouter

import (
	"net/url"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

// PathToURLValues is a decoder function for parameters in path.
func PathToURLValues(rc *fasthttp.RequestCtx, params url.Values) error {
	if routeCtx := fchi.RouteContext(rc); routeCtx != nil {
		for i, key := range routeCtx.URLParams.Keys {
			value, err := url.PathUnescape(routeCtx.URLParams.Values[i])
			if err != nil {
				return err
			}

			params[key] = []string{value}
		}

		return nil
	}

	return nil
}
