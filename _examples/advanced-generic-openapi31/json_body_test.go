//go:build go1.18

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/valyala/fasthttp"
)

// Benchmark_jsonBody-12    	   96762	     12042 ns/op	       208.0 B:rcvd/op	       188.0 B:sent/op	     83033 rps	   10312 B/op	     100 allocs/op.
func Benchmark_jsonBody(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
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
