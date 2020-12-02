package resttest

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/swaggest/assertjson"
	"github.com/swaggest/assertjson/json5"
)

// Expectation describes expected request and defines response.
type Expectation struct {
	Method        string
	RequestURI    string
	RequestHeader map[string]string
	RequestBody   []byte

	Status         int
	ResponseHeader map[string]string
	ResponseBody   []byte

	// Unlimited enables reusing of this expectation unlimited number of times.
	Unlimited bool
	// Repeated defines how many times this expectation should be used.
	Repeated int
}

// ServerMock serves predefined response for predefined request.
type ServerMock struct {
	// OnError is called on expectations mismatch or internal errors.
	OnError func(err error)

	// ErrorResponder allows custom failure responses.
	ErrorResponder func(rw http.ResponseWriter, err error)

	// DefaultResponseHeaders are added to every response to an expected request.
	DefaultResponseHeaders map[string]string

	mu           sync.Mutex
	server       *httptest.Server
	expectations []Expectation
}

// NewServerMock creates mocked server.
func NewServerMock() (*ServerMock, string) {
	m := ServerMock{}
	m.server = httptest.NewServer(&m)

	return &m, m.server.URL
}

// Expect adds expected operation.
func (c *ServerMock) Expect(e Expectation) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.expectations = append(c.expectations, e)
}

// Close closes mocked server.
func (c *ServerMock) Close() {
	c.server.Close()
}

// ServeHTTP asserts request expectations and serves mocked response.
func (c *ServerMock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.expectations) == 0 {
		body, err := ioutil.ReadAll(req.Body)
		if err == nil && len(body) > 0 {
			c.checkFail(rw, fmt.Errorf("unexpected request received: %s %s, body:\n%s", req.Method,
				req.RequestURI, string(body)))
		} else {
			c.checkFail(rw, fmt.Errorf("unexpected request received: %s %s", req.Method, req.RequestURI))
		}

		return
	}

	expectation := c.expectations[0]

	err := c.checkRequest(req, expectation)
	if c.checkFail(rw, err) {
		return
	}

	if c.DefaultResponseHeaders != nil {
		for k, v := range c.DefaultResponseHeaders {
			rw.Header().Set(k, v)
		}
	}

	if expectation.ResponseHeader != nil {
		for k, v := range expectation.ResponseHeader {
			rw.Header().Set(k, v)
		}
	}

	if expectation.Status == 0 {
		expectation.Status = http.StatusOK
	}

	rw.WriteHeader(expectation.Status)

	_, err = rw.Write(expectation.ResponseBody)
	if c.checkFail(rw, err) {
		return
	}

	if expectation.Unlimited {
		return
	}

	if expectation.Repeated > 0 {
		expectation.Repeated--
		c.expectations[0] = expectation

		if expectation.Repeated > 0 {
			return
		}
	}

	c.expectations = c.expectations[1:]
}

func (c *ServerMock) checkRequest(req *http.Request, expectation Expectation) error {
	if expectation.Method != "" && expectation.Method != req.Method {
		return fmt.Errorf("method %q expected, %q received", expectation.Method, req.Method)
	}

	if expectation.RequestURI != "" && expectation.RequestURI != req.RequestURI {
		return fmt.Errorf("request uri %q expected, %q received", expectation.RequestURI, req.RequestURI)
	}

	if len(expectation.RequestHeader) != 0 {
		for k, v := range expectation.RequestHeader {
			if req.Header.Get(k) != v {
				return fmt.Errorf("header %q with value %q expected, %q received", k, v, req.Header.Get(k))
			}
		}
	}

	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	if expectation.RequestBody == nil {
		return nil
	}

	if !json5.Valid(expectation.RequestBody) || !json5.Valid(reqBody) {
		if !bytes.Equal(expectation.RequestBody, reqBody) {
			return errors.New("unexpected request body")
		}
	} else {
		// Performing JSON comparison for JSON payloads and binary comparison otherwise.
		expectation.RequestBody, err = json5.Downgrade(expectation.RequestBody)
		if err != nil {
			return err
		}

		err := assertjson.FailNotEqual(expectation.RequestBody, reqBody)
		if err != nil {
			return fmt.Errorf("unexpected request body: %w", err)
		}
	}

	return nil
}

func (c *ServerMock) checkFail(rw http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	if c.OnError != nil {
		c.OnError(err)
	}

	if c.ErrorResponder != nil {
		c.ErrorResponder(rw, err)

		return true
	}

	rw.WriteHeader(http.StatusInternalServerError)

	_, err = rw.Write([]byte(err.Error()))
	if err != nil && c.OnError != nil {
		c.OnError(err)
	}

	return true
}

// ResetExpectations discards all expectation to reset the state of mock.
func (c *ServerMock) ResetExpectations() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.expectations = nil
}

// ExpectationsWereMet checks whether all queued expectations
// were met in order. If any of them was not met - an error is returned.
func (c *ServerMock) ExpectationsWereMet() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.expectations) != 0 {
		if len(c.expectations) == 1 && c.expectations[0].Unlimited {
			return nil
		}

		unmet := ""

		for _, e := range c.expectations {
			if e.Method != "" || e.RequestURI != "" {
				unmet += ", " + e.Method + " " + e.RequestURI
			} else {
				unmet += ", response " + string(e.ResponseBody)
			}
		}

		return errors.New("there are remaining expectations that were not met: " + unmet[2:])
	}

	return nil
}
