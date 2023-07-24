// Package openapi provides documentation collector.
package openapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/usecase"
)

type ContentType string

// Collector extracts OpenAPI documentation from HTTP handler and underlying use case interactor.
type Collector struct {
	mu sync.Mutex

	BasePath string // URL path to docs, default "/docs/".

	// CombineErrors can take a value of "oneOf" or "anyOf",
	// if not empty it enables logical schema grouping in case
	// of multiple responses with same HTTP status code.
	CombineErrors string

	// DefaultSuccessResponseContentType is a default success response content type.
	// If empty, "application/json" is used.
	DefaultSuccessResponseContentType string

	// DefaultErrorResponseContentType is a default error response content type.
	// If empty, "application/json" is used.
	DefaultErrorResponseContentType string

	gen *openapi3.Reflector

	ocAnnotations map[string][]func(oc openapi.OperationContext) error
	annotations   map[string][]func(*openapi3.Operation) error
	operationIDs  map[string]bool
}

// Reflector is an accessor to OpenAPI Reflector instance.
func (c *Collector) Reflector() *openapi3.Reflector {
	if c.gen == nil {
		c.gen = &openapi3.Reflector{}
	}

	return c.gen
}

// Annotate adds OpenAPI operation configuration that is applied during collection.
func (c *Collector) Annotate(method, pattern string, setup ...func(op *openapi3.Operation) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.annotations == nil {
		c.annotations = make(map[string][]func(op *openapi3.Operation) error)
	}

	c.annotations[method+pattern] = append(c.annotations[method+pattern], setup...)
}

// AnnotateOperation adds OpenAPI operation configuration that is applied during collection.
func (c *Collector) AnnotateOperation(method, pattern string, setup ...func(oc openapi.OperationContext) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ocAnnotations == nil {
		c.ocAnnotations = make(map[string][]func(oc openapi.OperationContext) error)
	}

	c.ocAnnotations[method+pattern] = append(c.ocAnnotations[method+pattern], setup...)
}

// CollectOperation prepares and adds OpenAPI operation.
func (c *Collector) CollectOperation(
	method, pattern string,
	annotations ...func(oc openapi.OperationContext) error,
) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to reflect API schema for %s %s: %w", method, pattern, err)
		}
	}()

	reflector := c.Reflector()

	oc, err := reflector.NewOperationContext(method, pattern)
	if err != nil {
		return err
	}

	for _, setup := range append(c.ocAnnotations[method+pattern], annotations...) {
		err = setup(oc)
		if err != nil {
			return err
		}
	}

	return reflector.AddOperation(oc)
}

