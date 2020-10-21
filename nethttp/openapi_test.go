package nethttp_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
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

	uh := nethttp.NewHandler(u,
		nethttp.SuccessfulResponseContentType("application/vnd.ms-excel"),
		nethttp.RequestMapping(new(struct {
			ID int `query:"ident"`
		})),
		nethttp.ResponseHeaderMapping(new(struct {
			Header int `header:"X-Hd"`
		})),
		nethttp.AnnotateOperation(func(op *openapi3.Operation) error {
			op.WithDescription("Hello!")

			return nil
		}),
	)

	c := openapi.Collector{}

	_ = nethttp.WrapHandler(uh,
		nethttp.OpenAPIMiddleware(&c),
		nethttp.HTTPBasicSecurityMiddleware(&c, "admin", "Admin Area."),
		nethttp.HTTPBearerSecurityMiddleware(&c, "api", "API Security.", "JWT"),
		nethttp.HandlerWithRouteMiddleware(http.MethodGet, "/test"),
	)

	sp, err := assertjson.MarshalIndentCompact(c.Reflector().Spec, "", " ", 100)
	require.NoError(t, err)

	assertjson.Equal(t, []byte(`{
	 "openapi":"3.0.2","info":{"title":"","version":""},
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
		  "content":{"application/json":{"schema":{"$ref":"#/components/schemas/RestErrResponse"}}}
		 }
		},
		"security":[{"api":[]},{"admin":[]}]
	   }
	  }
	 },
	 "components":{
	  "schemas":{
	   "RestErrResponse":{
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
