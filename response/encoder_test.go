package response_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
)

func TestEncoder_SetupOutput(t *testing.T) {
	e := response.Encoder{}

	type EmbeddedHeader struct {
		Foo int    `header:"X-Foo" json:"-"`
		Bar string `cookie:"bar" json:"-"`
	}

	type outputPort struct {
		EmbeddedHeader
		Name    string   `header:"X-Name" json:"-"`
		Items   []string `json:"items"`
		Cookie  int      `cookie:"coo,httponly,path=/foo" json:"-"`
		Cookie2 bool     `cookie:"coo2,httponly,secure,samesite=lax,path=/foo,max-age=86400" json:"-"`
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

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	output := e.MakeOutput(w, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	out.Foo = 321
	out.Bar = "baz"
	out.Name = "Jane"
	out.Items = []string{"one", "two", "three"}
	out.Cookie = 123
	out.Cookie2 = true

	e.WriteSuccessfulResponse(w, r, output, ht)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Jane", w.Header().Get("X-Name"))
	assert.Equal(t, "321", w.Header().Get("X-Foo"))
	assert.Equal(t, []string{
		"bar=baz",
		"coo=123; Path=/foo; HttpOnly",
		"coo2=true; Path=/foo; Max-Age=86400; HttpOnly; Secure; SameSite=Lax",
	}, w.Header()["Set-Cookie"])
	assert.Equal(t, "application/x-vnd-json", w.Header().Get("Content-Type"))
	assert.Equal(t, "32", w.Header().Get("Content-Length"))
	assert.Equal(t, `{"items":["one","two","three"]}`+"\n", w.Body.String())

	w = httptest.NewRecorder()
	e.WriteErrResponse(w, r, http.StatusExpectationFailed, rest.ErrResponse{
		ErrorText: "failed",
	})
	assert.Equal(t, http.StatusExpectationFailed, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "19", w.Header().Get("Content-Length"))
	assert.Equal(t, `{"error":"failed"}`+"\n", w.Body.String())

	out.Name = "Ja"
	w = httptest.NewRecorder()
	e.WriteSuccessfulResponse(w, r, output, ht)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Header().Get("X-Name"))
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "140", w.Header().Get("Content-Length"))
	assert.Equal(t, `{"status":"INTERNAL","error":"internal: bad response: validation failed",`+
		`"context":{"header:X-Name":["#: length must be >= 3, but got 2"]}}`+"\n", w.Body.String())
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

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	output := e.MakeOutput(w, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	out.Name = "Jane"

	_, err = out.Write([]byte("1,2,3"))
	require.NoError(t, err)

	e.WriteSuccessfulResponse(w, r, output, ht)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-vnd-foo", w.Header().Get("Content-Type"))
	assert.Equal(t, "1,2,3", w.Body.String())
	assert.Equal(t, "Jane", w.Header().Get("X-Name"))
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

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	output := e.MakeOutput(w, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	_, err = out.Write([]byte("1,2,3"))
	require.NoError(t, err)

	e.WriteSuccessfulResponse(w, r, output, ht)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-vnd-foo", w.Header().Get("Content-Type"))
	assert.Equal(t, "1,2,3", w.Body.String())
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

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	output := e.MakeOutput(w, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	out.Name = "Jane"
	out.Items = []string{"one", "two", "three"}

	e.WriteSuccessfulResponse(w, r, output, ht)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Jane", w.Header().Get("X-Name"))
	assert.Equal(t, "application/x-vnd-json", w.Header().Get("Content-Type"))
	assert.Equal(t, "32", w.Header().Get("Content-Length"))
	assert.Equal(t, `{"items":["one","two","three"]}`+"\n", w.Body.String())
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

func TestEncoder_SetupOutput_httpStatus(t *testing.T) {
	e := response.Encoder{}
	ht := rest.HandlerTrait{}
	e.SetupOutput(outputWithHTTPStatuses{}, &ht)
	assert.Equal(t, http.StatusCreated, ht.SuccessStatus)
}

func TestEncoder_Writer_httpStatus(t *testing.T) {
	e := response.Encoder{}
	e.SetupOutput(outputWithHTTPStatuses{}, &rest.HandlerTrait{})

	r, err := http.NewRequest(http.MethodPost, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	output := e.MakeOutput(w, rest.HandlerTrait{})
	e.WriteSuccessfulResponse(w, r, output, rest.HandlerTrait{})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestEmbeddedSetter_SetResponseWriter(t *testing.T) {
	e := response.Encoder{}

	type EmbeddedHeader struct {
		Foo int    `header:"X-Foo" json:"-"`
		Bar string `cookie:"bar" json:"-"`
	}

	type outputPort struct {
		response.EmbeddedSetter
		EmbeddedHeader
		Name    string   `header:"X-Name" json:"-"`
		Items   []string `json:"items"`
		Cookie  int      `cookie:"coo,httponly,path=/foo" json:"-"`
		Cookie2 bool     `cookie:"coo2,httponly,secure,samesite=lax,path=/foo,max-age=86400" json:"-"`
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

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	output := e.MakeOutput(w, ht)

	out, ok := output.(*outputPort)
	assert.True(t, ok)

	out.Foo = 321
	out.Bar = "baz"
	out.Name = "Jane"
	out.Items = []string{"one", "two", "three"}
	out.Cookie = 123
	out.Cookie2 = true

	e.WriteSuccessfulResponse(w, r, output, ht)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Jane", w.Header().Get("X-Name"))
	assert.Equal(t, "321", w.Header().Get("X-Foo"))
	assert.Equal(t, []string{
		"bar=baz",
		"coo=123; Path=/foo; HttpOnly",
		"coo2=true; Path=/foo; Max-Age=86400; HttpOnly; Secure; SameSite=Lax",
	}, w.Header()["Set-Cookie"])
	assert.Equal(t, "application/x-vnd-json", w.Header().Get("Content-Type"))
	assert.Equal(t, "32", w.Header().Get("Content-Length"))
	assert.Equal(t, `{"items":["one","two","three"]}`+"\n", w.Body.String())
	assert.Equal(t, w, out.ResponseWriter())
}

func TestEncoder_contentTypeRaw(t *testing.T) {
	type Resp struct {
		TextBody string `contentType:"text/plain"`
		CSVBody  string `contentType:"text/csv"`
	}

	ht := rest.HandlerTrait{}

	e := response.Encoder{}
	e.SetupOutput(Resp{}, &ht)

	w := httptest.NewRecorder()

	v := e.MakeOutput(w, ht)

	re, ok := v.(*Resp)
	assert.True(t, ok)

	re.CSVBody = "hello,world"

	e.WriteSuccessfulResponse(w, nil, v, ht)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "hello,world", w.Body.String())
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
}
