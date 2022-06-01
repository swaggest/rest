//go:build go1.18

package main

import (
	"net/http"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

// Benchmark_validation-4   	   18979	     53012 ns/op	       197 B:rcvd/op	       170 B:sent/op	     18861 rps	   14817 B/op	     131 allocs/op.
// Benchmark_validation-4   	   17665	     58243 ns/op	       177 B:rcvd/op	       170 B:sent/op	     17161 rps	   16349 B/op	     132 allocs/op.
func Benchmark_validation(b *testing.B) {
	r := NewRouter()

	srv := fchi.NewTestServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodPost)
		req.SetRequestURI(srv.URL + "/validation?q=true")
		req.Header.Set("X-Input", "12")
		req.Header.Set("Content-Type", "application/json")
		req.SetBody([]byte(`{"data":{"value":"abc"}}`))
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}

func Benchmark_noValidation(b *testing.B) {
	r := NewRouter()

	srv := fchi.NewTestServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodPost)
		req.SetRequestURI(srv.URL + "/no-validation?q=true")
		req.Header.Set("X-Input", "12")
		req.Header.Set("Content-Type", "application/json")
		req.SetBody([]byte(`{"data":{"value":"abc"}}`))
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}
