package rest

// ParamIn defines parameter location.
type ParamIn string

const (
	// ParamInPath indicates path parameters, such as `/users/{id}`.
	ParamInPath = ParamIn("path")

	// ParamInQuery indicates query parameters, such as `/users?page=10`.
	ParamInQuery = ParamIn("query")

	// ParamInBody indicates body value, such as `{"id": 10}`.
	ParamInBody = ParamIn("body")

	// ParamInFormData indicates body form parameters.
	ParamInFormData = ParamIn("formData")

	// ParamInCookie indicates cookie parameters, which are passed ParamIn the `Cookie` header,
	// such as `Cookie: debug=0; gdpr=2`.
	ParamInCookie = ParamIn("cookie")

	// ParamInHeader indicates header parameters, such as `X-Header: value`.
	ParamInHeader = ParamIn("header")
)

// RequestMapping describes how decoded request should be applied to container struct.
//
// It is defined as a map by parameter location.
// Each item is a map with struct field name as key and decoded field name as value.
//
// Example:
//
//	map[rest.ParamIn]map[string]string{rest.ParamInQuery:map[string]string{"ID": "id", "FirstName": "first-name"}}
type RequestMapping map[ParamIn]map[string]string

// RequestErrors is a list of validation or decoding errors.
//
// Key is field position (e.g. "path:id" or "body"), value is a list of issues with the field.
type RequestErrors map[string][]string

// Error returns error message.
func (re RequestErrors) Error() string {
	return "bad request"
}

// Fields returns request errors by field location and name.
func (re RequestErrors) Fields() map[string]interface{} {
	res := make(map[string]interface{}, len(re))

	for k, v := range re {
		res[k] = v
	}

	return res
}
