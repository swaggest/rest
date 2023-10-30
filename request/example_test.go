package request_test

import (
	"fmt"
	"net/http"

	"github.com/swaggest/rest/request"
)

func ExampleDecoder_Decode() {
	type MyRequest struct {
		Foo int    `header:"X-Foo"`
		Bar string `formData:"bar"`
		Baz bool   `query:"baz"`
	}

	// A decoder for particular structure, can be reused for multiple HTTP requests.
	myDecoder := request.NewDecoderFactory().MakeDecoder(http.MethodPost, new(MyRequest), nil)

	// Request and response writer from ServeHTTP.
	var (
		rw  http.ResponseWriter
		req *http.Request
	)

	// This code would presumably live in ServeHTTP.
	var myReq MyRequest

	if err := myDecoder.Decode(req, &myReq, nil); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}
}

func ExampleEmbeddedSetter_Request() {
	type MyRequest struct {
		request.EmbeddedSetter

		Foo int    `header:"X-Foo"`
		Bar string `formData:"bar"`
		Baz bool   `query:"baz"`
	}

	// A decoder for particular structure, can be reused for multiple HTTP requests.
	myDecoder := request.NewDecoderFactory().MakeDecoder(http.MethodPost, new(MyRequest), nil)

	// Request and response writer from ServeHTTP.
	var (
		rw  http.ResponseWriter
		req *http.Request
	)

	// This code would presumably live in ServeHTTP.
	var myReq MyRequest

	if err := myDecoder.Decode(req, &myReq, nil); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}

	// Access data from raw request.
	fmt.Println("Remote Addr:", myReq.Request().RemoteAddr)
}
