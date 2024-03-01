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
	ref openapi.Reflector

	ocAnnotations map[string][]func(oc openapi.OperationContext) error
	annotations   map[string][]func(*openapi3.Operation) error
	operationIDs  map[string]bool
}

// NewCollector creates an instance of OpenAPI Collector.
func NewCollector(r openapi.Reflector) *Collector {
	c := &Collector{
		ref: r,
	}

	if r3, ok := r.(*openapi3.Reflector); ok {
		c.gen = r3
	}

	return c
}

// SpecSchema returns OpenAPI specification schema.
func (c *Collector) SpecSchema() openapi.SpecSchema {
	return c.Refl().SpecSchema()
}

// Refl returns OpenAPI reflector.
func (c *Collector) Refl() openapi.Reflector {
	if c.ref != nil {
		return c.ref
	}

	return c.Reflector()
}

// Reflector is an accessor to OpenAPI Reflector instance.
func (c *Collector) Reflector() *openapi3.Reflector {
	if c.ref != nil && c.gen == nil {
		panic(fmt.Sprintf("conflicting OpenAPI reflector supplied: %T", c.ref))
	}

	if c.gen == nil {
		c.gen = openapi3.NewReflector()
	}

	return c.gen
}

// Annotate adds OpenAPI operation configuration that is applied during collection.
//
// Deprecated: use AnnotateOperation.
func (c *Collector) Annotate(method, pattern string, setup ...func(op *openapi3.Operation) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.annotations == nil {
		c.annotations = make(map[string][]func(op *openapi3.Operation) error)
	}

	c.annotations[method+pattern] = append(c.annotations[method+pattern], setup...)
}

// AnnotateOperation adds OpenAPI operation configuration that is applied during collection,
// method can be empty to indicate any method.
func (c *Collector) AnnotateOperation(method, pattern string, setup ...func(oc openapi.OperationContext) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ocAnnotations == nil {
		c.ocAnnotations = make(map[string][]func(oc openapi.OperationContext) error)
	}

	c.ocAnnotations[method+pattern] = append(c.ocAnnotations[method+pattern], setup...)
}

// HasAnnotation indicates if there is at least one annotation registered for this operation.
func (c *Collector) HasAnnotation(method, pattern string) bool {
	if len(c.ocAnnotations[method+pattern]) > 0 {
		return true
	}

	return len(c.ocAnnotations[pattern]) > 0
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

	reflector := c.Refl()

	oc, err := reflector.NewOperationContext(method, pattern)
	if err != nil {
		return err
	}

	for _, setup := range c.ocAnnotations[pattern] {
		if err = setup(oc); err != nil {
			return err
		}
	}

	for _, setup := range c.ocAnnotations[method+pattern] {
		if err = setup(oc); err != nil {
			return err
		}
	}

	for _, setup := range annotations {
		if err = setup(oc); err != nil {
			return err
		}
	}

	return reflector.AddOperation(oc)
}

// CollectUseCase adds use case handler to documentation.
func (c *Collector) CollectUseCase(
	method, pattern string,
	u usecase.Interactor,
	h rest.HandlerTrait,
	annotations ...func(oc openapi.OperationContext) error,
) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	defer func() {
		if err != nil {
			err = fmt.Errorf("reflect API schema for %s %s: %w", method, pattern, err)
		}
	}()

	reflector := c.Refl()

	oc, err := reflector.NewOperationContext(method, pattern)
	if err != nil {
		return err
	}

	c.setupInput(oc, u, h)
	c.setupOutput(oc, u, h)
	c.processUseCase(oc, u, h)

	an := append([]func(oc openapi.OperationContext) error(nil), c.ocAnnotations[method+pattern]...)
	an = append(an, h.OpenAPIAnnotations...)
	an = append(an, annotations...)

	for _, setup := range an {
		if err = setup(oc); err != nil {
			return err
		}
	}

	if o3, ok := oc.(openapi3.OperationExposer); ok {
		op := o3.Operation()

		for _, setup := range c.annotations[method+pattern] {
			if err = setup(op); err != nil {
				return err
			}
		}

		//nolint:staticcheck // To be removed with deprecations cleanup.
		for _, setup := range h.OperationAnnotations {
			if err = setup(op); err != nil {
				return err
			}
		}
	}

	return reflector.AddOperation(oc)
}

