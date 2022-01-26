package resttest

import (
	"github.com/bool64/httpmock"
)

// Expectation describes expected request and defines response.
//
// Deprecated: please use httpmock.Expectation.
type Expectation = httpmock.Expectation

// ServerMock serves predefined response for predefined request.
type ServerMock = httpmock.Server

// NewServerMock creates mocked server.
//
// Deprecated: please use httpmock.NewServer.
func NewServerMock() (*ServerMock, string) {
	return httpmock.NewServer()
}
