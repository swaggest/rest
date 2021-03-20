package chirouter_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
)

type HandlerWithFoo struct {
	http.Handler
}

func (h HandlerWithFoo) Foo() {}

type HandlerWithBar struct {
	http.Handler
}

func (h HandlerWithBar) Bar() {}

func (h HandlerWithBar) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if _, err := rw.Write([]byte("bar")); err != nil {
		panic(err)
	}

	h.Handler.ServeHTTP(rw, r)
}

func TestNewWrapper(t *testing.T) {
	var r chi.Router
	r = chi.NewRouter()

	r = chirouter.NewWrapper(r).With(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(handler.ServeHTTP)
	})

	mw := func(handler http.Handler) http.Handler {
		var bar interface{ Bar() }

		assert.False(t, nethttp.HandlerAs(handler, &bar))

		return HandlerWithBar{Handler: handler}
	}

	r.Use(mw)

	r.Group(func(r chi.Router) {
		r.Method(http.MethodPost,
			"/baz/{id}/",
			HandlerWithFoo{Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				val, err := chirouter.PathToURLValues(request)
				assert.NoError(t, err)
				assert.Equal(t, url.Values{"id": []string{"123"}}, val)
			})},
		)
	})

	r.Mount("/mount", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}))

	r.Route("/deeper/", func(r chi.Router) {
		r.Use(func(handler http.Handler) http.Handler {
			return handler
		})

		r.Get("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Head("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Post("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Put("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Trace("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Connect("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Options("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Patch("/foo", func(writer http.ResponseWriter, request *http.Request) {})
		r.Delete("/foo", func(writer http.ResponseWriter, request *http.Request) {})

		r.MethodFunc(http.MethodGet, "/cuux", func(writer http.ResponseWriter, request *http.Request) {})

		r.Handle("/bar", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {}))
	})

	for _, u := range []string{"/baz/123/", "/deeper/foo", "/mount/abc"} {
		req, err := http.NewRequest(http.MethodPost, u, nil)
		require.NoError(t, err)

		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)

		assert.Equal(t, "bar", rw.Body.String(), u)
	}
}
