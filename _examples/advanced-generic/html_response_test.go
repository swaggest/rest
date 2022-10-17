package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func Test_htmlResponse(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/html-response/123?filter=feel")
	require.NoError(t, err)

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	assert.Equal(t, "true", resp.Header.Get("X-Anti-Header"))
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))
	assert.Equal(t, `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>Foo</title>
	</head>
	<body>
		<a href="/html-response/124?filter=feel">Next Foo</a><br />
		<div>foo</div><div>bar</div><div>baz</div>
	</body>
</html>`, string(body), string(body))
}

// Benchmark_htmlResponse-12    	   89209	     12348 ns/op	         0.3801 50%:ms	         1.119 90%:ms	         2.553 99%:ms	         3.877 99.9%:ms	       370.0 B:rcvd/op	       108.0 B:sent/op	     80973 rps	    8279 B/op	     144 allocs/op.
func Benchmark_htmlResponse(b *testing.B) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	httptestbench.RoundTrip(b, 50, func(i int, req *fasthttp.Request) {
		req.SetRequestURI(srv.URL + "/html-response/123?filter=feel")
		req.Header.Set("X-Header", "true")
	}, func(i int, resp *fasthttp.Response) bool {
		return resp.StatusCode() == http.StatusOK
	})
}
