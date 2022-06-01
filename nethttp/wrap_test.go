package nethttp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest/nethttp"
	"github.com/valyala/fasthttp"
)

func TestWrapHandler(t *testing.T) {
	var flow []string

	h := nethttp.WrapHandler(
		fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			flow = append(flow, "handler")
		}),
		func(handler fchi.Handler) fchi.Handler {
			flow = append(flow, "mw1 registered")

			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				flow = append(flow, "mw1 before")
				handler.ServeHTTP(ctx, rc)
				flow = append(flow, "mw1 after")
			})
		},
		func(handler fchi.Handler) fchi.Handler {
			flow = append(flow, "mw2 registered")

			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				flow = append(flow, "mw2 before")
				handler.ServeHTTP(ctx, rc)
				flow = append(flow, "mw2 after")
			})
		},
		func(handler fchi.Handler) fchi.Handler {
			flow = append(flow, "mw3 registered")

			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				flow = append(flow, "mw3 before")
				handler.ServeHTTP(ctx, rc)
				flow = append(flow, "mw3 after")
			})
		},
	)

	h.ServeHTTP(context.Background(), nil)

	assert.Equal(t, []string{
		"mw3 registered", "mw2 registered", "mw1 registered",
		"mw1 before", "mw2 before", "mw3 before",
		"handler",
		"mw3 after", "mw2 after", "mw1 after",
	}, flow)
}

func TestHandlerAs_nil(t *testing.T) {
	var uh *nethttp.Handler

	assert.False(t, nethttp.HandlerAs(nil, &uh))
}
