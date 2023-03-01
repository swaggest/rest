//go:build go1.18

package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/valyala/fasthttp"
)

// Benchmark_outputHeaders-4   	   41424	     27054 ns/op	       154 B:rcvd/op	        77.0 B:sent/op	     36963 rps	    3641 B/op	      35 allocs/op.
func Benchmark_outputHeaders(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodGet)
		req.SetRequestURI(srv.URL + "/output-headers")
		req.Header.Set("X-Foo", "40")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}

func Test_outputHeaders(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/output-headers", nil)
	require.NoError(t, err)

	req.Header.Set("x-FoO", "40")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	assert.Equal(t, "abc", resp.Header.Get("X-Header"))
	assert.Equal(t, "20", resp.Header.Get("X-Foo"))
	assert.Equal(t, "10", resp.Header.Get("X-Omit-Empty"))
	assert.Equal(t, []string{"coo=123; HttpOnly"}, resp.Header.Values("Set-Cookie"))
	assertjson.Equal(t, []byte(`{"inBody":"def"}`), body)
}

func Test_outputHeaders_invalidReq(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/output-headers", nil)
	require.NoError(t, err)

	req.Header.Set("x-FoO", "5")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	assertjson.Equal(t,
		[]byte(`{"msg":"invalid argument: validation failed","details":{"header:X-Foo":["#: must be >= 10/1 but found 5"]}}`),
		body, string(body))
}

func Test_outputHeaders_invalidResp(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/output-headers", nil)
	require.NoError(t, err)

	req.Header.Set("x-FoO", "15")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, resp.StatusCode, http.StatusInternalServerError)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	assertjson.Equal(t,
		[]byte(`{"msg":"internal: bad response: validation failed","details":{"header:X-Foo":["#: must be >= 10/1 but found -5"]}}`),
		body, string(body))
}

func Test_outputHeaders_omitempty(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/output-headers", nil)
	require.NoError(t, err)

	req.Header.Set("x-FoO", "30")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	assert.Equal(t, "abc", resp.Header.Get("X-Header"))
	assert.Equal(t, "10", resp.Header.Get("X-Foo"))
	assert.Equal(t, []string(nil), resp.Header.Values("X-Omit-Empty"))
	assert.Equal(t, []string{"coo=123; HttpOnly"}, resp.Header.Values("Set-Cookie"))
	assertjson.Equal(t, []byte(`{"inBody":"def"}`), body)
}