// Collect adds use case handler to documentation.
func (c *Collector) Collect(
	method, pattern string,
	u usecase.Interactor,
	h rest.HandlerTrait,
	annotations ...func(*openapi3.Operation) error,
) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to reflect API schema for %s %s: %w", method, pattern, err)
		}
	}()

	reflector := c.Reflector()

	err = reflector.SpecEns().SetupOperation(method, pattern, func(op *openapi3.Operation) error {
		oc := openapi3.OperationContext{
			Operation:         op,
			HTTPMethod:        method,
			HTTPStatus:        h.SuccessStatus,
			RespContentType:   h.SuccessContentType,
			RespHeaderMapping: h.RespHeaderMapping,
		}

		err = c.setupInput(&oc, u, h)
		if err != nil {
			return fmt.Errorf("failed to setup request: %w", err)
		}

		err = c.setupOutput(&oc, u)
		if err != nil {
			return fmt.Errorf("failed to setup response: %w", err)
		}

		err = c.processUseCase(op, u, h)
		if err != nil {
			return err
		}

		for _, setup := range c.annotations[method+pattern] {
			err = setup(op)
			if err != nil {
				return err
			}
		}

		for _, setup := range annotations {
			err = setup(op)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (c *Collector) setupOutput(oc *openapi3.OperationContext, u usecase.Interactor) error {
	var (
		hasOutput usecase.HasOutputPort
		status    = http.StatusOK
		noContent bool
	)

	if usecase.As(u, &hasOutput) {
		oc.Output = hasOutput.OutputPort()

		if rest.OutputHasNoContent(oc.Output) {
			status = http.StatusNoContent
			noContent = true
		}
	} else {
		status = http.StatusNoContent
		noContent = true
	}

	if !noContent && oc.RespContentType == "" {
		oc.RespContentType = c.DefaultSuccessResponseContentType
	}

	if outputWithStatus, ok := oc.Output.(rest.OutputWithHTTPStatus); ok {
		for _, status := range outputWithStatus.ExpectedHTTPStatuses() {
			oc.HTTPStatus = status
			if err := c.Reflector().SetupResponse(*oc); err != nil {
				return err
			}
		}
	} else {
		if oc.HTTPStatus == 0 {
			oc.HTTPStatus = status
		}
		err := c.Reflector().SetupResponse(*oc)
		if err != nil {
			return err
		}
	}

	if oc.HTTPMethod == http.MethodHead {
		for code, resp := range oc.Operation.Responses.MapOfResponseOrRefValues {
			for contentType, cont := range resp.Response.Content {
				cont.Schema = nil
				resp.Response.Content[contentType] = cont
			}

			oc.Operation.Responses.MapOfResponseOrRefValues[code] = resp
		}
	}

	return nil
}

func (c *Collector) setupInput(oc *openapi3.OperationContext, u usecase.Interactor, h rest.HandlerTrait) error {
	var (
		hasInput usecase.HasInputPort

		err error
	)

	if usecase.As(u, &hasInput) {
		oc.Input = hasInput.InputPort()

		setRequestMapping(oc, h.ReqMapping)

		err = c.Reflector().SetupRequest(*oc)
		if err != nil {
			return err
		}
	}

	return nil
}

func setRequestMapping(oc *openapi3.OperationContext, mapping rest.RequestMapping) {
	if mapping != nil {
		oc.ReqQueryMapping = mapping[rest.ParamInQuery]
		oc.ReqPathMapping = mapping[rest.ParamInPath]
		oc.ReqHeaderMapping = mapping[rest.ParamInHeader]
		oc.ReqCookieMapping = mapping[rest.ParamInCookie]
		oc.ReqFormDataMapping = mapping[rest.ParamInFormData]
	}
}

func (c *Collector) processUseCase(op *openapi3.Operation, u usecase.Interactor, h rest.HandlerTrait) error {
	var (
		hasName        usecase.HasName
		hasTitle       usecase.HasTitle
		hasDescription usecase.HasDescription
		hasTags        usecase.HasTags
		hasDeprecated  usecase.HasIsDeprecated
	)

	if usecase.As(u, &hasName) {
		id := hasName.Name()

		if id != "" {
			if c.operationIDs == nil {
				c.operationIDs = make(map[string]bool)
			}

			idSuf := id
			suf := 1

			for c.operationIDs[idSuf] {
				suf++
				idSuf = id + strconv.Itoa(suf)
			}

			c.operationIDs[idSuf] = true

			op.WithID(idSuf)
		}
	}

	if usecase.As(u, &hasTitle) {
		title := hasTitle.Title()

		if title != "" {
			op.WithSummary(hasTitle.Title())
		}
	}

	if usecase.As(u, &hasTags) {
		tags := hasTags.Tags()

		if len(tags) > 0 {
			op.WithTags(hasTags.Tags()...)
		}
	}

	if usecase.As(u, &hasDescription) {
		desc := hasDescription.Description()

		if desc != "" {
			op.WithDescription(hasDescription.Description())
		}
	}

	if usecase.As(u, &hasDeprecated) && hasDeprecated.IsDeprecated() {
		op.WithDeprecated(true)
	}

	return c.processExpectedErrors(op, u, h)
}

func (c *Collector) setJSONResponse(op *openapi3.Operation, output interface{}, statusCode int) error {
	oc := openapi3.OperationContext{}
	oc.Operation = op
	oc.Output = output
	oc.HTTPStatus = statusCode

	if output != nil {
		oc.RespContentType = c.DefaultErrorResponseContentType
	}

	return c.Reflector().SetupResponse(oc)
}

func (c *Collector) processExpectedErrors(op *openapi3.Operation, u usecase.Interactor, h rest.HandlerTrait) error {
	var (
		errsByCode        = map[int][]interface{}{}
		statusCodes       []int
		hasExpectedErrors usecase.HasExpectedErrors
	)

	if !usecase.As(u, &hasExpectedErrors) {
		return nil
	}

	for _, e := range hasExpectedErrors.ExpectedErrors() {
		var (
			errResp    interface{}
			statusCode int
		)

		if h.MakeErrResp != nil {
			statusCode, errResp = h.MakeErrResp(context.Background(), e)
		} else {
			statusCode, errResp = rest.Err(e)
		}

		if statusCode < http.StatusOK || statusCode == http.StatusNotModified || statusCode == http.StatusNoContent {
			errResp = nil
		}

		if errsByCode[statusCode] == nil {
			statusCodes = append(statusCodes, statusCode)
		}

		errsByCode[statusCode] = append(errsByCode[statusCode], errResp)

		if err := c.setJSONResponse(op, errResp, statusCode); err != nil {
			return err
		}
	}

	return c.combineErrors(op, statusCodes, errsByCode)
}

func (c *Collector) combineErrors(op *openapi3.Operation, statusCodes []int, errsByCode map[int][]interface{}) error {
	for _, statusCode := range statusCodes {
		var (
			errResps = errsByCode[statusCode]
			err      error
		)

		if len(errResps) == 1 || c.CombineErrors == "" {
			err = c.setJSONResponse(op, errResps[0], statusCode)
		} else {
			switch c.CombineErrors {
			case "oneOf":
				err = c.setJSONResponse(op, jsonschema.OneOf(errResps...), statusCode)
			case "anyOf":
				err = c.setJSONResponse(op, jsonschema.AnyOf(errResps...), statusCode)
			default:
				return errors.New("oneOf/anyOf expected for openapi.Collector.CombineErrors, " +
					c.CombineErrors + " received")
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type unknownFieldsValidator interface {
	ForbidUnknownParams(in rest.ParamIn, forbidden bool)
}

func (c *Collector) provideParametersJSONSchemas(op openapi3.Operation, validator rest.JSONSchemaValidator) error {
	if fv, ok := validator.(unknownFieldsValidator); ok {
		for _, in := range []rest.ParamIn{rest.ParamInQuery, rest.ParamInCookie, rest.ParamInHeader} {
			if op.UnknownParamIsForbidden(openapi3.ParameterIn(in)) {
				fv.ForbidUnknownParams(in, true)
			}
		}
	}

	for _, p := range op.Parameters {
		pp := p.Parameter

		required := false
		if pp.Required != nil && *pp.Required {
			required = true
		}

		sc := paramSchema(pp)

		if sc == nil {
			if validator != nil {
				err := validator.AddSchema(rest.ParamIn(pp.In), pp.Name, nil, required)
				if err != nil {
					return fmt.Errorf("failed to add validation schema for parameter (%s, %s): %w", pp.In, pp.Name, err)
				}
			}

			continue
		}

		schema := sc.ToJSONSchema(c.Reflector().Spec)

		var (
			err        error
			schemaData []byte
		)

		if !schema.IsTrivial(c.Reflector().ResolveJSONSchemaRef) {
			schemaData, err = schema.JSONSchemaBytes()
			if err != nil {
				return fmt.Errorf("failed to build JSON Schema for parameter (%s, %s)", pp.In, pp.Name)
			}
		}

		if validator != nil {
			err = validator.AddSchema(rest.ParamIn(pp.In), pp.Name, schemaData, required)
			if err != nil {
				return fmt.Errorf("failed to add validation schema for parameter (%s, %s): %w", pp.In, pp.Name, err)
			}
		}
	}

	return nil
}

func paramSchema(p *openapi3.Parameter) *openapi3.SchemaOrRef {
	sc := p.Schema

	if sc == nil {
		if jsc, ok := p.Content["application/json"]; ok {
			sc = jsc.Schema
		}
	}

	return sc
}

// ProvideRequestJSONSchemas provides JSON Schemas for request structure.
func (c *Collector) ProvideRequestJSONSchemas(
	method string,
	input interface{},
	mapping rest.RequestMapping,
	validator rest.JSONSchemaValidator,
) error {
	op := openapi3.Operation{}
	oc := openapi3.OperationContext{
		Operation:  &op,
		HTTPMethod: method,
		Input:      input,
	}

	setRequestMapping(&oc, mapping)

	err := c.Reflector().SetupRequest(oc)
	if err != nil {
		return err
	}

	err = c.provideParametersJSONSchemas(op, validator)
	if err != nil {
		return err
	}

	if op.RequestBody == nil || op.RequestBody.RequestBody == nil {
		return nil
	}

	for ct, content := range op.RequestBody.RequestBody.Content {
		schema := content.Schema.ToJSONSchema(c.Reflector().Spec)
		if schema.IsTrivial(c.Reflector().ResolveJSONSchemaRef) {
			continue
		}

		if ct == "application/json" {
			schemaData, err := schema.JSONSchemaBytes()
			if err != nil {
				return fmt.Errorf("failed to build JSON Schema for request body: %w", err)
			}

			err = validator.AddSchema(rest.ParamInBody, "body", schemaData, false)
			if err != nil {
				return fmt.Errorf("failed to add validation schema for request body: %w", err)
			}
		}

		if ct == "application/x-www-form-urlencoded" {
			if err = provideFormDataSchemas(schema, validator); err != nil {
				return err
			}
		}
	}

	return nil
}

func provideFormDataSchemas(schema jsonschema.SchemaOrBool, validator rest.JSONSchemaValidator) error {
	for name, sch := range schema.TypeObject.Properties {
		if sch.TypeObject != nil && len(schema.TypeObject.ExtraProperties) > 0 {
			cp := *sch.TypeObject
			sch.TypeObject = &cp
			sch.TypeObject.ExtraProperties = schema.TypeObject.ExtraProperties
		}

		sb, err := sch.JSONSchemaBytes()
		if err != nil {
			return fmt.Errorf("failed to build JSON Schema for form data parameter %q: %w", name, err)
		}

		isRequired := false

		for _, req := range schema.TypeObject.Required {
			if req == name {
				isRequired = true

				break
			}
		}

		err = validator.AddSchema(rest.ParamInFormData, name, sb, isRequired)
		if err != nil {
			return fmt.Errorf("failed to add validation schema for request body: %w", err)
		}
	}

	return nil
}

func (c *Collector) provideHeaderSchemas(resp *openapi3.Response, validator rest.JSONSchemaValidator) error {
	for name, h := range resp.Headers {
		if h.Header.Schema == nil {
			continue
		}

		hh := h.Header
		schema := hh.Schema.ToJSONSchema(c.Reflector().Spec)

		var (
			err        error
			schemaData []byte
		)

		if !schema.IsTrivial(c.Reflector().ResolveJSONSchemaRef) {
			schemaData, err = schema.JSONSchemaBytes()
			if err != nil {
				return fmt.Errorf("failed to build JSON Schema for response header (%s)", name)
			}
		}

		required := false
		if hh.Required != nil && *hh.Required {
			required = true
		}

		if validator != nil {
			name = http.CanonicalHeaderKey(name)

			err = validator.AddSchema(rest.ParamInHeader, name, schemaData, required)
			if err != nil {
				return fmt.Errorf("failed to add validation schema for response header (%s): %w", name, err)
			}
		}
	}

	return nil
}

// ProvideResponseJSONSchemas provides JSON schemas for response structure.
func (c *Collector) ProvideResponseJSONSchemas(
	statusCode int,
	contentType string,
	output interface{},
	headerMapping map[string]string,
	validator rest.JSONSchemaValidator,
) error {
	op := openapi3.Operation{}
	oc := openapi3.OperationContext{
		Operation:         &op,
		HTTPStatus:        statusCode,
		Output:            output,
		RespHeaderMapping: headerMapping,
		RespContentType:   contentType,
	}

	if oc.RespContentType == "" {
		oc.RespContentType = c.DefaultSuccessResponseContentType
	}

	if err := c.Reflector().SetupResponse(oc); err != nil {
		return err
	}

	resp := op.Responses.MapOfResponseOrRefValues[strconv.Itoa(statusCode)].Response

	if err := c.provideHeaderSchemas(resp, validator); err != nil {
		return err
	}

	for _, cont := range resp.Content {
		if cont.Schema == nil {
			continue
		}

		schema := cont.Schema.ToJSONSchema(c.Reflector().Spec)

		if schema.IsTrivial(c.Reflector().ResolveJSONSchemaRef) {
			continue
		}

		schemaData, err := schema.JSONSchemaBytes()
		if err != nil {
			return errors.New("failed to build JSON Schema for response body")
		}

		if err := validator.AddSchema(rest.ParamInBody, "body", schemaData, false); err != nil {
			return fmt.Errorf("failed to add validation schema for response body: %w", err)
		}
	}

	return nil
}

func (c *Collector) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	document, err := json.MarshalIndent(c.Reflector().Spec, "", " ")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	rw.Header().Set("Content-Type", "application/json")

	_, err = rw.Write(document)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}
