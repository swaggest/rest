package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
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

	// MakeErrResp overrides error response builder instead of default Err,
	// returned values are HTTP status code and error structure to be marshaled.
	MakeErrResp func(ctx context.Context, err error) (int, interface{})

	// ReqMapping controls request decoding into use case input.
	// Optional, if not set field tags are used as mapping.
	ReqMapping RequestMapping

	RespHeaderMapping map[string]string
	RespCookieMapping map[string]http.Cookie

	// ReqValidator validates decoded request data.
	ReqValidator Validator

	// RespValidator validates decoded response data.
	RespValidator Validator

	// OperationAnnotations are called after operation setup and before adding operation to documentation.
	//
	// Deprecated: use OpenAPIAnnotations.
	OperationAnnotations []func(op *openapi3.Operation) error

	// OpenAPIAnnotations are called after operation setup and before adding operation to documentation.
	OpenAPIAnnotations []func(oc openapi.OperationContext) error
}

// RestHandler is an accessor.
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

	kind := rv.Kind()
	elemKind := reflect.Invalid

	if kind == reflect.Ptr {
		elemKind = rv.Elem().Kind()
	}

	hasTaggedFields := refl.HasTaggedFields(output, "json")
	isSliceOrMap := refl.IsSliceOrMap(output)
	hasEmbeddedSliceOrMap := refl.FindEmbeddedSliceOrMap(output) != nil
	isJSONMarshaler := refl.As(output, new(json.Marshaler))
	isPtrToInterface := elemKind == reflect.Interface
	isScalar := refl.IsScalar(output)

	if withWriter ||
		noContent ||
		hasTaggedFields ||
		isSliceOrMap ||
		hasEmbeddedSliceOrMap ||
		isJSONMarshaler ||
		isPtrToInterface ||
		isScalar {
		return false
	}

	return true
}