// Collect adds use case handler to documentation.
//
// Deprecated: use CollectUseCase.
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
			err = fmt.Errorf("reflect API schema for %s %s: %w", method, pattern, err)
		}
	}()

	reflector := c.Refl()

	oc, err := reflector.NewOperationContext(method, pattern)
	if err != nil {
		return err
	}

	c.setupInput(oc, u, h)
	c.setupOutput(oc, u, h)
	c.processUseCase(oc, u, h)

	for _, setup := range c.ocAnnotations[method+pattern] {
		err = setup(oc)
		if err != nil {
			return err
		}
	}

	if o3, ok := oc.(openapi3.OperationExposer); ok {
		op := o3.Operation()

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
	}

	return reflector.AddOperation(oc)
}

func (c *Collector) setupOutput(oc openapi.OperationContext, u usecase.Interactor, h rest.HandlerTrait) {
	var (
		hasOutput   usecase.HasOutputPort
		status      = http.StatusOK
		noContent   bool
		output      interface{}
		contentType = h.SuccessContentType
	)

	if usecase.As(u, &hasOutput) {
		output = hasOutput.OutputPort()

		if rest.OutputHasNoContent(output) {
			status = http.StatusNoContent
			noContent = true
		}
	} else {
		status = http.StatusNoContent
		noContent = true
	}

	if !noContent && contentType == "" {
		contentType = c.DefaultSuccessResponseContentType
	}

	if oc.Method() == http.MethodHead {
		output = nil
	}

	setupCU := func(cu *openapi.ContentUnit) {
		cu.ContentType = contentType
		cu.SetFieldMapping(openapi.InHeader, h.RespHeaderMapping)
	}

	if outputWithStatus, ok := output.(rest.OutputWithHTTPStatus); ok {
		for _, status := range outputWithStatus.ExpectedHTTPStatuses() {
			oc.AddRespStructure(output, func(cu *openapi.ContentUnit) {
				cu.HTTPStatus = status
				setupCU(cu)
			})
		}
	} else {
		if h.SuccessStatus != 0 {
			status = h.SuccessStatus
		}

		oc.AddRespStructure(output, func(cu *openapi.ContentUnit) {
			cu.HTTPStatus = status
			setupCU(cu)
		})
	}
}

func (c *Collector) setupInput(oc openapi.OperationContext, u usecase.Interactor, h rest.HandlerTrait) {
	var hasInput usecase.HasInputPort

	if usecase.As(u, &hasInput) {
		oc.AddReqStructure(hasInput.InputPort(), func(cu *openapi.ContentUnit) {
			setFieldMapping(cu, h.ReqMapping)
		})
	}
}

func setFieldMapping(cu *openapi.ContentUnit, mapping rest.RequestMapping) {
	if mapping != nil {
		cu.SetFieldMapping(openapi.InQuery, mapping[rest.ParamInQuery])
		cu.SetFieldMapping(openapi.InPath, mapping[rest.ParamInPath])
		cu.SetFieldMapping(openapi.InHeader, mapping[rest.ParamInHeader])
		cu.SetFieldMapping(openapi.InCookie, mapping[rest.ParamInCookie])
		cu.SetFieldMapping(openapi.InFormData, mapping[rest.ParamInFormData])
	}
}

func (c *Collector) processUseCase(oc openapi.OperationContext, u usecase.Interactor, h rest.HandlerTrait) {
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

			oc.SetID(idSuf)
		}
	}

	if usecase.As(u, &hasTitle) {
		title := hasTitle.Title()

		if title != "" {
			oc.SetSummary(hasTitle.Title())
		}
	}

	if usecase.As(u, &hasTags) {
		tags := hasTags.Tags()

		if len(tags) > 0 {
			oc.SetTags(hasTags.Tags()...)
		}
	}

	if usecase.As(u, &hasDescription) {
		desc := hasDescription.Description()

		if desc != "" {
			oc.SetDescription(hasDescription.Description())
		}
	}

	if usecase.As(u, &hasDeprecated) && hasDeprecated.IsDeprecated() {
		oc.SetIsDeprecated(true)
	}

	c.processOCExpectedErrors(oc, u, h)
}

func (c *Collector) setOCJSONResponse(oc openapi.OperationContext, output interface{}, statusCode int) {
	oc.AddRespStructure(output, func(cu *openapi.ContentUnit) {
		cu.HTTPStatus = statusCode

		if described, ok := output.(jsonschema.Described); ok {
			cu.Description = described.Description()
		}

		if output != nil {
			cu.ContentType = c.DefaultErrorResponseContentType
		}
	})
}

