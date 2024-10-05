package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func Test_directGzip(t *testing.T) {
	r := NewRouter()

	req, err := http.NewRequest(http.MethodGet, "/gzip-pass-through", nil)
	require.NoError(t, err)

	req.Header.Set("Accept-Encoding", "gzip")

	rw := httptest.NewRecorder()

	r.ServeHTTP(rw, req)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "330epditz19z", rw.Header().Get("Etag"))
	assert.Equal(t, "gzip", rw.Header().Get("Content-Encoding"))
	assert.Equal(t, "abc", rw.Header().Get("X-Header"))
	assert.Less(t, len(rw.Body.Bytes()), 500)
}

func Test_noDirectGzip(t *testing.T) {
	r := NewRouter()

	req, err := http.NewRequest(http.MethodGet, "/gzip-pass-through?plainStruct=1", nil)
	require.NoError(t, err)

	req.Header.Set("Accept-Encoding", "gzip")

	rw := httptest.NewRecorder()

	r.ServeHTTP(rw, req)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "", rw.Header().Get("Etag")) // No ETag for dynamic compression.
	assert.Equal(t, "gzip", rw.Header().Get("Content-Encoding"))
	assert.Equal(t, "cba", rw.Header().Get("X-Header"))
	assert.Less(t, len(rw.Body.Bytes()), 1000) // Worse compression for better speed.
}

func Test_directGzip_perf(t *testing.T) {
	res := testing.Benchmark(Benchmark_directGzip)

	if httptestbench.RaceDetectorEnabled {
		assert.Less(t, res.Extra["B:rcvd/op"], 640.0)
		assert.Less(t, res.Extra["B:sent/op"], 104.0)
		assert.Less(t, res.AllocsPerOp(), int64(60))
		assert.Less(t, res.AllocedBytesPerOp(), int64(9000))
	} else {
		assert.Less(t, res.Extra["B:rcvd/op"], 640.0)
		assert.Less(t, res.Extra["B:sent/op"], 104.0)
		assert.Less(t, res.AllocsPerOp(), int64(45))
		assert.Less(t, res.AllocedBytesPerOp(), int64(4200))
	}
}

// Direct gzip enabled.
// Benchmark_directGzip-4   	   48037	     24474 ns/op	       624 B:rcvd/op	       103 B:sent/op	     40860 rps	    3499 B/op	      36 allocs/op.
// Benchmark_directGzip-4   	   45792	     26102 ns/op	       624 B:rcvd/op	       103 B:sent/op	     38278 rps	    3063 B/op	      33 allocs/op.
func Benchmark_directGzip(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.Set("Accept-Encoding", "gzip")
		req.SetRequestURI(srv.URL + "/gzip-pass-through")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}

// Direct gzip enabled.
// Benchmark_directGzipHead-4   	   43804	     26481 ns/op	       168 B:rcvd/op	       104 B:sent/op	     37730 rps	    3507 B/op	      36 allocs/op.
// Benchmark_directGzipHead-4   	   45580	     32286 ns/op	       168 B:rcvd/op	       104 B:sent/op	     30963 rps	    3093 B/op	      33 allocs/op.
func Benchmark_directGzipHead(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodHead)
		req.Header.Set("Accept-Encoding", "gzip")
		req.SetRequestURI(srv.URL + "/gzip-pass-through")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}

// Direct gzip disabled, payload is marshaled and compressed for every request.
// Benchmark_noDirectGzip-4   	    8031	    136836 ns/op	      1029 B:rcvd/op	       117 B:sent/op	      7308 rps	    5382 B/op	      41 allocs/op.
// Benchmark_noDirectGzip-4   	    7587	    143294 ns/op	      1029 B:rcvd/op	       117 B:sent/op	      6974 rps	    4619 B/op	      38 allocs/op.
// Benchmark_noDirectGzip-4   	    7825	    157317 ns/op	      1029 B:rcvd/op	       117 B:sent/op	      6357 rps	    4655 B/op	      40 allocs/op.
func Benchmark_noDirectGzip(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.Set("Accept-Encoding", "gzip")
		req.SetRequestURI(srv.URL + "/gzip-pass-through?plainStruct=1")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}

// Direct gzip enabled, payload is unmarshaled and decompressed for every request in usecase body.
// Unmarshaling large JSON payloads can be much more expensive than explicitly creating them from Go values.
// Benchmark_directGzip_decode-4   	    2018	    499755 ns/op	       624 B:rcvd/op	       116 B:sent/op	      2001 rps	  403967 B/op	     496 allocs/op.
// Benchmark_directGzip_decode-4   	    2085	    526586 ns/op	       624 B:rcvd/op	       116 B:sent/op	      1899 rps	  403600 B/op	     493 allocs/op.
func Benchmark_directGzip_decode(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.Set("Accept-Encoding", "gzip")
		req.SetRequestURI(srv.URL + "/gzip-pass-through?countItems=1")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}

// Direct gzip disabled.
// Benchmark_noDirectGzip_decode-4   	    7603	    142173 ns/op	      1029 B:rcvd/op	       130 B:sent/op	      7034 rps	    5122 B/op	      43 allocs/op.
// Benchmark_noDirectGzip_decode-4   	    5836	    198000 ns/op	      1029 B:rcvd/op	       130 B:sent/op	      5051 rps	    5371 B/op	      42 allocs/op.
func Benchmark_noDirectGzip_decode(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.Set("Accept-Encoding", "gzip")
		req.SetRequestURI(srv.URL + "/gzip-pass-through?plainStruct=1&countItems=1")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}
