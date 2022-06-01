package response_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/valyala/fasthttp"
)

func TestEncoder_SetupOutput(t *testing.T) {
	e := response.Encoder{}

	type outputPort struct {
		Name  string   `header:"X-Name" json:"-"`
		Items []string `json:"items"`
	}

	ht := rest.HandlerTrait{
		SuccessContentType: "application/x-vnd-json",
	}

	validator := jsonschema.Validator{}
	require.NoError(t, validator.AddSchema(
		rest.ParamInHeader,
		"X-Name",
		[]byte(`{"type":"string","minLength":3}`),
		false),
	)

	ht.RespValidator = &validator

	e.SetupOutput(new(outputPort), &ht)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	output := e.MakeOutput(rc, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	out.Name = "Jane"
	out.Items = []string{"one", "two", "three"}

	e.WriteSuccessfulResponse(rc, output, ht)
	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "Jane", string(rc.Response.Header.Peek("X-Name")))
	assert.Equal(t, "application/x-vnd-json", string(rc.Response.Header.Peek("Content-Type")))
	assert.Equal(t, "32", string(rc.Response.Header.Peek("Content-Length")))
	assert.Equal(t, `{"items":["one","two","three"]}`+"\n", string(rc.Response.Body()))

	rc.Response = fasthttp.Response{}
	e.WriteErrResponse(rc, http.StatusExpectationFailed, rest.ErrResponse{
		ErrorText: "failed",
	})
	assert.Equal(t, http.StatusExpectationFailed, rc.Response.StatusCode())
	assert.Equal(t, "application/json; charset=utf-8", string(rc.Response.Header.Peek("Content-Type")))
	assert.Equal(t, "19", string(rc.Response.Header.Peek("Content-Length")))
	assert.Equal(t, `{"error":"failed"}`+"\n", string(rc.Response.Body()))

	out.Name = "Ja"
	rc.Response = fasthttp.Response{}
	e.WriteSuccessfulResponse(rc, output, ht)
	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	assert.Equal(t, "", string(rc.Response.Header.Peek("X-Name")))
	assert.Equal(t, "application/json; charset=utf-8", string(rc.Response.Header.Peek("Content-Type")))
	assert.Equal(t, "140", string(rc.Response.Header.Peek("Content-Length")))
	assert.Equal(t, `{"status":"INTERNAL","error":"internal: bad response: validation failed",`+
		`"context":{"header:X-Name":["#: length must be >= 3, but got 2"]}}`+"\n", string(rc.Response.Body()))
}

func TestEncoder_SetupOutput_withWriter(t *testing.T) {
	e := response.Encoder{}

	ht := rest.HandlerTrait{
		SuccessContentType: "application/x-vnd-foo",
	}

	type outputPort struct {
		Name string `header:"X-Name" json:"-"`
		usecase.OutputWithEmbeddedWriter
	}

	e.SetupOutput(new(outputPort), &ht)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	output := e.MakeOutput(rc, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	out.Name = "Jane"

	_, err := out.Write([]byte("1,2,3"))
	require.NoError(t, err)

	e.WriteSuccessfulResponse(rc, output, ht)
	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "application/x-vnd-foo", string(rc.Response.Header.Peek("Content-Type")))
	assert.Equal(t, "1,2,3", string(rc.Response.Body()))
	assert.Equal(t, "Jane", string(rc.Response.Header.Peek("X-Name")))
}

func TestEncoder_SetupOutput_withWriterContentType(t *testing.T) {
	e := response.Encoder{}

	ht := rest.HandlerTrait{
		SuccessContentType: "application/x-vnd-foo",
	}

	type outputPort struct {
		usecase.OutputWithEmbeddedWriter
	}

	e.SetupOutput(new(outputPort), &ht)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	output := e.MakeOutput(rc, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	_, err := out.Write([]byte("1,2,3"))
	require.NoError(t, err)

	e.WriteSuccessfulResponse(rc, output, ht)
	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "application/x-vnd-foo", string(rc.Response.Header.Peek("Content-Type")))
	assert.Equal(t, "1,2,3", string(rc.Response.Body()))
}

func TestEncoder_SetupOutput_nonPtr(t *testing.T) {
	e := response.Encoder{}

	type outputPort struct {
		Name  string   `header:"X-Name" json:"-"`
		Items []string `json:"items"`
	}

	ht := rest.HandlerTrait{
		SuccessContentType: "application/x-vnd-json",
	}

	validator := jsonschema.Validator{}
	require.NoError(t, validator.AddSchema(
		rest.ParamInHeader,
		"X-Name",
		[]byte(`{"type":"string","minLength":3}`),
		false),
	)

	ht.RespValidator = &validator

	e.SetupOutput(outputPort{}, &ht)

	rc := &fasthttp.RequestCtx{}
	rc.Request.SetRequestURI("/")

	output := e.MakeOutput(rc, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	out.Name = "Jane"
	out.Items = []string{"one", "two", "three"}

	e.WriteSuccessfulResponse(rc, output, ht)
	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	assert.Equal(t, "Jane", string(rc.Response.Header.Peek("X-Name")))
	assert.Equal(t, "application/x-vnd-json", string(rc.Response.Header.Peek("Content-Type")))
	assert.Equal(t, "32", string(rc.Response.Header.Peek("Content-Length")))
	assert.Equal(t, `{"items":["one","two","three"]}`+"\n", string(rc.Response.Body()))
}
