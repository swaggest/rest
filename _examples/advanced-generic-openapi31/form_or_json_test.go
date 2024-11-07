package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/valyala/fasthttp"
)

func Benchmark_formOrJSON(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	b.Run("form", func(b *testing.B) {
		httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
			req.Header.SetMethod(http.MethodPost)
			req.SetRequestURI(srv.URL + "/form-or-json/abc")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetBody([]byte(`field1=def&field2=123`))
		}, func(i int, resp *fasthttp.Response) bool {
			return resp.StatusCode() == http.StatusOK
		})
	})

	b.Run("json", func(b *testing.B) {
		httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
			req.Header.SetMethod(http.MethodPost)
			req.SetRequestURI(srv.URL + "/form-or-json/abc")
			req.Header.Set("Content-Type", "application/json")
			req.SetBody([]byte(`{"field1":"string","field2":0}`))
		}, func(i int, resp *fasthttp.Response) bool {
			return resp.StatusCode() == http.StatusOK
		})
	})
}
