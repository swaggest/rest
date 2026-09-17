package chirouter_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/request"
)

// Define request structure for your HTTP handler.
type myRequest struct {
	Query1    int     `query:"query1"`
	Path1     string  `path:"path1"`
	Path2     int     `path:"path2"`
	Header1   float64 `header:"X-Header-1"`
	FormData1 bool    `formData:"formData1"`
	FormData2 string  `formData:"formData2"`
}

type myResp struct {
	Sum    float64 `json:"sum"`
	Concat string  `json:"concat"`
}

func newMyHandler() *myHandler {
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.ApplyDefaults = true
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	return &myHandler{
		dec: decoderFactory.MakeDecoder(http.MethodGet, myRequest{}, nil),
	}
}

type myHandler struct {
	// Automated request decoding is not required to collect OpenAPI schema,
	// but it is good to have to establish a single source of truth and to simplify request reading.
	dec nethttp.RequestDecoder
}

func (m *myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var in myRequest

	if err := m.dec.Decode(r, &in, nil); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Serve request.
	out := myResp{
		Sum:    in.Header1 + float64(in.Path2) + float64(in.Query1),
		Concat: in.Path1 + in.FormData2 + strconv.FormatBool(in.FormData1),
	}

	j, err := json.Marshal(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(j)
}

// SetupOpenAPIOperation declares OpenAPI schema for the handler.
func (m *myHandler) SetupOpenAPIOperation(oc openapi.OperationContext) error {
	oc.SetTags("My Tag")
	oc.SetSummary("My Summary")
	oc.SetDescription("This endpoint aggregates request in structured way.")

	oc.AddReqStructure(myRequest{})
	oc.AddRespStructure(myResp{})
	oc.AddRespStructure(nil, openapi.WithContentType("text/html"), openapi.WithHTTPStatus(http.StatusBadRequest))
	oc.AddRespStructure(nil, openapi.WithContentType("text/html"), openapi.WithHTTPStatus(http.StatusInternalServerError))

	return nil
}

func ExampleNewOpenAPICollector() {
	// Your router does not need special instrumentation.
	router := chi.NewRouter()

	// If handler implements chirouter.OpenAPIPreparer, it will contribute detailed information to OpenAPI document.
	router.Method(http.MethodGet, "/foo/{path1}/bar/{path2}", newMyHandler())

	// If handler does not implement chirouter.OpenAPIPreparer, it will be exposed as incomplete.
	router.Post("/uninstrumented-handler/{path-item}", func(w http.ResponseWriter, r *http.Request) {})

	// A plain http.HandlerFunc can't implement chirouter.OpenAPIPreparer (funcs can't have methods),
	// but it can still get full documentation via Collector.AnnotateOperation, keyed by method and pattern.
	router.Get("/func-handler/{path-item}", func(w http.ResponseWriter, r *http.Request) {})

	// Setup OpenAPI schema.
	refl := openapi3.NewReflector()
	refl.SpecSchema().SetTitle("Sample API")
	refl.SpecSchema().SetVersion("v1.2.3")
	refl.SpecSchema().SetDescription("This is an example.")

	// Walk the router with OpenAPI collector.
	c := chirouter.NewOpenAPICollector(refl)

	// AnnotateOperation is the quickest way to document a func handler, but the method+pattern
	// key is a second copy of the route: if the route changes and this string is not updated to
	// match, the annotation silently stops applying (falls back to "Incomplete").
	c.Collector.AnnotateOperation(http.MethodGet, "/func-handler/{path-item}", func(oc openapi.OperationContext) error {
		oc.SetSummary("Func Handler")
		oc.AddReqStructure(struct {
			PathItem string `path:"path-item"`
		}{})
		oc.AddRespStructure(myResp{})

		return nil
	})

	// Collector.Describe avoids that duplication: it keys documentation by the handler's own
	// identity, so the route registration itself stays the single source of truth. Use it for
	// func handlers you register yourself, especially routes built programmatically/in bulk,
	// where keeping a second, string-keyed AnnotateOperation call in sync would be error-prone.
	router.Get("/identified-func/{path-item}", c.Describe(
		func(w http.ResponseWriter, r *http.Request) {},
		func(oc openapi.OperationContext) error {
			oc.SetSummary("Identified Func Handler")
			oc.AddReqStructure(struct {
				PathItem string `path:"path-item"`
			}{})
			oc.AddRespStructure(myResp{})

			return nil
		},
	))

	_ = chi.Walk(router, c.Walker)

	// Get the resulting schema.
	yml, _ := refl.Spec.MarshalYAML()
	fmt.Println(string(yml))

	// Output:
	// openapi: 3.0.3
	// info:
	//   description: This is an example.
	//   title: Sample API
	//   version: v1.2.3
	// paths:
	//   /foo/{path1}/bar/{path2}:
	//     get:
	//       description: This endpoint aggregates request in structured way.
	//       parameters:
	//       - in: query
	//         name: query1
	//         schema:
	//           type: integer
	//       - in: path
	//         name: path1
	//         required: true
	//         schema:
	//           type: string
	//       - in: path
	//         name: path2
	//         required: true
	//         schema:
	//           type: integer
	//       - in: header
	//         name: X-Header-1
	//         schema:
	//           format: double
	//           type: number
	//       responses:
	//         "200":
	//           content:
	//             application/json:
	//               schema:
	//                 $ref: '#/components/schemas/ChirouterTestMyResp'
	//           description: OK
	//         "400":
	//           content:
	//             text/html:
	//               schema:
	//                 type: string
	//           description: Bad Request
	//         "500":
	//           content:
	//             text/html:
	//               schema:
	//                 type: string
	//           description: Internal Server Error
	//       summary: My Summary
	//       tags:
	//       - My Tag
	//   /func-handler/{path-item}:
	//     get:
	//       parameters:
	//       - in: path
	//         name: path-item
	//         required: true
	//         schema:
	//           type: string
	//       responses:
	//         "200":
	//           content:
	//             application/json:
	//               schema:
	//                 $ref: '#/components/schemas/ChirouterTestMyResp'
	//           description: OK
	//       summary: Func Handler
	//   /identified-func/{path-item}:
	//     get:
	//       parameters:
	//       - in: path
	//         name: path-item
	//         required: true
	//         schema:
	//           type: string
	//       responses:
	//         "200":
	//           content:
	//             application/json:
	//               schema:
	//                 $ref: '#/components/schemas/ChirouterTestMyResp'
	//           description: OK
	//       summary: Identified Func Handler
	//   /uninstrumented-handler/{path-item}:
	//     post:
	//       description: Information about this operation was obtained using only HTTP method
	//         and path pattern. It may be incomplete and/or inaccurate.
	//       parameters:
	//       - in: path
	//         name: path-item
	//         required: true
	//         schema:
	//           type: string
	//       responses:
	//         "200":
	//           content:
	//             text/html:
	//               schema:
	//                 type: string
	//           description: OK
	//       tags:
	//       - Incomplete
	// components:
	//   schemas:
	//     ChirouterTestMyResp:
	//       properties:
	//         concat:
	//           type: string
	//         sum:
	//           format: double
	//           type: number
	//       type: object
}
