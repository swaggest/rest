package main

import (
	"context"
	"errors"

	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

type customErr struct {
	Message string                 `json:"msg"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func errorResponse() usecase.Interactor {
	type errType struct {
		Type string `query:"type" enum:"ok,invalid_argument,conflict" required:"true"`
	}

	type okResp struct {
		Status string `json:"status"`
	}

	u := usecase.NewIOI(new(errType), new(okResp), func(ctx context.Context, input, output interface{}) (err error) {
		var (
			in  = input.(*errType)
			out = output.(*okResp)
		)

		switch in.Type {
		case "ok":
			out.Status = "ok"
		case "invalid_argument":
			return status.Wrap(errors.New("bad value for foo"), status.InvalidArgument)
		case "conflict":
			return status.Wrap(ctxd.NewError(ctx, "conflict", "foo", "bar"),
				status.AlreadyExists)
		}

		return nil
	})

	u.SetTitle("Declare Expected Errors")
	u.SetDescription("This use case demonstrates documentation of expected errors.")
	u.SetExpectedErrors(status.InvalidArgument, anotherErr{}, status.FailedPrecondition, status.AlreadyExists)

	return u
}

// anotherErr is another custom error.
type anotherErr struct {
	Foo int `json:"foo"`
}

func (anotherErr) Error() string {
	return "foo happened"
}
