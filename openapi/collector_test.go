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

	require.NoError(t, c.Collect(http.MethodPost, "/foo", u, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	require.NoError(t, c.Collect(http.MethodGet, "/foo", nil, rest.HandlerTrait{
		ReqValidator: &jsonschema.Validator{},
	}))

	j, err := json.MarshalIndent(c.Reflector().Spec, "", " ")
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

	require.NoError(t, collector.Collect(http.MethodPost, "/test/{in-path}", u, h))
	require.NoError(t, collector.Collect(http.MethodPut, "/test/{in-path}", u, h))

	assertjson.EqualMarshal(t, []byte(`{
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
	}`), collector.Reflector().SpecEns())

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

	require.NoError(t, collector.Collect(http.MethodPost, "/test", u, h))

	assertjson.EqualMarshal(t, []byte(`{
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
						{"$ref":"#/components/schemas/OpenapiTestAnotherErr"},
						{"$ref":"#/components/schemas/RestErrResponse"}
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
	}`), collector.Reflector().SpecEns())
}
