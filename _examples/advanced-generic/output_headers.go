//go:build go1.18

package main

import (
	"context"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func outputHeaders() usecase.Interactor {
	type EmbeddedHeaders struct {
		Foo int `header:"X-foO,omitempty" json:"-" minimum:"10" required:"true" description:"Reduced by 20 in response."`
	}

	type headerOutput struct {
		EmbeddedHeaders
		Header    string `header:"x-HeAdEr" json:"-" description:"Sample response header."`
		OmitEmpty int    `header:"x-omit-empty,omitempty" json:"-" description:"Receives req value of X-Foo reduced by 30."`
		InBody    string `json:"inBody" deprecated:"true"`
		Cookie    int    `cookie:"coo,httponly,path:/foo" json:"-"`
	}

	type headerInput struct {
		EmbeddedHeaders
	}

	u := usecase.NewInteractor(func(ctx context.Context, in headerInput, out *headerOutput) (err error) {
		out.Header = "abc"
		out.InBody = "def"
		out.Cookie = 123
		out.Foo = in.Foo - 20
		out.OmitEmpty = in.Foo - 30

		return nil
	})

	u.SetTitle("Output With Headers")
	u.SetDescription("Output with headers.")
	u.SetTags("Response")
	u.SetExpectedErrors(status.Internal)

	return u
}
