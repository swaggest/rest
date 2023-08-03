package gorillamux

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

// PathToURLValues is a decoder function for parameters in path.
func PathToURLValues(r *http.Request) (url.Values, error) { //nolint:unparam // Matches request.DecoderFactory.SetDecoderFunc.
	muxVars := mux.Vars(r)
	res := make(url.Values, len(muxVars))

	for k, v := range muxVars {
		res.Set(k, v)
	}

	return res, nil
}
