package openapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	jschema "github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

var _ rest.JSONSchemaValidator = validatorMock{}

type validatorMock struct {
	ValidateDataFunc     func(in rest.ParamIn, namedData map[string]interface{}) error
	ValidateJSONBodyFunc func(jsonBody []byte) error
	HasConstraintsFunc   func(in rest.ParamIn) bool
	AddSchemaFunc        func(in rest.ParamIn, name string, schemaData []byte, required bool) error
}

func (v validatorMock) ValidateData(in rest.ParamIn, namedData map[string]interface{}) error {
	return v.ValidateDataFunc(in, namedData)
}

func (v validatorMock) ValidateJSONBody(jsonBody []byte) error {
	return v.ValidateJSONBodyFunc(jsonBody)
}

func (v validatorMock) HasConstraints(in rest.ParamIn) bool {
	return v.HasConstraintsFunc(in)
}

func (v validatorMock) AddSchema(in rest.ParamIn, name string, schemaData []byte, required bool) error {
	return v.AddSchemaFunc(in, name, schemaData, required)
}

func TestCollector_Collect(t *testing.T) {
	c := openapi.Collector{
		BasePath: "http://example.com/",
	}

	c.Annotate(http.MethodPost, "/foo", func(op *openapi3.Operation) error {
		op.WithDescription("This is Foo.")

		return nil
	})

	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	type input struct {
		Q string  `query:"q" required:"true"`
		H int     `header:"h" minimum:"10"`
		F float32 `formData:"f"`
		C bool    `cookie:"c"`
	}

	type output struct {
		Name   string `json:"name" maxLength:"32"`
		Number int    `json:"number"`
		Trace  string `header:"X-Trace"`
	}

	u.SetTitle("Create Task")
	u.SetDescription("Create task to be done.")
	u.Input = new(input)
	u.Output = new(output)
	u.SetIsDeprecated(true)
	u.SetExpectedErrors(
		status.AlreadyExists,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	require.NoError(t, c.CollectUseCase(http.MethodPost, "/foo", u, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	require.NoError(t, c.CollectUseCase(http.MethodGet, "/foo", nil, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	j, err := json.MarshalIndent(c.SpecSchema(), "", " ")
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	c.ServeHTTP(rw, nil)

	assertjson.Equal(t, j, rw.Body.Bytes())

	val := validatorMock{
		AddSchemaFunc: func(in rest.ParamIn, name string, schemaData []byte, required bool) error {
			return nil
		},
	}
	assert.NoError(t, c.ProvideResponseJSONSchemas(http.StatusOK, "application/json", new(output), nil, val))
}

func TestCollector_Collect_requestMapping(t *testing.T) {
	type input struct {
		InHeader   string `minLength:"2"`
		InQuery    jschema.Date
		InCookie   *time.Time
		InFormData time.Time
		InPath     bool
		InFile     multipart.File
	}

	u := usecase.IOInteractor{}

	u.SetTitle("Title")
	u.SetName("name")
	u.SetIsDeprecated(true)
	u.Input = new(input)

	mapping := rest.RequestMapping{
		rest.ParamInFormData: map[string]string{"InFormData": "in_form_data", "InFile": "upload"},
		rest.ParamInCookie:   map[string]string{"InCookie": "in_cookie"},
		rest.ParamInQuery:    map[string]string{"InQuery": "in_query"},
		rest.ParamInHeader:   map[string]string{"InHeader": "X-In-Header"},
		rest.ParamInPath:     map[string]string{"InPath": "in-path"},
	}

	h := rest.HandlerTrait{
		ReqMapping: mapping,
	}

	collector := openapi.Collector{}

	require.NoError(t, collector.CollectUseCase(http.MethodPost, "/test/{in-path}", u, h))
	require.NoError(t, collector.CollectUseCase(http.MethodPut, "/test/{in-path}", u, h))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3","info":{"title":"","version":""},
	  "paths":{
		"/test/{in-path}":{
		  "post":{
			"summary":"Title","operationId":"name",
			"parameters":[
			  {
				"name":"in_query","in":"query",
				"schema":{"type":"string","format":"date"}
			  },
			  {
				"name":"in-path","in":"path","required":true,
				"schema":{"type":"boolean"}
			  },
			  {
				"name":"in_cookie","in":"cookie",
				"schema":{"type":"string","format":"date-time","nullable":true}
			  },
			  {
				"name":"X-In-Header","in":"header",
				"schema":{"minLength":2,"type":"string"}
			  }
			],
			"requestBody":{
			  "content":{
				"multipart/form-data":{
				  "schema":{"$ref":"#/components/schemas/FormDataOpenapiTestInput"}
				}
			  }
			},
			"responses":{"204":{"description":"No Content"}},"deprecated":true
		  },
		  "put":{
			"summary":"Title","operationId":"name2",
			"parameters":[
			  {
				"name":"in_query","in":"query",
				"schema":{"type":"string","format":"date"}
			  },
			  {
				"name":"in-path","in":"path","required":true,
				"schema":{"type":"boolean"}
			  },
			  {
				"name":"in_cookie","in":"cookie",
				"schema":{"type":"string","format":"date-time","nullable":true}
			  },
			  {
				"name":"X-In-Header","in":"header",
				"schema":{"minLength":2,"type":"string"}
			  }
			],
			"requestBody":{
			  "content":{
				"multipart/form-data":{
				  "schema":{"$ref":"#/components/schemas/FormDataOpenapiTestInput"}
				}
			  }
			},
			"responses":{"204":{"description":"No Content"}},"deprecated":true
		  }
		}
	  },
	  "components":{
		"schemas":{
		  "FormDataMultipartFile":{"type":"string","format":"binary","nullable":true},
		  "FormDataOpenapiTestInput":{
			"type":"object",
			"properties":{
			  "in_form_data":{"type":"string","format":"date-time"},
			  "upload":{"$ref":"#/components/schemas/FormDataMultipartFile"}
			}
		  }
		}
	  }
	}`, collector.SpecSchema())

	val := validatorMock{
		AddSchemaFunc: func(in rest.ParamIn, name string, schemaData []byte, required bool) error {
			return nil
		},
	}
	assert.NoError(t, collector.ProvideRequestJSONSchemas(http.MethodPost, new(input), mapping, val))
}

// anotherErr is another custom error.
type anotherErr struct {
	Foo int `json:"foo"`
}

func (anotherErr) Error() string {
	return "foo happened"
}

func TestCollector_Collect_CombineErrors(t *testing.T) {
	u := usecase.IOInteractor{}

	u.SetTitle("Title")
	u.SetName("name")
	u.SetExpectedErrors(status.InvalidArgument, anotherErr{}, status.FailedPrecondition, status.AlreadyExists)

	h := rest.HandlerTrait{}
	h.MakeErrResp = func(ctx context.Context, err error) (int, interface{}) {
		code, er := rest.Err(err)

		var ae anotherErr

		if errors.As(err, &ae) {
			return http.StatusBadRequest, ae
		}

		return code, er
	}

	collector := openapi.Collector{}
	collector.CombineErrors = "oneOf"

	require.NoError(t, collector.CollectUseCase(http.MethodPost, "/test", u, h))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3","info":{"title":"","version":""},
	  "paths":{
		"/test":{
		  "post":{
			"summary":"Title","operationId":"name",
			"responses":{
			  "204":{"description":"No Content"},
			  "400":{
				"description":"Bad Request",
				"content":{
				  "application/json":{
					"schema":{
					  "oneOf":[
						{"$ref":"#/components/schemas/RestErrResponse"},
						{"$ref":"#/components/schemas/OpenapiTestAnotherErr"}
					  ]
					}
				  }
				}
			  },
			  "409":{
				"description":"Conflict",
				"content":{
				  "application/json":{"schema":{"$ref":"#/components/schemas/RestErrResponse"}}
				}
			  },
			  "412":{
				"description":"Precondition Failed",
				"content":{
				  "application/json":{"schema":{"$ref":"#/components/schemas/RestErrResponse"}}
				}
			  }
			}
		  }
		}
	  },
	  "components":{
		"schemas":{
		  "OpenapiTestAnotherErr":{"type":"object","properties":{"foo":{"type":"integer"}}},
		  "RestErrResponse":{
			"type":"object",
			"properties":{
			  "code":{"type":"integer","description":"Application-specific error code."},
			  "context":{
				"type":"object","additionalProperties":{},
				"description":"Application context."
			  },
			  "error":{"type":"string","description":"Error message."},
			  "status":{"type":"string","description":"Status text."}
			}
		  }
		}
	  }
	}`, collector.SpecSchema())
}

// Output that implements OutputWithHTTPStatus interface.
type outputWithHTTPStatuses struct {
	Number int `json:"number"`
}

func (outputWithHTTPStatuses) HTTPStatus() int {
	return http.StatusCreated
}

func (outputWithHTTPStatuses) ExpectedHTTPStatuses() []int {
	return []int{http.StatusCreated, http.StatusOK}
}

func TestCollector_Collect_multipleHttpStatuses(t *testing.T) {
	c := openapi.Collector{}
	u := usecase.IOInteractor{}
	u.SetTitle("Title")
	u.SetName("name")
	u.Input = new(struct{})
	u.Output = new(outputWithHTTPStatuses)

	require.NoError(t, c.CollectUseCase(http.MethodPost, "/foo", u, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	assertjson.EqMarshal(t, `{
	  "openapi": "3.0.3",
	  "info": {
		"title": "",
		"version": ""
	  },
	  "paths": {
		"/foo": {
		  "post": {
			"summary": "Title",
			"operationId": "name",
			"responses": {
			  "200": {
				"description": "OK",
				"content": {
				  "application/json": {
					"schema": {
					  "$ref": "#/components/schemas/OpenapiTestOutputWithHTTPStatuses"
					}
				  }
				}
			  },
			  "201": {
				"description": "Created",
				"content": {
				  "application/json": {
					"schema": {
					  "$ref": "#/components/schemas/OpenapiTestOutputWithHTTPStatuses"
					}
				  }
				}
			  }
			}
		  }
		}
	  },
	  "components": {
		"schemas": {
		  "OpenapiTestOutputWithHTTPStatuses": {
			"type": "object",
			"properties": {
			  "number": {
				"type": "integer"
			  }
			}
		  }
		}
	  }
	}`, c.SpecSchema())
}

func TestCollector_Collect_queryObject(t *testing.T) {
	c := openapi.Collector{}
	u := usecase.IOInteractor{}

	type jsonFilter struct {
		Foo string `json:"foo"`
	}

	type deepObjectFilter struct {
		Bar string `query:"bar"`
	}

	type inputQueryObject struct {
		Query            map[int]float64  `query:"in_query" description:"Object value in query."`
		JSONFilter       jsonFilter       `query:"json_filter" description:"JSON object value in query."`
		DeepObjectFilter deepObjectFilter `query:"deep_object_filter" description:"Deep object value in query params."`
	}

	u.Input = new(inputQueryObject)

	require.NoError(t, c.CollectUseCase(http.MethodGet, "/foo", u, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3","info":{"title":"","version":""},
	  "paths":{
		"/foo":{
		  "get":{
			"parameters":[
			  {
				"name":"in_query","in":"query",
				"description":"Object value in query.","style":"deepObject",
				"explode":true,
				"schema":{
				  "type":"object","additionalProperties":{"type":"number"},
				  "description":"Object value in query."
				}
			  },
			  {
				"name":"json_filter","in":"query",
				"description":"JSON object value in query.",
				"content":{
				  "application/json":{
					"schema":{"$ref":"#/components/schemas/QueryOpenapiTestJsonFilter"}
				  }
				}
			  },
			  {
				"name":"deep_object_filter","in":"query",
				"description":"Deep object value in query params.",
				"style":"deepObject","explode":true,
				"schema":{"$ref":"#/components/schemas/QueryOpenapiTestDeepObjectFilter"}
			  }
			],
			"responses":{"204":{"description":"No Content"}}
		  }
		}
	  },
	  "components":{
		"schemas":{
		  "QueryOpenapiTestDeepObjectFilter":{"type":"object","properties":{"bar":{"type":"string"}}},
		  "QueryOpenapiTestJsonFilter":{"type":"object","properties":{"foo":{"type":"string"}}}
		}
	  }
	}`, c.SpecSchema())
}

func TestCollector_Collect_head_no_response(t *testing.T) {
	c := openapi.Collector{}
	u := usecase.IOInteractor{}

	type resp struct {
		Foo string `json:"foo"`
		Bar string `header:"X-Bar"`
	}

	u.Output = new(resp)

	require.NoError(t, c.CollectUseCase(http.MethodHead, "/foo", u, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	require.NoError(t, c.CollectUseCase(http.MethodGet, "/foo", u, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3","info":{"title":"","version":""},
	  "paths":{
		"/foo":{
		  "get":{
			"responses":{
			  "200":{
				"description":"OK",
				"headers":{"X-Bar":{"style":"simple","schema":{"type":"string"}}},
				"content":{
				  "application/json":{"schema":{"$ref":"#/components/schemas/OpenapiTestResp"}}
				}
			  }
			}
		  },
		  "head":{
			"responses":{
			  "200":{
				"description":"OK",
				"headers":{"X-Bar":{"style":"simple","schema":{"type":"string"}}}
			  }
			}
		  }
		}
	  },
	  "components":{
		"schemas":{
		  "OpenapiTestResp":{"type":"object","properties":{"foo":{"type":"string"}}}
		}
	  }
	}`, c.SpecSchema())
}
