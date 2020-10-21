package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/valyala/fasthttp"
)

// Benchmark_jsonBodyValidation-4   	   19126	     60170 ns/op	       194 B:rcvd/op	       192 B:sent/op	     16620 rps	   18363 B/op	     144 allocs/op.
func Benchmark_jsonBodyValidation(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodPost)
		req.SetRequestURI(srv.URL + "/json-body-validation/abc?in_query=123")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Header", "def")
		req.SetBody([]byte(`{"id":321,"name":"Jane"}`))
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}
