package rest_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest-fasthttp"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func TestHTTPStatusFromCanonicalCode(t *testing.T) {
	maxStatusCode := 17
	for i := 0; i <= maxStatusCode; i++ {
		s := status.Code(i)
		assert.NotEmpty(t, rest.HTTPStatusFromCanonicalCode(s))
	}
}

type errWithHTTPStatus int

func (e errWithHTTPStatus) Error() string {
	return "failed very much"
}

func (e errWithHTTPStatus) HTTPStatus() int {
	return int(e)
}

func TestErr(t *testing.T) {
	err := usecase.Error{
		StatusCode: status.InvalidArgument,
		Value:      errors.New("failed"),
		Context:    map[string]interface{}{"hello": "world"},
	}
	code, er := rest.Err(err)

	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, map[string]interface{}{"hello": "world"}, er.Context)
	assert.Equal(t, "invalid argument: failed", er.Error())
	assert.Equal(t, "INVALID_ARGUMENT", er.StatusText)
	assert.Equal(t, 0, er.AppCode)
	assert.Equal(t, err, er.Unwrap())

	j, jErr := json.Marshal(er)
	assert.NoError(t, jErr)
	assert.Equal(t,
		`{"status":"INVALID_ARGUMENT","error":"invalid argument: failed","context":{"hello":"world"}}`,
		string(j),
	)

	code, er = rest.Err(status.DataLoss)

	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Nil(t, er.Context)
	assert.Equal(t, "data loss", er.Error())
	assert.Equal(t, "DATA_LOSS", er.StatusText)
	assert.Equal(t, 0, er.AppCode)
	assert.Equal(t, status.DataLoss, er.Unwrap())

	err = usecase.Error{
		AppCode:    123,
		StatusCode: status.InvalidArgument,
		Value:      errors.New("failed"),
	}
	code, er = rest.Err(err)

	assert.Nil(t, er.Context)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "invalid argument: failed", er.Error())
	assert.Equal(t, "INVALID_ARGUMENT", er.StatusText)
	assert.Equal(t, 123, er.AppCode)
	assert.Equal(t, err, er.Unwrap())

	code, er = rest.Err(errWithHTTPStatus(http.StatusTeapot))

	assert.Nil(t, er.Context)
	assert.Equal(t, http.StatusTeapot, code)
	assert.Equal(t, "failed very much", er.Error())
	assert.Equal(t, "", er.StatusText)
	assert.Equal(t, 0, er.AppCode)

	assert.Panics(t, func() {
		_, er := rest.Err(nil)
		assert.NoError(t, er)
	})
}
