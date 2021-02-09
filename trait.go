package rest

import (
	"context"
	"reflect"

	json "github.com/goccy/go-json"
	"github.com/swaggest/refl"
	"github.com/swaggest/usecase"
)

// HandlerTrait controls basic behavior of rest handler.
type HandlerTrait struct {
	// SuccessStatus is an HTTP status code to set on successful use case interaction.
	//
	// Default is 200 (OK) or 204 (No Content).
	SuccessStatus int

	// SuccessContentType is a Content-Type of successful response, default application/json.
	SuccessContentType string

	// MakeErrResp overrides error response builder instead of default Err.
	MakeErrResp func(ctx context.Context, err error) (int, interface{})

	// ReqMapping controls request decoding into use case input.
	// Optional, if not set field tags are used as mapping.
	ReqMapping RequestMapping

	RespHeaderMapping map[string]string

	// ReqValidator validates decoded request data.
	ReqValidator Validator

	// RespValidator validates decoded response data.
	RespValidator Validator
}

// RestHandler is a an accessor.
func (h *HandlerTrait) RestHandler() *HandlerTrait {
	return h
}

// RequestMapping returns custom mapping for request decoder.
func (h *HandlerTrait) RequestMapping() RequestMapping {
	return h.ReqMapping
}

// OutputHasNoContent indicates if output does not seem to have any content body to render in response.
func OutputHasNoContent(output interface{}) bool {
	if output == nil {
		return true
	}

	_, withWriter := output.(usecase.OutputWithWriter)
	_, noContent := output.(usecase.OutputWithNoContent)

	rv := reflect.ValueOf(output)

	if !withWriter && !noContent &&
		!refl.HasTaggedFields(output, "json") &&
		!refl.IsSliceOrMap(output) &&
		refl.FindEmbeddedSliceOrMap(output) == nil &&
		!refl.As(output, new(json.Marshaler)) &&
		(rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Interface) {
		return true
	}

	return false
}
