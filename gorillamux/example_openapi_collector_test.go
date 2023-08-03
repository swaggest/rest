package gorillamux_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/gorillamux"
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
	decoderFactory.SetDecoderFunc(rest.ParamInPath, gorillamux.PathToURLValues)

	return &myHandler{
		dec: decoderFactory.MakeDecoder(http.MethodPost, myRequest{}, nil),
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
	router := mux.NewRouter()

	// If handler implements gorillamux.OpenAPIPreparer, it will contribute detailed information to OpenAPI document.
	router.Handle("/foo/{path1}/bar/{path2}", newMyHandler()).Methods(http.MethodGet)

	// If handler does not implement gorillamux.OpenAPIPreparer, it will be exposed as incomplete.
	router.Handle("/uninstrumented-handler/{path-item}",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).Methods(http.MethodPost)

	// Setup OpenAPI schema.
	refl := openapi3.NewReflector()
	refl.SpecSchema().SetTitle("Sample API")
	refl.SpecSchema().SetVersion("v1.2.3")
	refl.SpecSchema().SetDescription("This is an example.")

	// Walk the router with OpenAPI collector.
	c := gorillamux.NewOpenAPICollector(refl)

	_ = router.Walk(c.Walker)

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
	//           type: number
	//       responses:
	//         "200":
	//           content:
	//             application/json:
	//               schema:
	//                 $ref: '#/components/schemas/GorillamuxTestMyResp'
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
	//   /uninstrumented-handler/{path-item}:
	//     post:
	//       description: Information about this operation was obtained using only HTTP method
	//         and path pattern. It may be incomplete and/or inaccurate.
	//       parameters:
	//       - in: path
	//         name: path-item
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
	//     GorillamuxTestMyResp:
	//       properties:
	//         concat:
	//           type: string
	//         sum:
	//           type: number
	//       type: object
}
