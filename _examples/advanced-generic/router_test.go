//go:build go1.18

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
)

func TestNewRouter(t *testing.T) {
	r := NewRouter()

	req, err := http.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	require.NoError(t, err)

	rw := httptest.NewRecorder()

	r.ServeHTTP(rw, req)
	assert.Equal(t, http.StatusOK, rw.Code)

	actualSchema, err := assertjson.MarshalIndentCompact(json.RawMessage(rw.Body.Bytes()), "", "  ", 120)
	require.NoError(t, err)

	expectedSchema, err := os.ReadFile("_testdata/openapi.json")
	require.NoError(t, err)

	if !assertjson.Equal(t, expectedSchema, rw.Body.Bytes(), string(actualSchema)) {
		require.NoError(t, os.WriteFile("_testdata/openapi_last_run.json", actualSchema, 0o600))
	}
}
