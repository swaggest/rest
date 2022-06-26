package gzip_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest-fasthttp/gzip"
	"github.com/swaggest/rest-fasthttp/response"
	gzip2 "github.com/swaggest/rest-fasthttp/response/gzip"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

func TestWriteJSON(t *testing.T) {
	v := make([]string, 0, 100)

	for i := 0; i < 100; i++ {
		v = append(v, "Quis autem vel eum iure reprehenderit, qui in ea voluptate velit esse, "+
			"quam nihil molestiae consequatur, vel illum, qui dolorem eum fugiat, quo voluptas nulla pariatur?")
	}

	cont := gzip.JSONContainer{}
	require.NoError(t, cont.PackJSON(v))

	var vv []string

	require.NoError(t, cont.UnpackJSON(&vv))
	assert.Equal(t, v, vv)

	cj, err := json.Marshal(cont)
	require.NoError(t, err)

	vj, err := json.Marshal(v)
	require.NoError(t, err)

	assertjson.Equal(t, cj, vj)

	u := struct {
		usecase.Interactor
		usecase.WithOutput
	}{}

	var ur interface{} = cont

	u.Output = new(interface{})
	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		*output.(*interface{}) = ur

		return nil
	})

	h := fhttp.NewHandler(u)
	h.SetResponseEncoder(&response.Encoder{})

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	rc.Request.Header.Set("Accept-Encoding", "deflate, gzip")
	gzip2.Middleware(h).ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "gzip", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Equal(t, "1ofolk6sr5j4r", string(rc.Response.Header.Peek("Etag")))
	assert.Equal(t, cont.GzipCompressedJSON(), rc.Response.Body())

	rc.Request.Header.Del("Accept-Encoding")
	rc.Response = fasthttp.Response{}
	gzip2.Middleware(h).ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Equal(t, "1ofolk6sr5j4r", string(rc.Response.Header.Peek("Etag")))
	assert.Equal(t, append(vj, '\n'), rc.Response.Body())

	ur = v

	rc.Request.Header.Set("Accept-Encoding", "deflate, gzip")
	rc.Response = fasthttp.Response{}

	gzip2.Middleware(h).ServeHTTP(rc, rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "gzip", string(rc.Response.Header.Peek("Content-Encoding")))
	assert.Equal(t, "", string(rc.Response.Header.Peek("Etag")))
}
