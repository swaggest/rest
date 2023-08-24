package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/assertjson"
)

func Test_dynamicOutput(t *testing.T) {
	r := NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/dynamic-schema?bar=ccc&type=ok", nil)
	req.Header.Set("foo", "456")

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	assertjson.Equal(t, []byte(`{"bar": "ccc","status": "ok"}`), rw.Body.Bytes())
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "456", rw.Header().Get("foo"))
}
