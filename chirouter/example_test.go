package chirouter_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/request"
)

func ExamplePathToURLValues() {
	// Instantiate decoder factory with gorillamux.PathToURLValues.
	// Single factory can be used to create multiple request decoders.
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.ApplyDefaults = true
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	// Define request structure for your HTTP handler.
	type myRequest struct {
		Query1    int     `query:"query1"`
		Path1     string  `path:"path1"`
		Path2     int     `path:"path2"`
		Header1   float64 `header:"X-Header-1"`
		FormData1 bool    `formData:"formData1"`
		FormData2 string  `formData:"formData2"`
	}

	// Create decoder for that request structure.
	dec := decoderFactory.MakeDecoder(http.MethodPost, myRequest{}, nil)

	router := chi.NewRouter()

	// Now in router handler you can decode *http.Request into a Go structure.
	router.Handle("/foo/{path1}/bar/{path2}", http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		var in myRequest

		_ = dec.Decode(r, &in, nil)

		fmt.Printf("%+v\n", in)
	}))

	// Serving example URL.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/foo/abc/bar/123?query1=321",
		bytes.NewBufferString("formData1=true&formData2=def"))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Header-1", "1.23")

	router.ServeHTTP(w, req)
	// Output:
	// {Query1:321 Path1:abc Path2:123 Header1:1.23 FormData1:true FormData2:def}
}
