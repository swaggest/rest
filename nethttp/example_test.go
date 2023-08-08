package nethttp_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/swaggest/assertjson"
	oapi "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
)

func ExampleSecurityMiddleware() {
	// Create router.
	s := web.NewService(openapi3.NewReflector())

	// Configure an actual security middleware.
	serviceTokenAuth := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") != "<secret>" {
				http.Error(w, "Authentication failed.", http.StatusUnauthorized)

				return
			}

			h.ServeHTTP(w, req)
		})
	}

	// Configure documentation middleware to describe actual security middleware.
	serviceTokenDoc := nethttp.APIKeySecurityMiddleware(s.OpenAPICollector,
		"serviceToken", "Authorization", oapi.InHeader, "Service token.")

	u := usecase.NewIOI(nil, nil, func(ctx context.Context, input, output interface{}) error {
		// Do something.
		return nil
	})

	// Add use case handler to router with security middleware.
	s.
		With(serviceTokenAuth, serviceTokenDoc). // Apply a pair of middlewares: actual security and documentation.
		Method(http.MethodGet, "/foo", nethttp.NewHandler(u))

	schema, _ := assertjson.MarshalIndentCompact(s.OpenAPISchema(), "", " ", 120)
	fmt.Println(string(schema))

	// Output:
	// {
	//  "openapi":"3.0.3","info":{"title":"","version":""},
	//  "paths":{
	//   "/foo":{
	//    "get":{
	//     "summary":"Example Security Middleware","operationId":"rest/nethttp_test.ExampleSecurityMiddleware",
	//     "responses":{
	//      "204":{"description":"No Content"},
	//      "401":{
	//       "description":"Unauthorized",
	//       "content":{"application/json":{"schema":{"$ref":"#/components/schemas/RestErrResponse"}}}
	//      }
	//     },
	//     "security":[{"serviceToken":[]}]
	//    }
	//   }
	//  },
	//  "components":{
	//   "schemas":{
	//    "RestErrResponse":{
	//     "type":"object",
	//     "properties":{
	//      "code":{"type":"integer","description":"Application-specific error code."},
	//      "context":{"type":"object","additionalProperties":{},"description":"Application context."},
	//      "error":{"type":"string","description":"Error message."},"status":{"type":"string","description":"Status text."}
	//     }
	//    }
	//   },
	//   "securitySchemes":{"serviceToken":{"type":"apiKey","name":"Authorization","in":"header","description":"Service token."}}
	//  }
	// }
}
