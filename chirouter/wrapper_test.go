package chirouter_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
	"github.com/swaggest/rest-fasthttp"
	"github.com/swaggest/rest-fasthttp/chirouter"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/valyala/fasthttp"
)

type HandlerWithFoo struct {
	fchi.Handler
}

func (h HandlerWithFoo) Foo() {}

type HandlerWithBar struct {
	fchi.Handler
}

func (h HandlerWithFoo) ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	if _, err := rc.Write([]byte("foo")); err != nil {
		panic(err)
	}

	h.Handler.ServeHTTP(ctx, rc)
}

func (h HandlerWithBar) Bar() {}

func (h HandlerWithBar) ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	h.Handler.ServeHTTP(ctx, rc)

	if _, err := rc.Write([]byte("bar")); err != nil {
		panic(err)
	}
}

func TestNewWrapper(t *testing.T) {
	r := chirouter.NewWrapper(fchi.NewRouter()).With(func(handler fchi.Handler) fchi.Handler {
		return fchi.HandlerFunc(handler.ServeHTTP)
	})

	handlersCnt := 0
	totalCnt := 0

	mw := func(handler fchi.Handler) fchi.Handler {
		var (
			withRoute rest.HandlerWithRoute
			bar       interface{ Bar() }
			foo       interface{ Foo() }
		)

		totalCnt++

		if fhttp.HandlerAs(handler, &withRoute) {
			handlersCnt++

			assert.False(t, fhttp.HandlerAs(handler, &bar), "%s", handler)
			assert.True(t, fhttp.HandlerAs(handler, &foo), "%s", handler)
		}

		return HandlerWithBar{Handler: handler}
	}

	r.Use(mw)

	r.Group(func(r fchi.Router) {
		r.Method(http.MethodPost,
			"/baz/{id}/",
			HandlerWithFoo{Handler: fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				val := make(url.Values)
				err := chirouter.PathToURLValues(rc, val)
				assert.NoError(t, err)
				assert.Equal(t, url.Values{"id": []string{"123"}}, val)
			})},
		)
	})

	r.Mount("/mount",
		HandlerWithFoo{Handler: fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {})},
	)

	r.Route("/deeper", func(r fchi.Router) {
		r.Use(func(handler fchi.Handler) fchi.Handler {
			return HandlerWithFoo{Handler: handler}
		})

		r.Get("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Head("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Post("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Put("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Trace("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Connect("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Options("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Patch("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
		r.Delete("/foo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))

		r.Method(http.MethodGet, "/cuux", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))

		r.Handle("/bar", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))
	})

	for _, u := range []string{"/baz/123/", "/deeper/foo", "/mount/abc"} {
		rc := &fasthttp.RequestCtx{}
		rc.Request.SetRequestURI(u)
		rc.Request.Header.SetMethod(http.MethodPost)

		r.ServeHTTP(rc, rc)

		assert.Equal(t, "foobar", string(rc.Response.Body()), u)
	}

	assert.Equal(t, 14, handlersCnt)
	assert.Equal(t, 20, totalCnt)
}

func TestWrapper_Use_precedence(t *testing.T) {
	var log []string

	// Vanilla fchi router.
	cr := fchi.NewRouter()
	cr.Use(
		func(handler fchi.Handler) fchi.Handler {
			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				log = append(log, "cmw1 before")
				handler.ServeHTTP(ctx, rc)
				log = append(log, "cmw1 after")
			})
		},

		func(handler fchi.Handler) fchi.Handler {
			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				log = append(log, "cmw2 before")
				handler.ServeHTTP(ctx, rc)
				log = append(log, "cmw2 after")
			})
		},
	)

	// Wrapped chi router.
	wr := chirouter.NewWrapper(fchi.NewRouter())
	wr.Use(
		func(handler fchi.Handler) fchi.Handler {
			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				log = append(log, "wmw1 before")
				handler.ServeHTTP(ctx, rc)
				log = append(log, "wmw1 after")
			})
		},

		func(handler fchi.Handler) fchi.Handler {
			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				log = append(log, "wmw2 before")
				handler.ServeHTTP(ctx, rc)
				log = append(log, "wmw2 after")
			})
		},
	)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	h := fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		log = append(log, "h")
	})

	// Both routers should invoke middlewares in the same order.
	cr.Method(http.MethodGet, "/", h)
	wr.Method(http.MethodGet, "/", h)

	cr.ServeHTTP(rc, rc)
	wr.ServeHTTP(rc, rc)
	assert.Equal(t, []string{
		"cmw1 before", "cmw2 before", "h", "cmw2 after", "cmw1 after",
		"wmw1 before", "wmw2 before", "h", "wmw2 after", "wmw1 after",
	}, log)
}

// This test covers original behavior discrepancy between wrapper and router
// in how middlewares are applied.
// Router runs middlewares for every request that comes in before route matching,
// and so middlewares like StripSlashes can affect route matching in a useful way.
// Wrapper in contrast was creating a new handler by running middlewares during handler
// registration, and then adding prepared handler to router.
// In this case middlewares were "baked-in" the handler and so, were running only
// after route match.
// For the use case of StripSlashes that would result in not found, because middleware was
// invoked AFTER route matching, not BEFORE.

// Solution to this problem was passing middlewares to Router as is, the problem however is
// that Router does not allow unwrapping handlers (that is the purpose of Wrapper) to introspect
// or augment handlers.
func TestWrapper_Use_StripSlashes(t *testing.T) {
	var log []string

	r := fchi.NewRouter()
	r.Use(
		middleware.StripSlashes,

		func(handler fchi.Handler) fchi.Handler {
			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				handler.ServeHTTP(ctx, rc)
			})
		},
	)

	// Wrapped chi router.
	wr := chirouter.NewWrapper(fchi.NewRouter())
	wr.Use(
		middleware.StripSlashes,

		func(handler fchi.Handler) fchi.Handler {
			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				handler.ServeHTTP(ctx, rc)
			})
		},
	)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/foo/")

	h := fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		if _, err := rc.Write([]byte("OK")); err != nil {
			log = append(log, err.Error())
		}

		log = append(log, "h")
	})

	r.Method(http.MethodGet, "/foo", h)
	r.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "OK", string(rc.Response.Body()))

	rc.Response = fasthttp.Response{}

	wr.Method(http.MethodGet, "/foo", h)
	wr.ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "OK", string(rc.Response.Body()))

	assert.Equal(t, []string{
		"h", "h",
	}, log)
}
