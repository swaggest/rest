//go:build go1.18
// +build go1.18

package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func outputHeaders() usecase.Interactor {
	type headerOutput struct {
		Header string `header:"X-Header" json:"-" description:"Sample response header."`
		InBody string `json:"inBody" deprecated:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, _ interface{}, out *headerOutput) (err error) {
		out.Header = "abc"
		out.InBody = "def"

		return nil
	})

	u.SetTitle("Output With Headers")
	u.SetDescription("Output with headers.")

	return u
}
