package resttest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/assertjson/json5"
)

// Client keeps state of expectations.
type Client struct {
	ConcurrencyLevel int
	JSONComparer     assertjson.Comparer

	baseURL string
	Headers map[string]string

	resp     *http.Response
	respBody []byte

	reqHeaders map[string]string
	reqBody    []byte
	reqMethod  string
	reqURI     string

	// reqConcurrency is a number of simultaneous requests to send.
	reqConcurrency int

	otherRespBody []byte
	otherResp     *http.Response
}

var (
	errEmptyBody                = errors.New("received empty body")
	errResponseCardinality      = errors.New("response status cardinality too high")
	errUnexpectedBody           = errors.New("unexpected body")
	errUnexpectedResponseStatus = errors.New("unexpected response status")
	errUnexpectedResponseHeader = errors.New("unexpected response header")
	errOperationNotIdempotent   = errors.New("operation is not idempotent")
)

const defaultConcurrencyLevel = 10

// NewClient creates client instance.
func NewClient(baseURL string) *Client {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	return (&Client{
		baseURL:      baseURL,
		JSONComparer: assertjson.Comparer{IgnoreDiff: assertjson.IgnoreDiff},
	}).Reset()
}

// Reset deletes client state.
func (c *Client) Reset() *Client {
	c.reqHeaders = map[string]string{}

	c.resp = nil
	c.respBody = nil

	c.reqMethod = ""
	c.reqURI = ""
	c.reqBody = nil

	c.reqConcurrency = 0
	c.otherResp = nil
	c.otherRespBody = nil

	return c
}

// WithMethod sets request HTTP method.
func (c *Client) WithMethod(method string) *Client {
	c.reqMethod = method

	return c
}

// WithPath sets request URI path.
//
// Deprecated: use WithURI.
func (c *Client) WithPath(path string) *Client {
	c.reqURI = path

	return c
}

// WithURI sets request URI.
func (c *Client) WithURI(uri string) *Client {
	c.reqURI = uri

	return c
}

// WithBody sets request body.
func (c *Client) WithBody(body []byte) *Client {
	c.reqBody = body

	return c
}

// WithContentType sets request content type.
func (c *Client) WithContentType(contentType string) *Client {
	c.reqHeaders["Content-Type"] = contentType

	return c
}

// WithHeader sets request header.
func (c *Client) WithHeader(key, value string) *Client {
	c.reqHeaders[http.CanonicalHeaderKey(key)] = value

	return c
}

