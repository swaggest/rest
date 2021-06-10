package resttest_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/bool64/shared"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest/resttest"
)

func TestNewClient(t *testing.T) {
	cnt := int64(0)
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/foo?q=1", r.URL.String())
		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, `{"foo":"bar"}`, string(b))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "abc", r.Header.Get("X-Header"))
		assert.Equal(t, "def", r.Header.Get("X-Custom"))

		c, err := r.Cookie("c1")
		assert.NoError(t, err)
		assert.Equal(t, "1", c.Value)

		c, err = r.Cookie("c2")
		assert.NoError(t, err)
		assert.Equal(t, "2", c.Value)

		ncnt := atomic.AddInt64(&cnt, 1)
		rw.Header().Set("Content-Type", "application/json")
		if ncnt > 1 {
			rw.WriteHeader(http.StatusConflict)
			_, err := rw.Write([]byte(`{"error":"conflict"}`))
			assert.NoError(t, err)
		} else {
			rw.WriteHeader(http.StatusAccepted)
			_, err := rw.Write([]byte(`{"bar":"foo", "dyn": "abc"}`))
			assert.NoError(t, err)
		}
	}))

	defer srv.Close()

	vars := &shared.Vars{}

	c := resttest.NewClient(srv.URL)
	c.JSONComparer.Vars = vars
	c.ConcurrencyLevel = 50
	c.Headers = map[string]string{
		"X-Header": "abc",
	}

	c.Reset().
		WithMethod(http.MethodPost).
		WithHeader("X-Custom", "def").
		WithContentType("application/json").
		WithBody([]byte(`{"foo":"bar"}`)).
		WithCookie("c1", "1").
		WithCookie("c2", "2").
		WithURI("/foo?q=1").
		Concurrently()

	assert.NoError(t, c.ExpectResponseStatus(http.StatusAccepted))
	assert.NoError(t, c.ExpectResponseBody([]byte(`{"bar":"foo","dyn":"$var1"}`)))
	assert.NoError(t, c.ExpectResponseHeader("Content-Type", "application/json"))
	assert.NoError(t, c.ExpectOtherResponsesStatus(http.StatusConflict))
	assert.NoError(t, c.ExpectOtherResponsesBody([]byte(`{"error":"conflict"}`)))
	assert.NoError(t, c.ExpectOtherResponsesHeader("Content-Type", "application/json"))
	assert.NoError(t, c.CheckUnexpectedOtherResponses())
	assert.EqualError(t, c.ExpectNoOtherResponses(), "unexpected response status, expected: 202 (Accepted), received: 409 (Conflict)")

	val, found := vars.Get("$var1")
	assert.True(t, found)
	assert.Equal(t, "abc", val)
}

func TestNewClient_failedExpectation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, err := writer.Write([]byte(`{"bar":"foo"}`))
		assert.NoError(t, err)
	}))
	defer srv.Close()
	c := resttest.NewClient(srv.URL)

	c.OnBodyMismatch = func(received []byte) {
		assert.Equal(t, `{"bar":"foo"}`, string(received))
		println(received)
	}

	c.WithURI("/")
	assert.EqualError(t, c.ExpectResponseBody([]byte(`{"foo":"bar}"`)),
		"unexpected body, expected: {\"foo\":\"bar}\", received: {\"bar\":\"foo\"}")
}
