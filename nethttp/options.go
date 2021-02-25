package nethttp

import (
	"net/http"
	"reflect"

	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/refl"
	"github.com/swaggest/rest"
)

// OptionsMiddleware applies options to encountered nethttp.Handler.
func OptionsMiddleware(options ...func(h *Handler)) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		var rh *Handler

		if HandlerAs(h, &rh) {
			rh.options = append(rh.options, options...)

			for _, option := range options {
				option(rh)
			}
		}

		return h
	}
}

// AnnotateOperation allows customizations of prepared operations.
func AnnotateOperation(annotations ...func(operation *openapi3.Operation) error) func(h *Handler) {
	return func(h *Handler) {
		h.OperationAnnotations = append(h.OperationAnnotations, annotations...)
	}
}

// SuccessfulResponseContentType sets Content-Type of successful response.
func SuccessfulResponseContentType(contentType string) func(h *Handler) {
	return func(h *Handler) {
		h.SuccessContentType = contentType
	}
}

// SuccessStatus sets status code of successful response.
func SuccessStatus(status int) func(h *Handler) {
	return func(h *Handler) {
		h.SuccessStatus = status
	}
}

// RequestMapping creates rest.RequestMapping from struct tags.
//
// This can be used to decouple mapping from usecase input with additional struct.
func RequestMapping(v interface{}) func(h *Handler) {
	return func(h *Handler) {
		m := make(rest.RequestMapping)

		for _, in := range []rest.ParamIn{
			rest.ParamInFormData,
			rest.ParamInQuery,
			rest.ParamInHeader,
			rest.ParamInPath,
			rest.ParamInCookie,
		} {
			mm := make(map[string]string)

			refl.WalkTaggedFields(reflect.ValueOf(v), func(v reflect.Value, sf reflect.StructField, tag string) {
				mm[sf.Name] = tag
			}, string(in))

			if len(mm) > 0 {
				m[in] = mm
			}
		}

		if len(m) > 0 {
			h.ReqMapping = m
		}
	}
}

// ResponseHeaderMapping creates headers mapping from struct tags.
//
// This can be used to decouple mapping from usecase input with additional struct.
func ResponseHeaderMapping(v interface{}) func(h *Handler) {
	return func(h *Handler) {
		if mm, ok := v.(map[string]string); ok {
			h.RespHeaderMapping = mm

			return
		}

		mm := make(map[string]string)

		refl.WalkTaggedFields(reflect.ValueOf(v), func(v reflect.Value, sf reflect.StructField, tag string) {
			mm[sf.Name] = tag
		}, "header")

		if len(mm) > 0 {
			h.RespHeaderMapping = mm
		}
	}
}
