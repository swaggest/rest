package request // nolint:testpackage

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest"
	"github.com/valyala/fasthttp"
)

func Test_decodeJSONBody(t *testing.T) {
	createBody := []byte(`{"amount": 123,"customerId": "248df4b7-aa70-47b8-a036-33ac447e668d","type": "withdraw"}`)
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("/US/order/348df4b7-aa70-47b8-a036-33ac447e668d")
	rc.Request.SetBody(createBody)

	type Input struct {
		Amount     int    `json:"amount"`
		CustomerID string `json:"customerId"`
		Type       string `json:"type"`
	}

	i := Input{}
	assert.NoError(t, decodeJSONBody(readJSON)(&rc, &i, nil))
	assert.Equal(t, 123, i.Amount)
	assert.Equal(t, "248df4b7-aa70-47b8-a036-33ac447e668d", i.CustomerID)
	assert.Equal(t, "withdraw", i.Type)

	vl := rest.ValidatorFunc(func(in rest.ParamIn, namedData map[string]interface{}) error {
		return nil
	})

	i = Input{}
	assert.NoError(t, decodeJSONBody(readJSON)(&rc, &i, vl))
	assert.Equal(t, 123, i.Amount)
	assert.Equal(t, "248df4b7-aa70-47b8-a036-33ac447e668d", i.CustomerID)
	assert.Equal(t, "withdraw", i.Type)
}

func Test_decodeJSONBody_emptyBody(t *testing.T) {
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("any")

	var i []int

	err := decodeJSONBody(readJSON)(&rc, &i, nil)
	assert.EqualError(t, err, "missing request body to decode json")
}

func Test_decodeJSONBody_badContentType(t *testing.T) {
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("any")
	rc.Request.SetBody([]byte("123"))
	rc.Request.Header.Set("Content-Type", "text/plain")

	var i []int

	err := decodeJSONBody(readJSON)(&rc, &i, nil)
	assert.EqualError(t, err, "request with application/json content type expected, received: text/plain")
}

func Test_decodeJSONBody_decodeFailed(t *testing.T) {
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("any")
	rc.Request.SetBody([]byte("abc"))

	var i []int

	err := decodeJSONBody(readJSON)(&rc, &i, nil)
	assert.Error(t, err)
}

func Test_decodeJSONBody_unmarshalFailed(t *testing.T) {
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("any")
	rc.Request.SetBody([]byte("123"))

	var i []int

	err := decodeJSONBody(readJSON)(&rc, &i, nil)
	assert.EqualError(t, err, "failed to decode json: json: cannot unmarshal number into Go value of type []int")
}

func Test_decodeJSONBody_validateFailed(t *testing.T) {
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(http.MethodPost)
	rc.Request.SetRequestURI("any")
	rc.Request.SetBody([]byte("[123]"))

	var i []int

	vl := rest.ValidatorFunc(func(in rest.ParamIn, namedData map[string]interface{}) error {
		return errors.New("failed")
	})

	err := decodeJSONBody(readJSON)(&rc, &i, vl)
	assert.EqualError(t, err, "failed")
}
