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

	u := usecase.NewInteractor(func(ctx context.Context, in errType, out *okResp) (err error) {
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
	u.SetExpectedErrors(status.InvalidArgument, status.AlreadyExists)

	return u
}
