package rest

import "encoding/json"

// Validator validates a map of decoded data.
type Validator interface {
	// ValidateData validates decoded request/response data and returns error in case of invalid data.
	ValidateData(in ParamIn, namedData map[string]interface{}) error

	// ValidateJSONBody validates JSON encoded body and returns error in case of invalid data.
	ValidateJSONBody(jsonBody []byte) error

	// HasConstraints indicates if there are validation rules for parameter location.
	HasConstraints(in ParamIn) bool
}

// ValidatorFunc implements Validator with a func.
type ValidatorFunc func(in ParamIn, namedData map[string]interface{}) error

// ValidateData implements Validator.
func (v ValidatorFunc) ValidateData(in ParamIn, namedData map[string]interface{}) error {
	return v(in, namedData)
}

// HasConstraints indicates if there are validation rules for parameter location.
func (v ValidatorFunc) HasConstraints(_ ParamIn) bool {
	return true
}

// ValidateJSONBody implements Validator.
func (v ValidatorFunc) ValidateJSONBody(body []byte) error {
	return v(ParamInBody, map[string]interface{}{"body": json.RawMessage(body)})
}

// RequestJSONSchemaProvider provides request JSON Schemas.
type RequestJSONSchemaProvider interface {
	ProvideRequestJSONSchemas(
		method string,
		input interface{},
		mapping RequestMapping,
		validator JSONSchemaValidator,
	) error
}

// ResponseJSONSchemaProvider provides response JSON Schemas.
type ResponseJSONSchemaProvider interface {
	ProvideResponseJSONSchemas(
		statusCode int,
		contentType string,
		output interface{},
		headerMapping map[string]string,
		validator JSONSchemaValidator,
	) error
}

// JSONSchemaValidator defines JSON schema validator.
type JSONSchemaValidator interface {
	Validator

	// AddSchema accepts JSON schema for a request parameter or response value.
	AddSchema(in ParamIn, name string, schemaData []byte, required bool) error
}

// RequestValidatorFactory creates request validator for particular structured Go input value.
type RequestValidatorFactory interface {
	MakeRequestValidator(method string, input interface{}, mapping RequestMapping) Validator
}

// ResponseValidatorFactory creates response validator for particular structured Go output value.
type ResponseValidatorFactory interface {
	MakeResponseValidator(
		statusCode int,
		contentType string,
		output interface{},
		headerMapping map[string]string,
	) Validator
}

// ValidationErrors is a list of validation errors.
//
// Key is field position (e.g. "path:id" or "body"), value is a list of issues with the field.
type ValidationErrors map[string][]string

// Error returns error message.
func (re ValidationErrors) Error() string {
	return "validation failed"
}

// Fields returns request errors by field location and name.
func (re ValidationErrors) Fields() map[string]interface{} {
	res := make(map[string]interface{}, len(re))

	for k, v := range re {
		res[k] = v
	}

	return res
}
