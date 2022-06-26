//go:build go1.18

package main

import (
	"net/http"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

// Benchmark_jsonBody-12    	   68124	     17828 ns/op	       226.0 B:rcvd/op	       188.0 B:sent/op	     56083 rps	    6864 B/op	      85 allocs/op.
func Benchmark_jsonBody(b *testing.B) {
	r := NewRouter()

	srv := fchi.NewTestServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodPost)
		req.SetRequestURI(srv.URL + "/json-body/abc?in_query=2006-01-02")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Header", "def")
		req.SetBody([]byte(`{"id":321,"name":"Jane"}`))
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusCreated
	})
}
