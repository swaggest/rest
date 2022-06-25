package fhttp_test

import (
	"net/http"
	"testing"

	"github.com/bool64/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra/fhttp"
	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/infra/service"
)

func Test_taskLifeSpan(t *testing.T) {
	l := infra.NewServiceLocator(service.Config{})
	defer l.Close()

	srv := fchi.NewTestServer(fhttp.NewRouter(l))
	defer srv.Close()

	rc := httpmock.NewClient(srv.URL)

	rc.WithMethod(http.MethodPost).WithURI("/dev/tasks").
		WithContentType("application/json").
		WithBody([]byte(`{"deadline": "2020-05-17T11:12:42.085Z","goal": "string"}`)).
		Concurrently()

	assert.NoError(t, rc.ExpectResponseStatus(http.StatusCreated))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"createdAt": "<ignore-diff>",`+
		`"deadline": "2020-05-17T11:12:42.085Z","goal": "string","id": 1}`)))

	assert.NoError(t, rc.ExpectOtherResponsesStatus(http.StatusConflict))
	assert.NoError(t, rc.ExpectOtherResponsesBody([]byte(`{"status":"ALREADY_EXISTS",`+
		`"error":"already exists: task with same goal already exists",`+
		`"context":{"task":{"id":1,"goal":"string","deadline":"2020-05-17T11:12:42.085Z","createdAt":"<ignore-diff>"}}}`)))

	rc.Reset().WithMethod(http.MethodPost).WithURI("/dev/tasks").
		WithContentType("application/json").
		WithBody([]byte(`{"deadline": "2020-35-17T11:12:42.085Z","goal": "do it!"}`)).
		Concurrently()

	assert.NoError(t, rc.ExpectResponseStatus(http.StatusBadRequest))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{
	 "status":"INVALID_ARGUMENT",
	 "error":"<ignore-diff>"
	}`)))

	rc.Reset().WithMethod(http.MethodPost).WithURI("/dev/tasks").
		WithContentType("application/json").
		WithBody([]byte(`{"deadline": "2020-05-17T11:12:42.085Z","goal": ""}`)).
		Concurrently()

	assert.NoError(t, rc.ExpectResponseStatus(http.StatusBadRequest))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"status":"INVALID_ARGUMENT",`+
		`"error":"invalid argument: validation failed","context":{"body":["#/goal: length must be \u003e= 1, but got 0"]}}`)))

	rc.Reset().WithMethod(http.MethodPost).WithURI("/dev/tasks").
		WithContentType("application/json").
		WithBody([]byte(`{"deadline": "2XXX-05-17T11:12:42.085Z","goal": "do it!"}`)).
		Concurrently()

	assert.NoError(t, rc.ExpectResponseStatus(http.StatusBadRequest))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{
	 "status":"INVALID_ARGUMENT",
	 "error":"<ignore-diff>"
	}`)))

	rc.Reset().WithMethod(http.MethodGet).WithURI("/dev/tasks/1").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusOK))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"id":1,"goal":"string","deadline":"2020-05-17T11:12:42.085Z",`+
		`"createdAt":"<ignore-diff>"}`)))

	rc.Reset().WithMethod(http.MethodGet).WithURI("/dev/tasks").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusOK))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`[{"id":1,"goal":"string","deadline":"2020-05-17T11:12:42.085Z",`+
		`"createdAt":"<ignore-diff>"}]`)))

	rc.Reset().WithMethod(http.MethodPut).WithURI("/dev/tasks/1").
		WithContentType("application/json").
		WithBody([]byte(`{"deadline": "2020-05-17T11:12:42.085Z","goal": "foo"}`)).
		Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusNoContent))
	assert.NoError(t, rc.ExpectResponseBody(nil))

	rc.Reset().WithMethod(http.MethodGet).WithURI("/dev/tasks/1").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusOK))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"id":1,"goal":"foo","deadline":"2020-05-17T11:12:42.085Z",`+
		`"createdAt":"<ignore-diff>"}`)))

	rc.Reset().WithMethod(http.MethodDelete).WithURI("/dev/tasks/1").Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusNoContent))
	assert.NoError(t, rc.ExpectResponseBody(nil))

	assert.NoError(t, rc.ExpectOtherResponsesStatus(http.StatusBadRequest))
	assert.NoError(t, rc.ExpectOtherResponsesBody([]byte(`{"status":"FAILED_PRECONDITION",`+
		`"error":"failed precondition: task is already closed"}`)))

	rc.Reset().WithMethod(http.MethodPost).WithURI("/user/tasks").
		WithBody([]byte(`{"deadline": "2022-06-20T23:49:10.227Z","goal": "goal!"}`)).Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusUnauthorized))
	assert.NoError(t, rc.ExpectResponseBody(nil))

	rc.Reset().WithMethod(http.MethodPost).WithURI("/user/tasks").
		WithHeader("Authorization", "Basic dXNlcjp1c2Vy"). // user:user.
		WithBody([]byte(`{"deadline": "2022-06-20T23:49:10.227Z","goal": "goal!"}`)).Concurrently()
	assert.NoError(t, rc.ExpectResponseStatus(http.StatusCreated))
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"id":"<ignore-diff>","goal":"goal!","deadline":"2022-06-20T23:49:10.227Z","createdAt":"<ignore-diff>"}`)))
}
