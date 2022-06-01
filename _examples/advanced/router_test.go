package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/valyala/fasthttp"
)

func TestNewRouter(t *testing.T) {
	r := NewRouter()

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/docs/openapi.json")

	r.ServeHTTP(rc, rc)
	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())

	actualSchema, err := assertjson.MarshalIndentCompact(json.RawMessage(rc.Response.Body()), "", "  ", 120)
	require.NoError(t, err)

	expectedSchema, err := ioutil.ReadFile("_testdata/openapi.json")
	require.NoError(t, err)

	if !assertjson.Equal(t, expectedSchema, rc.Response.Body(), string(actualSchema)) {
		require.NoError(t, ioutil.WriteFile("_testdata/openapi_last_run.json", actualSchema, 0o600))
	}
}
