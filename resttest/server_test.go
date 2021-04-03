package resttest_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/bool64/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest/resttest"
)

func assertRoundTrip(t *testing.T, baseURL string, expectation resttest.Expectation) {
	t.Helper()

	var bodyReader io.Reader

	if expectation.RequestBody != nil {
		bodyReader = bytes.NewReader(expectation.RequestBody)
	}

	req, err := http.NewRequest(expectation.Method, baseURL+expectation.RequestURI, bodyReader)
	require.NoError(t, err)

	for k, v := range expectation.RequestHeader {
		req.Header.Set(k, v)
	}

	for n, v := range expectation.RequestCookie {
		req.AddCookie(&http.Cookie{Name: n, Value: v})
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

	if expectation.Status == 0 {
		expectation.Status = http.StatusOK
	}

	assert.Equal(t, expectation.Status, resp.StatusCode)
	assert.Equal(t, string(expectation.ResponseBody), string(body))

	// Asserting default for successful responses.
	if resp.StatusCode != http.StatusInternalServerError {
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	}

	if len(expectation.ResponseHeader) > 0 {
		for k, v := range expectation.ResponseHeader {
			assert.Equal(t, v, resp.Header.Get(k))
		}
	}
}

func TestServerMock_ServeHTTP(t *testing.T) {
	// Creating REST service mock.
	mock, baseURL := resttest.NewServerMock()
	defer mock.Close()

	mock.DefaultResponseHeaders = map[string]string{
		"Content-Type": "application/json",
	}

	// Requesting mock without expectations fails.
	assertRoundTrip(t, baseURL, resttest.Expectation{
		RequestURI:   "/test?test=test",
		Status:       http.StatusInternalServerError,
		ResponseBody: []byte("unexpected request received: GET /test?test=test"),
	})

	// Setting expectations for first request.
	exp1 := resttest.Expectation{
		Method:        http.MethodPost,
		RequestURI:    "/test?test=test",
		RequestHeader: map[string]string{"Authorization": "Bearer token"},
		RequestCookie: map[string]string{"c1": "v1", "c2": "v2"},
		RequestBody:   []byte(`{"request":"body"}`),

		Status:       http.StatusCreated,
		ResponseBody: []byte(`{"response":"body"}`),
	}
	mock.Expect(exp1)

	// Setting expectations for second request.
	exp2 := resttest.Expectation{
		Method:      http.MethodPost,
		RequestURI:  "/test?test=test",
		RequestBody: []byte(`not a JSON`),

		ResponseHeader: map[string]string{
			"X-Foo": "bar",
		},
		ResponseBody: []byte(`{"response":"body2"}`),
	}
	mock.Expect(exp2)

	// Sending first request.
	assertRoundTrip(t, baseURL, exp1)

	// Expectations were not met yet.
	assert.EqualError(t, mock.ExpectationsWereMet(),
		"there are remaining expectations that were not met: POST /test?test=test")

	// Sending second request.
	assertRoundTrip(t, baseURL, exp2)

	// Expectations were met.
	assert.NoError(t, mock.ExpectationsWereMet())

	// Requesting mock without expectations fails.
	assertRoundTrip(t, baseURL, resttest.Expectation{
		RequestURI:   "/test?test=test",
		Status:       http.StatusInternalServerError,
		ResponseBody: []byte("unexpected request received: GET /test?test=test"),
	})
}

func TestServerMock_ServeHTTP_error(t *testing.T) {
	// Creating REST service mock.
	mock, baseURL := resttest.NewServerMock()
	defer mock.Close()

	// Setting expectations for first request.
	mock.Expect(resttest.Expectation{
		Method:        http.MethodPost,
		RequestURI:    "/test?test=test",
		RequestHeader: map[string]string{"X-Foo": "bar"},
	})

	// Sending request with wrong uri.
	req, err := http.NewRequest(http.MethodPost, baseURL+"/wrong-uri", bytes.NewReader([]byte(`{"request":"body"}`)))
	require.NoError(t, err)
	req.Header.Set("X-Foo", "bar")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, `request uri "/test?test=test" expected, "/wrong-uri" received`, string(respBody))

	// Sending request with wrong method.
	req, err = http.NewRequest(http.MethodGet, baseURL+"/test?test=test", bytes.NewReader([]byte(`{"request":"body"}`)))
	require.NoError(t, err)
	req.Header.Set("X-Foo", "bar")

	resp, err = http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, `method "POST" expected, "GET" received`, string(respBody))

	// Sending request with wrong header.
	req, err = http.NewRequest(http.MethodPost, baseURL+"/test?test=test", bytes.NewReader([]byte(`{"request":"body"}`)))
	require.NoError(t, err)
	req.Header.Set("X-Foo", "space")

	resp, err = http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	respBody, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, `header "X-Foo" with value "bar" expected, "space" received`, string(respBody))
}

func TestServerMock_ServeHTTP_concurrency(t *testing.T) {
	// Creating REST service mock.
	mock, url := resttest.NewServerMock()
	defer mock.Close()

	n := 50

	for i := 0; i < n; i++ {
		// Setting expectations for first request.
		mock.Expect(resttest.Expectation{
			Method:       http.MethodGet,
			RequestURI:   "/test?test=test",
			ResponseBody: []byte("body"),
		})
	}

	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			// Sending request with wrong header.
			req, err := http.NewRequest(http.MethodGet, url+"/test?test=test", nil)
			require.NoError(t, err)
			req.Header.Set("X-Foo", "space")

			resp, err := http.DefaultTransport.RoundTrip(req)
			require.NoError(t, err)

			respBody, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, resp.Body.Close())
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, `body`, string(respBody))
		}()
	}

	wg.Wait()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerMock_ResetExpectations(t *testing.T) {
	// Creating REST service mock.
	mock, _ := resttest.NewServerMock()
	defer mock.Close()

	mock.Expect(resttest.Expectation{
		Method:       http.MethodGet,
		RequestURI:   "/test?test=test",
		ResponseBody: []byte("body"),
	})

	assert.Error(t, mock.ExpectationsWereMet())
	mock.ResetExpectations()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerMock_vars(t *testing.T) {
	sm, url := resttest.NewServerMock()
	sm.JSONComparer.Vars = &shared.Vars{}
	sm.Expect(resttest.Expectation{
		Method:       http.MethodGet,
		RequestURI:   "/",
		RequestBody:  []byte(`{"foo":"bar","dyn":"$var1"}`),
		ResponseBody: []byte(`{"bar":"foo","dynEcho":"$var1"}`),
	})

	req, err := http.NewRequest(http.MethodGet, url+"/", strings.NewReader(`{"foo":"bar","dyn":"abc"}`))
	require.NoError(t, err)

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	require.NoError(t, resp.Body.Close())

	assert.Equal(t, `{"bar":"foo","dynEcho":"abc"}`, string(body))
}
