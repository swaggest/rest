package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/valyala/fasthttp"
)

// Benchmark_validation-4   	   18979	     53012 ns/op	       197 B:rcvd/op	       170 B:sent/op	     18861 rps	   14817 B/op	     131 allocs/op.
// Benchmark_validation-4   	   17665	     58243 ns/op	       177 B:rcvd/op	       170 B:sent/op	     17161 rps	   16349 B/op	     132 allocs/op.
func Benchmark_validation(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
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

	srv := httptest.NewServer(r)
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

func TestNewRouter_validation(t *testing.T) {
	r := NewRouter()

	t.Run("invalid_request_headers", func(t *testing.T) {
		// Invalid response header.
		req, err := http.NewRequest(http.MethodPost, "/validation",
			bytes.NewReader([]byte(`{"data":{"value":"valid"}}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Input", "5")

		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Equal(t, "application/problem+json", rw.Header().Get("Content-Type"))
		assertjson.EqualMarshal(t, []byte(`{
		  "msg":"invalid argument: validation failed",
		  "details":{"header:X-Input":["#: must be \u003e= 10/1 but found 5"]}
		}`), json.RawMessage(rw.Body.Bytes()))
	})

	t.Run("invalid_request_body", func(t *testing.T) {
		// Invalid response header.
		req, err := http.NewRequest(http.MethodPost, "/validation",
			bytes.NewReader([]byte(`{"data":{"value":"a"}}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Input", "15")

		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Equal(t, "application/problem+json", rw.Header().Get("Content-Type"))
		assertjson.EqualMarshal(t, []byte(`{
		  "msg":"invalid argument: validation failed",
		  "details":{"body":["#/data/value: length must be \u003e= 3, but got 1"]}
		}`), json.RawMessage(rw.Body.Bytes()))
	})

	t.Run("invalid_response_headers", func(t *testing.T) {
		// Invalid response header.
		req, err := http.NewRequest(http.MethodPost, "/validation",
			bytes.NewReader([]byte(`{"data":{"value":"foo"}}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Input", "45")

		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Equal(t, "application/problem+json", rw.Header().Get("Content-Type"))
		assertjson.EqualMarshal(t, []byte(`{
		  "msg":"internal: bad response: validation failed",
		  "details":{"header:X-Output":["#: must be \u003c= 20/1 but found 45"]}
		}`), json.RawMessage(rw.Body.Bytes()))
	})

	t.Run("invalid_response_body", func(t *testing.T) {
		rw := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodPost, "/validation",
			bytes.NewReader([]byte(`{"data":{"value":"toooo long"}}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Input", "15")

		r.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusInternalServerError, rw.Code)
		assert.Equal(t, "application/problem+json", rw.Header().Get("Content-Type"))
		assertjson.EqualMarshal(t, []byte(`{
		  "msg":"internal: bad response: validation failed",
		  "details":{"body":["#/data/value: length must be \u003c= 7, but got 10"]}
		}`), json.RawMessage(rw.Body.Bytes()))
	})

	t.Run("invalid_response_body", func(t *testing.T) {
		rw := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodPost, "/validation",
			bytes.NewReader([]byte(`{"data":{"value":"good"}}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Input", "15")

		r.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		assert.Equal(t, "application/dummy+json", rw.Header().Get("Content-Type"))
		assertjson.EqualMarshal(t, []byte(`{"data":{"value":"good"}}`), json.RawMessage(rw.Body.Bytes()))
	})
}
