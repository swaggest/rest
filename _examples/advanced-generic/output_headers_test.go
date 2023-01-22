//go:build go1.18

package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/valyala/fasthttp"
)

// Benchmark_outputHeaders-4   	   41424	     27054 ns/op	       154 B:rcvd/op	        77.0 B:sent/op	     36963 rps	    3641 B/op	      35 allocs/op.
func Benchmark_outputHeaders(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodGet)
		req.SetRequestURI(srv.URL + "/output-headers")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}

func Test_outputHeaders(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/output-headers")
	require.NoError(t, err)

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	assert.Equal(t, "abc", resp.Header.Get("X-Header"))
	assert.Equal(t, []string{"coo=123; HttpOnly"}, resp.Header.Values("Set-Cookie"))
	assertjson.Equal(t, []byte(`{"inBody":"def"}`), body)
}
