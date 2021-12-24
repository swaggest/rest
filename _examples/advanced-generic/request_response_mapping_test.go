//go:build go1.18
// +build go1.18

package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func Test_requestResponseMapping(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/req-resp-mapping",
		bytes.NewReader([]byte(`val2=3`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Header", "abc")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Empty(t, body)

	assert.Equal(t, "abc", resp.Header.Get("X-Value-1"))
	assert.Equal(t, "3", resp.Header.Get("X-Value-2"))
}

func Benchmark_requestResponseMapping(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.Header.SetMethod(http.MethodPost)
		req.SetRequestURI(srv.URL + "/req-resp-mapping")
		req.Header.Set("X-Header", "abc")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBody([]byte(`val2=3`))
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusNoContent
	})
}
