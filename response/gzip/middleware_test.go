package gzip_test

import (
	"bytes"
	gz "compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gzip2 "github.com/swaggest/rest/gzip"
	"github.com/swaggest/rest/response/gzip"
)

func TestMiddleware(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := gzip.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(resp)
		assert.NoError(t, err)
	}))

	rw := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)

	require.NoError(t, err)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rw, r)

	assert.Equal(t, "gzip", rw.Header().Get("Content-Encoding"))
	assert.Less(t, rw.Body.Len(), len(resp)) // Response is compressed.
	assert.Equal(t, resp, gzipDecode(t, rw.Body.Bytes()))

	rw = httptest.NewRecorder()
	h.ServeHTTP(rw, r)

	assert.Equal(t, "gzip", rw.Header().Get("Content-Encoding"))
	assert.Less(t, rw.Body.Len(), len(resp)) // Response is compressed.
	assert.Equal(t, resp, gzipDecode(t, rw.Body.Bytes()))

	rw = httptest.NewRecorder()

	r.Header.Set("Accept-Encoding", "deflate, br")
	h.ServeHTTP(rw, r)

	assert.Equal(t, "", rw.Header().Get("Content-Encoding"))
	assert.Equal(t, rw.Body.Len(), len(resp)) // Response is not compressed.
	assert.Equal(t, resp, rw.Body.Bytes())

	rw = httptest.NewRecorder()

	r.Header.Del("Accept-Encoding")
	h.ServeHTTP(rw, r)

	assert.Equal(t, "", rw.Header().Get("Content-Encoding"))
	assert.Equal(t, rw.Body.Len(), len(resp)) // Response is not compressed.
	assert.Equal(t, resp, rw.Body.Bytes())
}

// BenchmarkMiddleware measures performance of handler with compression.
//
// Sample result:
// BenchmarkMiddleware-12    	  108810	      9619 ns/op	    1223 B/op	      11 allocs/op.
func BenchmarkMiddleware(b *testing.B) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := gzip.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(resp)
		assert.NoError(b, err)
	}))

	r, err := http.NewRequest(http.MethodGet, "/", nil)

	require.NoError(b, err)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, r)
	}
}

// BenchmarkMiddleware_control measures performance of handler without compression.
//
// Sample result:
// BenchmarkMiddleware_control-4   	  214824	      5945 ns/op	   11184 B/op	       9 allocs/op.
func BenchmarkMiddleware_control(b *testing.B) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(resp)
		assert.NoError(b, err)
	})

	r, err := http.NewRequest(http.MethodGet, "/", nil)

	require.NoError(b, err)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, r)
	}
}

func TestMiddleware_concurrency(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	respGz := gzipEncode(t, resp)
	h := gzip.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(resp)
		assert.NoError(t, err)
	}))

	hg := gzip.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := gzip2.WriteCompressedBytes(respGz, rw)
		assert.NoError(t, err)
	}))

	n := 100
	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			rw := httptest.NewRecorder()
			r, err := http.NewRequest(http.MethodGet, "/", nil)

			require.NoError(t, err)
			r.Header.Set("Accept-Encoding", "gzip, deflate, br")

			h.ServeHTTP(rw, r)

			assert.Equal(t, "gzip", rw.Header().Get("Content-Encoding"))
			assert.Less(t, rw.Body.Len(), len(resp)) // Response is compressed.
			assert.Equal(t, resp, gzipDecode(t, rw.Body.Bytes()))

			rw = httptest.NewRecorder()

			hg.ServeHTTP(rw, r)

			assert.Equal(t, "gzip", rw.Header().Get("Content-Encoding"))
			assert.Less(t, rw.Body.Len(), len(resp)) // Response is compressed.
			assert.Equal(t, respGz, rw.Body.Bytes())
		}()
	}

	wg.Wait()
}

func TestGzipResponseWriter_ExpectCompressedBytes(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	respGz := gzipEncode(t, resp)

	h := gzip.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := gzip2.WriteCompressedBytes(respGz, rw)
		assert.NoError(t, err)
	}))

	rw := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)

	require.NoError(t, err)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rw, r)

	assert.Equal(t, "gzip", rw.Header().Get("Content-Encoding"))
	assert.Less(t, rw.Body.Len(), len(resp)) // Response is compressed.
	assert.Equal(t, respGz, rw.Body.Bytes())
}

func TestMiddleware_skipContentEncoding(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := gzip.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Encoding", "br")
		_, err := rw.Write(resp)
		assert.NoError(t, err)
	}))

	rw := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)

	require.NoError(t, err)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rw, r)

	assert.Equal(t, "br", rw.Header().Get("Content-Encoding"))
	assert.Equal(t, rw.Body.Len(), len(resp)) // Response is not compressed.
	assert.Equal(t, resp, rw.Body.Bytes())
}

func TestMiddleware_noContent(t *testing.T) {
	h := gzip.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNoContent)

		// Second call does not hurt.
		rw.WriteHeader(http.StatusNoContent)
	}))

	rw := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)

	require.NoError(t, err)
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rw, r)

	assert.Equal(t, "", rw.Header().Get("Content-Encoding"))
	assert.Equal(t, rw.Body.Len(), 0)
}

func gzipEncode(t *testing.T, data []byte) []byte {
	t.Helper()

	b := bytes.Buffer{}
	w := gz.NewWriter(&b)

	_, err := w.Write(data)
	require.NoError(t, err)

	require.NoError(t, w.Close())

	return b.Bytes()
}

func gzipDecode(t *testing.T, data []byte) []byte {
	t.Helper()

	b := bytes.NewReader(data)

	r, err := gz.NewReader(b)
	require.NoError(t, err)

	j, err := io.ReadAll(r)
	require.NoError(t, err)

	require.NoError(t, r.Close())

	return j
}
