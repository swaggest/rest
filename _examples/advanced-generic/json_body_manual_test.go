//go:build go1.18

package main

import (
	"net/http"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

// Benchmark_jsonBodyManual-12    	  147058	      8812 ns/op	       226.0 B:rcvd/op	       195.0 B:sent/op	    113469 rps	     728 B/op	      18 allocs/op.
func Benchmark_jsonBodyManual(b *testing.B) {
	r := NewRouter()

	srv := fchi.NewTestServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodPost)
		req.SetRequestURI(srv.URL + "/json-body-manual/abc?in_query=2006-01-02")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Header", "def")
		req.SetBody([]byte(`{"id":321,"name":"Jane"}`))
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusCreated
	})
}
