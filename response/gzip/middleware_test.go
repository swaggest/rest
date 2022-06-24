package gzip_test

import (
	"bytes"
	gz "compress/gzip"
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/fchi"
	gzip2 "github.com/swaggest/rest-fasthttp/gzip"
	"github.com/swaggest/rest-fasthttp/response/gzip"
	"github.com/valyala/fasthttp"
)

func TestMiddleware(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := gzip.Middleware(fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		_, err := rc.Write(resp)
		assert.NoError(t, err)
	}))

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, "gzip", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Less(t, len(rc.Response.Body()), len(resp)) // Response is compressed.
	assert.Equal(t, resp, gzipDecode(t, rc.Response.Body()))

	rc = &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, "gzip", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Less(t, len(rc.Response.Body()), len(resp)) // Response is compressed.
	assert.Equal(t, resp, gzipDecode(t, rc.Response.Body()))

	rc = &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "br")
	h.ServeHTTP(rc, rc)

	assert.Equal(t, "", string(rc.Response.Header.Peek("Content-Encoding")))
	require.Equal(t, len(rc.Response.Body()), len(resp)) // Response is not compressed.
	assert.Equal(t, resp, rc.Response.Body())

	rc = &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	h.ServeHTTP(rc, rc)

	assert.Equal(t, "", string(rc.Response.Header.Peek("Content-Encoding")))
	require.Equal(t, len(rc.Response.Body()), len(resp)) // Response is not compressed.
	assert.Equal(t, resp, rc.Response.Body())
}

// BenchmarkMiddleware measures performance of handler with compression.
//
// Sample result:
// BenchmarkMiddleware-12    	  108810	      9619 ns/op	    1223 B/op	      11 allocs/op.
func BenchmarkMiddleware(b *testing.B) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := gzip.Middleware(fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		_, err := rc.Write(resp)
		assert.NoError(b, err)
	}))

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rc.Response = fasthttp.Response{}

		h.ServeHTTP(rc, rc)
	}
}

// BenchmarkMiddleware_control measures performance of handler without compression.
//
// Sample result:
// BenchmarkMiddleware_control-4   	  214824	      5945 ns/op	   11184 B/op	       9 allocs/op.
func BenchmarkMiddleware_control(b *testing.B) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		_, err := rc.Write(resp)
		assert.NoError(b, err)
	})

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rc.Response = fasthttp.Response{}
		h.ServeHTTP(rc, rc)
	}
}

func TestMiddleware_concurrency(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	respGz := gzipEncode(t, resp)
	h := gzip.Middleware(fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		_, err := rc.Write(resp)
		assert.NoError(t, err)
	}))

	hg := gzip.Middleware(fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		_, err := gzip2.WriteCompressedBytes(respGz, rc)
		assert.NoError(t, err)
	}))

	n := 100
	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			rc := &fasthttp.RequestCtx{}
			rc.Request.SetRequestURI("/")
			rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

			h.ServeHTTP(rc, rc)

			assert.Equal(t, "gzip", string(rc.Response.Header.Peek("Content-Encoding")))
			assert.Less(t, len(rc.Response.Body()), len(resp)) // Response is compressed.
			assert.Equal(t, resp, gzipDecode(t, rc.Response.Body()))

			rc.Response = fasthttp.Response{}

			hg.ServeHTTP(rc, rc)

			assert.Equal(t, "gzip", string(rc.Response.Header.Peek("Content-Encoding")))
			assert.Less(t, len(rc.Response.Body()), len(resp)) // Response is compressed.
			assert.True(t, bytes.Equal(resp, gzipDecode(t, rc.Response.Body())))
		}()
	}

	wg.Wait()
}

func TestGzipResponseWriter_ExpectCompressedBytes(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	respGz := gzipEncode(t, resp)

	h := gzip.Middleware(fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		_, err := gzip2.WriteCompressedBytes(respGz, rc)
		assert.NoError(t, err)
	}))

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, "gzip", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Less(t, len(rc.Response.Body()), len(resp)) // Response is compressed.
	assert.Equal(t, respGz, rc.Response.Body())
}

func TestMiddleware_skipContentEncoding(t *testing.T) {
	resp := []byte(strings.Repeat("A", 10000) + "!!!")
	h := gzip.Middleware(fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.Header.Set("Content-Encoding", "br")
		_, err := rc.Write(resp)
		assert.NoError(t, err)
	}))

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, "br", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Equal(t, len(rc.Response.Body()), len(resp)) // Response is not compressed.
	assert.Equal(t, resp, rc.Response.Body())
}

func TestMiddleware_noContent(t *testing.T) {
	h := gzip.Middleware(fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(http.StatusNoContent)

		// Second call does not hurt.
		rc.Response.SetStatusCode(http.StatusNoContent)
	}))

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")
	rc.Request.Header.Set("Accept-Encoding", "gzip, deflate, br")

	h.ServeHTTP(rc, rc)

	assert.Equal(t, "", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Equal(t, len(rc.Response.Body()), 0)
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

	j, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	require.NoError(t, r.Close())

	return j
}
