//go:build go1.18

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/valyala/fasthttp"
)

// Benchmark_jsonBodyManual-12    	  125672	      8542 ns/op	       208.0 B:rcvd/op	       195.0 B:sent/op	    117048 rps	    4523 B/op	      49 allocs/op.
func Benchmark_jsonBodyManual(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
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
