package nethttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest/nethttp"
)

func TestMiddlewareIsWrapper(t *testing.T) {
	wrapper := func(handler http.Handler) http.Handler {
		if nethttp.IsWrapperChecker(handler) {
			return handler
		}

		return handler
	}

	notWrapper := func(handler http.Handler) http.Handler {
		return handler
	}

	assert.True(t, nethttp.MiddlewareIsWrapper(wrapper))
	assert.False(t, nethttp.MiddlewareIsWrapper(notWrapper))
}
