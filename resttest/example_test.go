package resttest_test

import (
	"fmt"
	"net/http"

	"github.com/swaggest/rest/resttest"
)

func ExampleNewClient() {
	// Prepare server mock.
	sm, url := resttest.NewServerMock()
	defer sm.Close()

	// This example shows Client and ServerMock working together for sake of portability.
	// In real-world scenarios Client would complement real server or ServerMock would complement real HTTP client.

	// Set successful expectation for first request out of concurrent batch.
	exp := resttest.Expectation{
		Method:     http.MethodPost,
		RequestURI: "/foo?q=1",
		RequestHeader: map[string]string{
			"X-Custom":     "def",
			"X-Header":     "abc",
			"Content-Type": "application/json",
		},
		RequestBody:  []byte(`{"foo":"bar"}`),
		Status:       http.StatusAccepted,
		ResponseBody: []byte(`{"bar":"foo"}`),
	}
	sm.Expect(exp)

	// Set failing expectation for other requests of concurrent batch.
	exp.Status = http.StatusConflict
	exp.ResponseBody = []byte(`{"error":"conflict"}`)
	exp.Unlimited = true
	sm.Expect(exp)

	// Prepare client request.
	c := resttest.NewClient(url)
	c.ConcurrencyLevel = 50
	c.Headers = map[string]string{
		"X-Header": "abc",
	}

	c.Reset().
		WithMethod(http.MethodPost).
		WithHeader("X-Custom", "def").
		WithContentType("application/json").
		WithBody([]byte(`{"foo":"bar"}`)).
		WithURI("/foo?q=1").
		Concurrently()

	// Check expectations errors.
	fmt.Println(
		c.ExpectResponseStatus(http.StatusAccepted),
		c.ExpectResponseBody([]byte(`{"bar":"foo"}`)),
		c.ExpectOtherResponsesStatus(http.StatusConflict),
		c.ExpectOtherResponsesBody([]byte(`{"error":"conflict"}`)),
	)

	// Output:
	// <nil> <nil> <nil> <nil>
}
