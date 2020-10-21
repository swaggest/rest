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
)

// Client keeps state of expectations.
type Client struct {
	baseURL          string
	Headers          map[string]string
	ConcurrencyLevel int

	resp     *http.Response
	respBody []byte

	reqHeaders map[string]string
	reqBody    []byte
	reqMethod  string
	reqPath    string

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
	errOperationNotIdempotent   = errors.New("operation is not idempotent")
)

const defaultConcurrencyLevel = 10

// NewClient creates client instance.
func NewClient(baseURL string) *Client {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	return (&Client{
		baseURL: baseURL,
	}).Reset()
}

// Reset deletes client state.
func (c *Client) Reset() *Client {
	c.reqHeaders = map[string]string{}

	c.resp = nil
	c.respBody = nil

	c.reqMethod = ""
	c.reqPath = ""
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

// WithPath sets request URL path.
func (c *Client) WithPath(path string) *Client {
	c.reqPath = path

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
	c.reqHeaders[key] = value

	return c
}

func (c *Client) do() (err error) {
	if c.reqConcurrency < 1 {
		c.reqConcurrency = 1
	}

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	resps := make(map[int]*http.Response, 2)
	bodies := make(map[int][]byte, 2)

	// A map of responses count by status code.
	statusCodeCount := make(map[int]int, 2)

	errs := make([]error, 0)

	for i := 0; i < c.reqConcurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			resp, err := c.doOnce()
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()

				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}

			err = resp.Body.Close()
			if err != nil {
				panic(err)
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

	req, err := http.NewRequestWithContext(context.Background(), c.reqMethod, c.baseURL+c.reqPath, reqBody)
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

func (c *Client) assertResponseCode(statusCode int, resp *http.Response) error {
	if resp.StatusCode != statusCode {
		return fmt.Errorf("%w, expected: %d (%s), received: %d (%s)", errUnexpectedResponseStatus,
			statusCode, http.StatusText(statusCode), resp.StatusCode, http.StatusText(resp.StatusCode))
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

	if json.Valid(expected) {
		err := assertjson.FailNotEqual(expected, received)
		if err != nil {
			return fmt.Errorf("%w\nreceived: %s ", err, string(received))
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
