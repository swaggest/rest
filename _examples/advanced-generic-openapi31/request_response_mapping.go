//go:build go1.18

package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func reqRespMapping() usecase.Interactor {
	type inputPort struct {
		Val1 string `description:"Simple scalar value with sample validation." required:"true" minLength:"3"`
		Val2 int    `description:"Simple scalar value with sample validation." required:"true" minimum:"3"`
	}

	type outputPort struct {
		Val1 string `json:"-" description:"Simple scalar value with sample validation." required:"true" minLength:"3"`
		Val2 int    `json:"-" description:"Simple scalar value with sample validation." required:"true" minimum:"3"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in inputPort, out *outputPort) (err error) {
		out.Val1 = in.Val1
		out.Val2 = in.Val2

		return nil
	})

	u.SetTitle("Request Response Mapping")
	u.SetName("reqRespMapping")
	u.SetDescription("This use case has transport concerns fully decoupled with external req/resp mapping.")
	u.SetTags("Request", "Response")

	return u
}