func (c *Collector) processOCExpectedErrors(oc openapi.OperationContext, u usecase.Interactor, h rest.HandlerTrait) {
	var (
		errsByCode        = map[int][]interface{}{}
		statusCodes       []int
		hasExpectedErrors usecase.HasExpectedErrors
	)

	if !usecase.As(u, &hasExpectedErrors) {
		return
	}

	for _, e := range hasExpectedErrors.ExpectedErrors() {
		var (
			errResp     interface{}
			statusCode  int
			description string
		)

		var described jsonschema.Described
		if errors.As(e, &described) {
			description = described.Description()
		}

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

		oc.AddRespStructure(errResp, func(cu *openapi.ContentUnit) {
			cu.HTTPStatus = statusCode
			cu.Description = description

			if errResp != nil {
				cu.ContentType = c.DefaultErrorResponseContentType
			}
		})
	}

	c.combineOCErrors(oc, statusCodes, errsByCode)
}

func (c *Collector) combineOCErrors(oc openapi.OperationContext, statusCodes []int, errsByCode map[int][]interface{}) {
	for _, statusCode := range statusCodes {
		errResps := errsByCode[statusCode]

		if len(errResps) == 1 || c.CombineErrors == "" {
			c.setOCJSONResponse(oc, errResps[0], statusCode)
		} else {
			switch c.CombineErrors {
			case "oneOf":
				c.setOCJSONResponse(oc, jsonschema.OneOf(errResps...), statusCode)
			case "anyOf":
				c.setOCJSONResponse(oc, jsonschema.AnyOf(errResps...), statusCode)
			default:
				panic("oneOf/anyOf expected for openapi.Collector.CombineErrors, " +
					c.CombineErrors + " received")
			}
		}
	}
}

type unknownFieldsValidator interface {
	ForbidUnknownParams(in rest.ParamIn, forbidden bool)
}

// ProvideRequestJSONSchemas provides JSON Schemas for request structure.
func (c *Collector) ProvideRequestJSONSchemas(
	method string,
	input interface{},
	mapping rest.RequestMapping,
	validator rest.JSONSchemaValidator,
) error {
	cu := openapi.ContentUnit{}
	cu.Structure = input
	setFieldMapping(&cu, mapping)

	r := c.Refl()

	err := r.WalkRequestJSONSchemas(method, cu, c.jsonSchemaCallback(validator, r), func(oc openapi.OperationContext) {
		fv, ok := validator.(unknownFieldsValidator)
		if !ok {
			return
		}

		for _, in := range []openapi.In{openapi.InQuery, openapi.InCookie, openapi.InHeader} {
			if oc.UnknownParamsAreForbidden(in) {
				fv.ForbidUnknownParams(rest.ParamIn(in), true)
			}
		}
	})

	return err
}

// ProvideResponseJSONSchemas provides JSON schemas for response structure.
func (c *Collector) ProvideResponseJSONSchemas(
	statusCode int,
	contentType string,
	output interface{},
	headerMapping map[string]string,
	validator rest.JSONSchemaValidator,
) error {
	cu := openapi.ContentUnit{}
	cu.Structure = output
	cu.SetFieldMapping(openapi.InHeader, headerMapping)
	cu.ContentType = contentType
	cu.HTTPStatus = statusCode

	if cu.ContentType == "" {
		cu.ContentType = c.DefaultSuccessResponseContentType
	}

	r := c.Refl()
	err := r.WalkResponseJSONSchemas(cu, c.jsonSchemaCallback(validator, r), nil)

	return err
}

func (c *Collector) jsonSchemaCallback(validator rest.JSONSchemaValidator, r openapi.Reflector) openapi.JSONSchemaCallback {
	return func(in openapi.In, paramName string, schema *jsonschema.SchemaOrBool, required bool) error {
		loc := string(in) + "." + paramName
		if loc == "body.body" {
			loc = "body"
		}

		if schema == nil || schema.IsTrivial(r.ResolveJSONSchemaRef) {
			if err := validator.AddSchema(rest.ParamIn(in), paramName, nil, required); err != nil {
				return fmt.Errorf("add validation schema %s: %w", loc, err)
			}

			return nil
		}

		schemaData, err := schema.JSONSchemaBytes()
		if err != nil {
			return fmt.Errorf("marshal schema %s: %w", loc, err)
		}

		if err = validator.AddSchema(rest.ParamIn(in), paramName, schemaData, required); err != nil {
			return fmt.Errorf("add validation schema %s: %w", loc, err)
		}

		return nil
	}
}

func (c *Collector) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	document, err := json.MarshalIndent(c.SpecSchema(), "", " ")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	rw.Header().Set("Content-Type", "application/json")

	_, err = rw.Write(document)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}
