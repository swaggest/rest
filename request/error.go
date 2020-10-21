package request

import "errors"

// These errors may be returned on request decoding failure.
var (
	ErrJSONExpected        = errors.New("request with application/json content type expected")
	ErrMissingRequestBody  = errors.New("missing request body")
	ErrMissingRequiredFile = errors.New("missing required file")
)
