// Package jsonschema implements request validator with github.com/santhosh-tekuri/jsonschema/v2.
package jsonschema

import (
	"bytes"
	"encoding/json"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/swaggest/rest"
)

var _ rest.Validator = &Validator{}

// Validator is a JSON Schema based validator.
type Validator struct {
	// JSONMarshal controls custom marshaler, nil value enables "encoding/json".
	JSONMarshal func(interface{}) ([]byte, error)

	inNamedSchemas map[rest.ParamIn]map[string]*jsonschema.Schema
	inRequired     map[rest.ParamIn][]string
}

// NewFactory creates new validator factory.
func NewFactory(
	requestSchemas rest.RequestJSONSchemaProvider,
	responseSchemas rest.ResponseJSONSchemaProvider,
) Factory {
	return Factory{
		requestSchemas:  requestSchemas,
		responseSchemas: responseSchemas,
	}
}

// Factory makes JSON Schema request validators.
//
// Please use NewFactory to create an instance.
type Factory struct {
	// JSONMarshal controls custom marshaler, nil value enables "encoding/json".
	JSONMarshal func(interface{}) ([]byte, error)

	requestSchemas  rest.RequestJSONSchemaProvider
	responseSchemas rest.ResponseJSONSchemaProvider
}

// MakeRequestValidator creates request validator for HTTP method and input structure.
func (f Factory) MakeRequestValidator(
	method string,
	input interface{},
	mapping rest.RequestMapping,
) rest.Validator {
	v := Validator{
		JSONMarshal: f.JSONMarshal,
	}

	err := f.requestSchemas.ProvideRequestJSONSchemas(method, input, mapping, &v)
	if err != nil {
		panic(err)
	}

	return &v
}

// MakeResponseValidator creates response validator.
//
// Header mapping is a map of struct field name to header name.
func (f Factory) MakeResponseValidator(
	statusCode int,
	contentType string,
	output interface{},
	headerMapping map[string]string,
) rest.Validator {
	v := Validator{
		JSONMarshal: f.JSONMarshal,
	}

	err := f.responseSchemas.ProvideResponseJSONSchemas(statusCode, contentType, output, headerMapping, &v)
	if err != nil {
		panic(err)
	}

	if len(v.inNamedSchemas) == 0 {
		return nil
	}

	return &v
}

const ephemeralSchemaID = "https://rest/schema.json"

// AddSchema registers schema for validation.
func (v *Validator) AddSchema(in rest.ParamIn, name string, jsonSchema []byte, required bool) error {
	if v.JSONMarshal == nil {
		v.JSONMarshal = json.Marshal
	}

	if v.inNamedSchemas == nil {
		v.inNamedSchemas = make(map[rest.ParamIn]map[string]*jsonschema.Schema)
		v.inRequired = make(map[rest.ParamIn][]string)
	}

	if _, ok := v.inNamedSchemas[in]; !ok {
		v.inNamedSchemas[in] = make(map[string]*jsonschema.Schema)
		v.inRequired[in] = make([]string, 0)
	}

	if required {
		v.inRequired[in] = append(v.inRequired[in], name)
	}

	if len(jsonSchema) == 0 {
		return nil
	}

	compiler := jsonschema.NewCompiler()

	err := compiler.AddResource(ephemeralSchemaID, bytes.NewBuffer(jsonSchema))
	if err != nil {
		return err
	}

	schema, err := compiler.Compile(ephemeralSchemaID)
	if err != nil {
		return err
	}

	v.inNamedSchemas[in][name] = schema

	return nil
}

func (v *Validator) checkRequired(in rest.ParamIn, namedData map[string]interface{}) []string {
	required := v.inRequired[in]

	if len(required) == 0 {
		return nil
	}

	var missing []string

	for _, name := range v.inRequired[in] {
		if _, ok := namedData[name]; !ok {
			missing = append(missing, name)
		}
	}

	return missing
}

// ValidateJSONBody performs validation of JSON body.
func (v *Validator) ValidateJSONBody(jsonBody []byte) error {
	name := "body"

	schema, found := v.inNamedSchemas[rest.ParamInBody][name]
	if !found {
		return nil
	}

	var val interface{}
	if err := json.Unmarshal(jsonBody, &val); err != nil {
		return err
	}

	err := schema.Validate(val)
	if err == nil {
		return nil
	}

	errs := make(rest.ValidationErrors, 1)

	// nolint:errorlint // Error is not wrapped, type assertion is more performant.
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		errs[name] = appendError(errs[name], ve)
	} else {
		errs[name] = append(errs[name], err.Error())
	}

	return errs
}

// HasConstraints indicates if there are validation rules for parameter location.
func (v *Validator) HasConstraints(in rest.ParamIn) bool {
	return len(v.inNamedSchemas[in]) > 0
}

// ValidateData performs validation of a mapped request data.
func (v *Validator) ValidateData(in rest.ParamIn, namedData map[string]interface{}) error {
	var errs rest.ValidationErrors

	missing := v.checkRequired(in, namedData)
	if len(missing) != 0 {
		errs = make(rest.ValidationErrors, len(missing))

		for _, name := range missing {
			errs[string(in)+":"+name] = []string{"missing value"}
		}
	}

	for name, value := range namedData {
		schema, found := v.inNamedSchemas[in][name]
		if !found {
			continue
		}

		err := schema.Validate(value)
		if err == nil {
			continue
		}

		if _, ok := err.(jsonschema.InvalidJSONTypeError); ok { // nolint:errorlint // Error is not wrapped, type assertion is more performant.
			b, e := json.Marshal(value)
			if e != nil {
				return e
			}

			var val interface{}
			if e := json.Unmarshal(b, &val); e != nil {
				return e
			}

			err = schema.Validate(val)
			if err == nil {
				continue
			}
		}

		if errs == nil {
			errs = make(rest.ValidationErrors, 1)
		}

		errKey := string(in) + ":" + name

		// nolint:errorlint // Error is not wrapped, type assertion is more performant.
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			errs[errKey] = appendError(errs[errKey], ve)
		} else {
			errs[errKey] = append(errs[errKey], err.Error())
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func appendError(errorMessages []string, err *jsonschema.ValidationError) []string {
	if err.Message != "doesn't validate with https://rest/schema.json#" || len(err.Causes) == 0 {
		errorMessages = append(errorMessages, "#"+err.InstanceLocation+": "+err.Message)
	}

	for _, ec := range err.Causes {
		errorMessages = appendError(errorMessages, ec)
	}

	return errorMessages
}
