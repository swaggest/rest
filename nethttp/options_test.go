package nethttp_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
)

func TestRequestBodyContent(t *testing.T) {
	h := &nethttp.Handler{}

	r := openapi3.NewReflector()
	oc, err := r.NewOperationContext(http.MethodPost, "/")
	require.NoError(t, err)

	nethttp.RequestBodyContent("text/plain")(h)
	require.Len(t, h.OpenAPIAnnotations, 1)
	require.NoError(t, h.OpenAPIAnnotations[0](oc))

	require.NoError(t, r.AddOperation(oc))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3","info":{"title":"","version":""},
	  "paths":{
		"/":{
		  "post":{
			"requestBody":{"content":{"text/plain":{"schema":{"type":"string"}}}},
			"responses":{"204":{"description":"No Content"}}
		  }
		}
	  }
	}`, r.SpecSchema())
}

func TestRequestBodyContent_webService(t *testing.T) {
	s := web.DefaultService()

	u := usecase.NewIOI(new(string), nil, func(ctx context.Context, input, output interface{}) error {
		return nil
	})

	s.Post("/text-req-body", u, nethttp.RequestBodyContent("text/csv"))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3","info":{"title":"","version":""},
	  "paths":{
		"/text-req-body":{
		  "post":{
			"summary":"Test Request Body Content _ web Service",
			"operationId":"rest/nethttp_test.TestRequestBodyContent_webService",
			"requestBody":{"content":{"text/csv":{"schema":{"type":"string"}}}},
			"responses":{"204":{"description":"No Content"}}
		  }
		}
	  }
	}`, s.OpenAPISchema())
}
