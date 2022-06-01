package gzip

import (
	"context"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

// Middleware enables gzip compression of handler response for requests that accept gzip encoding.
func Middleware(next fchi.Handler) fchi.Handler {
	f := fasthttp.CompressHandler(func(rc *fasthttp.RequestCtx) {
		next.ServeHTTP(rc, rc)
	})

	return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		f(rc)
	})
}
