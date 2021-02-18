package nethttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest/_examples/task-api/internal/infra"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/nethttp"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/service"
	"github.com/swaggest/rest/resttest"
)

func Test_taskLifeSpan(t *testing.T) {
	l := infra.NewServiceLocator(service.Config{})
	defer l.Close()

	srv := httptest.NewServer(nethttp.NewRouter(l))
	defer srv.Close()

	rc := resttest.NewClient(srv.URL)

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
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"status":"INVALID_ARGUMENT",`+
		`"error":"invalid argument: failed to decode json: `+
		`parsing time \"\"2020-35-17T11:12:42.085Z\"\": month out of range"}`)))

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
	assert.NoError(t, rc.ExpectResponseBody([]byte(`{"status":"INVALID_ARGUMENT",`+
		`"error":"invalid argument: failed to decode json: parsing time \"\"2XXX-05-17T11:12:42.085Z\"\" `+
		`as \"\"2006-01-02T15:04:05Z07:00\"\": cannot parse \"-05-17T11:12:42.085Z\"\" as \"2006\""}`)))

	rc.Reset().WithMethod(http.MethodGet).WithPath("/dev/tasks/1").Concurrently()
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
}
