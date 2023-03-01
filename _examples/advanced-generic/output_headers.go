//go:build go1.18

package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func outputHeaders() usecase.Interactor {
	type EmbeddedHeaders struct {
		Foo int `header:"X-Foo" json:"-"`
	}

	type headerOutput struct {
		EmbeddedHeaders
		Header string `header:"X-Header" json:"-" description:"Sample response header."`
		InBody string `json:"inBody" deprecated:"true"`
		Cookie int    `cookie:"coo,httponly,path:/foo" json:"-"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, _ struct{}, out *headerOutput) (err error) {
		out.Header = "abc"
		out.InBody = "def"
		out.Cookie = 123
		out.Foo = 321

		return nil
	})

	u.SetTitle("Output With Headers")
	u.SetDescription("Output with headers.")
	u.SetTags("Response")

	return u
}
