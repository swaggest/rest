package fhttp_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest-fasthttp/fhttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/usecase"
)

func TestOpenAPIMiddleware(t *testing.T) {
	u := &struct {
		usecase.Interactor
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.Input = new(Input)
	u.Output = new(struct {
		Value  string `json:"val"`
		Header int
	})
	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		in, ok := input.(*Input)
		assert.True(t, ok)
		assert.Equal(t, 123, in.ID)

		return nil
	})

	uh := fhttp.NewHandler(u,
		fhttp.SuccessfulResponseContentType("application/vnd.ms-excel"),
		fhttp.RequestMapping(new(struct {
			ID int `query:"ident"`
		})),
		fhttp.ResponseHeaderMapping(new(struct {
			Header int `header:"X-Hd"`
		})),
		fhttp.AnnotateOperation(func(op *openapi3.Operation) error {
			op.WithDescription("Hello!")

			return nil
		}),
	)

	c := openapi.Collector{}

	_ = fhttp.WrapHandler(uh,
		fhttp.OpenAPIMiddleware(&c),
		fhttp.HTTPBasicSecurityMiddleware(&c, "admin", "Admin Area."),
		fhttp.HTTPBearerSecurityMiddleware(&c, "api", "API Security.", "JWT",
			fhttp.SecurityResponse(new(struct {
				Error string `json:"error"`
			}), http.StatusForbidden)),
		fhttp.HandlerWithRouteMiddleware(http.MethodGet, "/test"),
	)

	sp, err := assertjson.MarshalIndentCompact(c.Reflector().Spec, "", " ", 100)
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
	 "openapi":"3.0.3","info":{"title":"","version":""},
	 "paths":{
	  "/test":{
	   "get":{
		"description":"Hello!","parameters":[{"name":"ident","in":"query","schema":{"type":"integer"}}],
		"responses":{
		 "200":{
		  "description":"OK","headers":{"X-Hd":{"style":"simple","schema":{"type":"integer"}}},
		  "content":{
		   "application/vnd.ms-excel":{"schema":{"type":"object","properties":{"val":{"type":"string"}}}}
		  }
		 },
		 "401":{
		  "description":"Unauthorized",
		  "content":{"application/json":{"schema":{"$ref":"#/components/schemas/RestFasthttpErrResponse"}}}
		 },
		 "403":{
		  "description":"Forbidden",
		  "content":{"application/json":{"schema":{"type":"object","properties":{"error":{"type":"string"}}}}}
		 }
		},
		"security":[{"api":[]},{"admin":[]}]
	   }
	  }
	 },
	 "components":{
	  "schemas":{
	   "RestFasthttpErrResponse":{
		"type":"object",
		"properties":{
		 "code":{"type":"integer","description":"Application-specific error code."},
		 "context":{"type":"object","additionalProperties":{},"description":"Application context."},
		 "error":{"type":"string","description":"Error message."},
		 "status":{"type":"string","description":"Status text."}
		}
	   }
	  },
	  "securitySchemes":{
	   "admin":{"type":"http","scheme":"basic","description":"Admin Area."},
	   "api":{"type":"http","scheme":"bearer","bearerFormat":"JWT","description":"API Security."}
	  }
	 }
	}`), sp, string(sp))
}
