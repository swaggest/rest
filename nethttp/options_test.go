package nethttp_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
)

func TestRequestBodyContent(t *testing.T) {
	h := &nethttp.Handler{}
	op := openapi3.Operation{}

	nethttp.RequestBodyContent("text/plain")(h)
	require.Len(t, h.OperationAnnotations, 1)
	require.NoError(t, h.OperationAnnotations[0](&op))
	assertjson.EqualMarshal(t, []byte(`{
	  "requestBody":{"content":{"text/plain":{"schema":{"type":"string"}}}},
	  "responses":{}
	}`), op)
}
