package nethttp

import (
	"net/http"
	"reflect"

	"github.com/swaggest/openapi-go"
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

// AnnotateOpenAPIOperation allows customization of OpenAPI operation, that is reflected from the Handler.
func AnnotateOpenAPIOperation(annotations ...func(oc openapi.OperationContext) error) func(h *Handler) {
	return func(h *Handler) {
		h.OpenAPIAnnotations = append(h.OpenAPIAnnotations, annotations...)
	}
}

// AnnotateOperation allows customizations of prepared operations.
//
// Deprecated: use AnnotateOpenAPIOperation.
func AnnotateOperation(annotations ...func(operation *openapi3.Operation) error) func(h *Handler) {
	return func(h *Handler) {
		for _, a := range annotations {
			a := a

			h.OpenAPIAnnotations = append(h.OpenAPIAnnotations, func(oc openapi.OperationContext) error {
				if o3, ok := oc.(openapi3.OperationExposer); ok {
					return a(o3.Operation())
				}

				return nil
			})
		}
	}
}

// RequestBodyContent enables string request body with content type (e.g. text/plain).
func RequestBodyContent(contentType string) func(h *Handler) {
	return func(h *Handler) {
		h.OpenAPIAnnotations = append(h.OpenAPIAnnotations, func(oc openapi.OperationContext) error {
			oc.AddReqStructure(nil, func(cu *openapi.ContentUnit) {
				cu.ContentType = contentType
			})

			return nil
		})
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

			refl.WalkTaggedFields(reflect.ValueOf(v), func(_ reflect.Value, sf reflect.StructField, tag string) {
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

		refl.WalkTaggedFields(reflect.ValueOf(v), func(_ reflect.Value, sf reflect.StructField, tag string) {
			mm[sf.Name] = tag
		}, "header")

		if len(mm) > 0 {
			h.RespHeaderMapping = mm
		}
	}
}
