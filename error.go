package rest

import (
	"errors"
	"net/http"

	"github.com/swaggest/usecase/status"
)

// HTTPCodeAsError exposes HTTP status code as use case error that can be translated to response status.
type HTTPCodeAsError int

// Error return HTTP status text.
func (c HTTPCodeAsError) Error() string {
	return http.StatusText(int(c))
}

// HTTPStatus returns HTTP status code.
func (c HTTPCodeAsError) HTTPStatus() int {
	return int(c)
}

// ErrWithHTTPStatus exposes HTTP status code.
type ErrWithHTTPStatus interface {
	error
	HTTPStatus() int
}

// ErrWithFields exposes structured context of error.
type ErrWithFields interface {
	error
	Fields() map[string]interface{}
}

// ErrWithAppCode exposes application error code.
type ErrWithAppCode interface {
	error
	AppErrCode() int
}

// ErrWithCanonicalStatus exposes canonical status code.
type ErrWithCanonicalStatus interface {
	error
	Status() status.Code
}

// Err creates HTTP status code and ErrResponse for error.
//
// You can use it with use case status code:
//
//	rest.Err(status.NotFound)
func Err(err error) (int, ErrResponse) {
	if err == nil {
		panic("nil error received")
	}

	er := ErrResponse{}

	var (
		withHTTPStatus      ErrWithHTTPStatus
		withCanonicalStatus ErrWithCanonicalStatus
		withAppCode         ErrWithAppCode
		withFields          ErrWithFields
	)

	er.err = err
	er.ErrorText = err.Error()
	er.httpStatusCode = http.StatusInternalServerError

	if errors.As(err, &withCanonicalStatus) {
		us := withCanonicalStatus.Status()
		er.httpStatusCode = HTTPStatusFromCanonicalCode(us)
		er.StatusText = us.String()
	}

	if errors.As(err, &withHTTPStatus) {
		er.httpStatusCode = withHTTPStatus.HTTPStatus()
	}

	if errors.As(err, &withAppCode) {
		er.AppCode = withAppCode.AppErrCode()
	}

	if errors.As(err, &withFields) {
		er.Context = withFields.Fields()
	}

	if er.ErrorText == er.StatusText {
		er.ErrorText = ""
	}

	return er.httpStatusCode, er
}

// ErrResponse is HTTP error response body.
type ErrResponse struct {
	StatusText string                 `json:"status,omitempty" description:"Status text."`
	AppCode    int                    `json:"code,omitempty" description:"Application-specific error code."`
	ErrorText  string                 `json:"error,omitempty" description:"Error message."`
	Context    map[string]interface{} `json:"context,omitempty" description:"Application context."`

	err            error // Original error.
	httpStatusCode int   // HTTP response status code.
}

// Error implements error.
func (e ErrResponse) Error() string {
	if e.ErrorText != "" {
		return e.ErrorText
	}

	return e.StatusText
}

// Unwrap returns parent error.
func (e ErrResponse) Unwrap() error {
	return e.err
}

// HTTPStatusFromCanonicalCode returns http status accordingly to use case status code.
func HTTPStatusFromCanonicalCode(c status.Code) int {
	switch c {
	case status.OK:
		return http.StatusOK
	case status.Canceled:
		// Custom nginx status "499 Client Closed Request" is a recommended mapping, but 500 is more compatible.
		return http.StatusInternalServerError
	case status.Unknown:
		return http.StatusInternalServerError
	case status.InvalidArgument:
		return http.StatusBadRequest
	case status.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case status.NotFound:
		return http.StatusNotFound
	case status.AlreadyExists:
		return http.StatusConflict
	case status.PermissionDenied:
		return http.StatusForbidden
	case status.ResourceExhausted:
		return http.StatusTooManyRequests
	case status.FailedPrecondition:
		return http.StatusPreconditionFailed
	case status.Aborted:
		return http.StatusConflict
	case status.OutOfRange:
		return http.StatusBadRequest
	case status.Unimplemented:
		return http.StatusNotImplemented
	case status.Internal:
		return http.StatusInternalServerError
	case status.Unavailable:
		return http.StatusServiceUnavailable
	case status.DataLoss:
		return http.StatusInternalServerError
	case status.Unauthenticated:
		return http.StatusUnauthorized
	}

	return http.StatusInternalServerError
}