func (c *Client) do() (err error) {
	if c.reqConcurrency < 1 {
		c.reqConcurrency = 1
	}

	// A map of responses count by status code.
	statusCodeCount := make(map[int]int, 2)
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	resps := make(map[int]*http.Response, 2)
	bodies := make(map[int][]byte, 2)

	for i := 0; i < c.reqConcurrency; i++ {
		wg.Add(1)

		go func() {
			var er error

			defer func() {
				if er != nil {
					mu.Lock()
					err = er
					mu.Unlock()
				}

				wg.Done()
			}()

			resp, er := c.doOnce()
			if er != nil {
				return
			}

			body, er := ioutil.ReadAll(resp.Body)
			if er != nil {
				return
			}

			er = resp.Body.Close()
			if er != nil {
				return
			}

			mu.Lock()
			if _, ok := statusCodeCount[resp.StatusCode]; !ok {
				resps[resp.StatusCode] = resp
				bodies[resp.StatusCode] = body
				statusCodeCount[resp.StatusCode] = 1
			} else {
				statusCodeCount[resp.StatusCode]++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if err != nil {
		return err
	}

	return c.checkResponses(statusCodeCount, bodies, resps)
}

// CheckResponses checks if responses qualify idempotence criteria.
//
// Operation is considered idempotent in one of two cases:
//  * all responses have same status code (e.g. GET /resource: all 200 OK),
//  * all responses but one have same status code (e.g. POST /resource: one 200 OK, many 409 Conflict).
//
// Any other case is considered an idempotence violation.
func (c *Client) checkResponses(
	statusCodeCount map[int]int,
	bodies map[int][]byte,
	resps map[int]*http.Response,
) error {
	var (
		statusCode      int
		otherStatusCode int
	)

	switch {
	case len(statusCodeCount) == 1:
		for code := range statusCodeCount {
			statusCode = code

			break
		}
	case len(statusCodeCount) > 1:
		for code, cnt := range statusCodeCount {
			if cnt == 1 {
				statusCode = code
			} else {
				otherStatusCode = code
			}
		}
	default:
		return fmt.Errorf("%w: %v", errResponseCardinality, statusCodeCount)
	}

	if statusCode == 0 {
		responses := ""
		for c, b := range bodies {
			responses += fmt.Sprintf("\nstatus %d with %d responses, sample body: %s",
				c, statusCodeCount[c], strings.Trim(string(b), "\n"))
		}

		return fmt.Errorf("%w: %v", errOperationNotIdempotent, responses)
	}

	c.resp = resps[statusCode]
	c.respBody = bodies[statusCode]

	if otherStatusCode != 0 {
		c.otherResp = resps[otherStatusCode]
		c.otherRespBody = bodies[otherStatusCode]
	}

	return nil
}

func (c *Client) doOnce() (*http.Response, error) {
	var reqBody io.Reader
	if len(c.reqBody) > 0 {
		reqBody = bytes.NewBuffer(c.reqBody)
	}

	req, err := http.NewRequestWithContext(context.Background(), c.reqMethod, c.baseURL+c.reqURI, reqBody)
	if err != nil {
		return nil, err
	}

	if len(c.Headers) > 0 {
		for k, v := range c.Headers {
			req.Header.Set(k, v)
		}
	}

	if len(c.reqHeaders) > 0 {
		for k, v := range c.reqHeaders {
			req.Header.Set(k, v)
		}
	}

	return http.DefaultTransport.RoundTrip(req)
}

// ExpectResponseStatus sets expected response status code.
func (c *Client) ExpectResponseStatus(statusCode int) error {
	if c.resp == nil {
		err := c.do()
		if err != nil {
			return err
		}
	}

	return c.assertResponseCode(statusCode, c.resp)
}

// ExpectResponseHeader asserts expected response header value.
func (c *Client) ExpectResponseHeader(key, value string) error {
	if c.resp == nil {
		err := c.do()
		if err != nil {
			return err
		}
	}

	return c.assertResponseHeader(key, value, c.resp)
}

// ExpectOtherResponsesStatus sets expectation for response status to be received one or more times during concurrent
// calling.
//
// For example, it may describe "Not Found" response on multiple DELETE or "Conflict" response on multiple POST.
// Does not affect single (non-concurrent) calls.
func (c *Client) ExpectOtherResponsesStatus(statusCode int) error {
	if c.resp == nil {
		err := c.do()
		if err != nil {
			return err
		}
	}

	return c.assertResponseCode(statusCode, c.otherResp)
}

// ExpectOtherResponsesHeader sets expectation for response header value to be received one or more times during
// concurrent calling.
func (c *Client) ExpectOtherResponsesHeader(key, value string) error {
	if c.resp == nil {
		err := c.do()
		if err != nil {
			return err
		}
	}

	return c.assertResponseHeader(key, value, c.otherResp)
}

func (c *Client) assertResponseCode(statusCode int, resp *http.Response) error {
	if resp.StatusCode != statusCode {
		return fmt.Errorf("%w, expected: %d (%s), received: %d (%s)", errUnexpectedResponseStatus,
			statusCode, http.StatusText(statusCode), resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}

func (c *Client) assertResponseHeader(key, value string, resp *http.Response) error {
	expected, err := json.Marshal(value)
	if err != nil {
		return err
	}

	received, err := json.Marshal(resp.Header.Get(key))
	if err != nil {
		return err
	}

	if err := c.JSONComparer.FailNotEqual(expected, received); err != nil {
		return err
	}

	return nil
}

// ExpectResponseBody sets expectation for response body to be received.
//
// In concurrent mode such response mush be met only once or for all calls.
func (c *Client) ExpectResponseBody(body []byte) error {
	if c.resp == nil {
		err := c.do()
		if err != nil {
			return err
		}
	}

	return c.checkBody(body, c.respBody)
}

// ExpectOtherResponsesBody sets expectation for response body to be received one or more times during concurrent
// calling.
//
// For example, it may describe "Not Found" response on multiple DELETE or "Conflict" response on multiple POST.
// Does not affect single (non-concurrent) calls.
func (c *Client) ExpectOtherResponsesBody(body []byte) error {
	if c.resp == nil {
		err := c.do()
		if err != nil {
			return err
		}
	}

	return c.checkBody(body, c.otherRespBody)
}

func (c *Client) checkBody(expected, received []byte) error {
	if len(received) == 0 {
		if len(expected) == 0 {
			return nil
		}

		return errEmptyBody
	}

	if json5.Valid(expected) && json5.Valid(received) {
		expected, err := json5.Downgrade(expected)
		if err != nil {
			return err
		}

		err = c.JSONComparer.FailNotEqual(expected, received)
		if err != nil {
			recCompact, cerr := assertjson.MarshalIndentCompact(json.RawMessage(received), "", " ", 100)
			if cerr == nil {
				received = recCompact
			}

			return fmt.Errorf("%w\nreceived:\n%s ", err, string(received))
		}

		return nil
	}

	if !bytes.Equal(expected, received) {
		return fmt.Errorf("%w, expected: %s, received: %s",
			errUnexpectedBody, string(expected), string(received))
	}

	return nil
}

// Concurrently enables concurrent calls to idempotent endpoint.
func (c *Client) Concurrently() *Client {
	c.reqConcurrency = c.ConcurrencyLevel
	if c.reqConcurrency == 0 {
		c.reqConcurrency = defaultConcurrencyLevel
	}

	return c
}
