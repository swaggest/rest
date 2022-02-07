package nethttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest/nethttp"
)

func TestWrapHandler(t *testing.T) {
	var flow []string

	h := nethttp.WrapHandler(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			flow = append(flow, "handler")
		}),
		func(handler http.Handler) http.Handler {
			flow = append(flow, "mw1 registered")

			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				flow = append(flow, "mw1 before")
				handler.ServeHTTP(writer, request)
				flow = append(flow, "mw1 after")
			})
		},
		func(handler http.Handler) http.Handler {
			flow = append(flow, "mw2 registered")

			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				flow = append(flow, "mw2 before")
				handler.ServeHTTP(writer, request)
				flow = append(flow, "mw2 after")
			})
		},
		func(handler http.Handler) http.Handler {
			flow = append(flow, "mw3 registered")

			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				flow = append(flow, "mw3 before")
				handler.ServeHTTP(writer, request)
				flow = append(flow, "mw3 after")
			})
		},
	)

	h.ServeHTTP(nil, nil)

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
