package gzip_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/rest/gzip"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/response"
	gzip2 "github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/usecase"
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
	u.Interactor = usecase.Interact(func(_ context.Context, _, output interface{}) error {
		*output.(*interface{}) = ur

		return nil
	})

	h := nethttp.NewHandler(u)
	h.SetResponseEncoder(&response.Encoder{})

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	r.Header.Set("Accept-Encoding", "deflate, gzip")
	gzip2.Middleware(h).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "1ofolk6sr5j4r", w.Header().Get("Etag"))
	assert.Equal(t, cont.GzipCompressedJSON(), w.Body.Bytes())

	w = httptest.NewRecorder()

	r.Header.Del("Accept-Encoding")
	gzip2.Middleware(h).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "1ofolk6sr5j4r", w.Header().Get("Etag"))
	assert.Equal(t, append(vj, '\n'), w.Body.Bytes())

	w = httptest.NewRecorder()
	ur = v

	r.Header.Set("Accept-Encoding", "deflate, gzip")
	gzip2.Middleware(h).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "", w.Header().Get("Etag"))
}
