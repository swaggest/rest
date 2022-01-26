package resttest

import (
	"github.com/bool64/httpmock"
)

// Client keeps state of expectations.
//
// Deprecated: please use httpmock.Client.
type Client = httpmock.Client

// NewClient creates client instance, baseURL may be empty if Client.SetBaseURL is used later.
//
// Deprecated: please use httpmock.NewClient.
func NewClient(baseURL string) *Client {
	return httpmock.NewClient(baseURL)
}
