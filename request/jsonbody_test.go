package request //nolint:testpackage

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest"
)

func Test_decodeJSONBody(t *testing.T) {
	createBody := bytes.NewReader(
		[]byte(`{"amount": 123,"customerId": "248df4b7-aa70-47b8-a036-33ac447e668d","type": "withdraw"}`))
	createReq, err := http.NewRequest(http.MethodPost, "/US/order/348df4b7-aa70-47b8-a036-33ac447e668d", createBody)
	assert.NoError(t, err)

	type Input struct {
		Amount     int    `json:"amount"`
		CustomerID string `json:"customerId"`
		Type       string `json:"type"`
	}

	i := Input{}
	assert.NoError(t, decodeJSONBody(readJSON, false)(createReq, &i, nil))
	assert.Equal(t, 123, i.Amount)
	assert.Equal(t, "248df4b7-aa70-47b8-a036-33ac447e668d", i.CustomerID)
	assert.Equal(t, "withdraw", i.Type)

	vl := rest.ValidatorFunc(func(in rest.ParamIn, namedData map[string]interface{}) error {
		return nil
	})

	i = Input{}
	_, err = createBody.Seek(0, io.SeekStart)
	assert.NoError(t, err)
	assert.NoError(t, decodeJSONBody(readJSON, false)(createReq, &i, vl))
	assert.Equal(t, 123, i.Amount)
	assert.Equal(t, "248df4b7-aa70-47b8-a036-33ac447e668d", i.CustomerID)
	assert.Equal(t, "withdraw", i.Type)
}

func Test_decodeJSONBody_emptyBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "any", nil)
	require.NoError(t, err)

	var i []int

	err = decodeJSONBody(readJSON, false)(req, &i, nil)
	assert.EqualError(t, err, "missing request body")
}

func Test_decodeJSONBody_badContentType(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "any", bytes.NewBufferString("123"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	var i []int

	err = decodeJSONBody(readJSON, false)(req, &i, nil)
	assert.EqualError(t, err, "request with application/json content type expected, received: text/plain")
}

func Test_decodeJSONBody_decodeFailed(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "any", bytes.NewBufferString("abc"))
	require.NoError(t, err)

	var i []int

	err = decodeJSONBody(readJSON, false)(req, &i, nil)
	assert.Error(t, err)
}

func Test_decodeJSONBody_unmarshalFailed(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "any", bytes.NewBufferString("123"))
	require.NoError(t, err)

	var i []int

	err = decodeJSONBody(readJSON, false)(req, &i, nil)
	assert.EqualError(t, err, "failed to decode json: json: cannot unmarshal number into Go value of type []int")
}

func Test_decodeJSONBody_validateFailed(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "any", bytes.NewBufferString("[123]"))
	require.NoError(t, err)

	var i []int

	vl := rest.ValidatorFunc(func(in rest.ParamIn, namedData map[string]interface{}) error {
		return errors.New("failed")
	})

	err = decodeJSONBody(readJSON, false)(req, &i, vl)
	assert.EqualError(t, err, "failed")
}

func Test_decodeJSONBody_tolerateFormData(t *testing.T) {
	createBody := bytes.NewReader(
		[]byte(`amount=123&customerId=248df4b7-aa70-47b8-a036-33ac447e668d&type=withdraw`))
	createReq, err := http.NewRequest(http.MethodPost, "/US/order/348df4b7-aa70-47b8-a036-33ac447e668d", createBody)
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	assert.NoError(t, err)

	type Input struct {
		Amount     int    `json:"amount" formData:"amount"`
		CustomerID string `json:"customerId" formData:"customerId"`
		Type       string `json:"type" formData:"type"`
	}

	i := Input{}
	assert.NoError(t, decodeJSONBody(readJSON, true)(createReq, &i, nil))
	assert.Empty(t, i.Amount)
	assert.Empty(t, i.CustomerID)
	assert.Empty(t, i.Type)
}
