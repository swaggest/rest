package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/valyala/fasthttp"
)

// Benchmark_jsonBody-4   	   29671	     37417 ns/op	       194 B:rcvd/op	       181 B:sent/op	     26705 rps	    6068 B/op	      58 allocs/op.
// Benchmark_jsonBody-4   	   29749	     35934 ns/op	       194 B:rcvd/op	       181 B:sent/op	     27829 rps	    6063 B/op	      57 allocs/op.
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
		return resp.StatusCode() == http.StatusOK
	})
}
