package main

import (
	"context"
	"io"
	"net/http"

	"github.com/swaggest/usecase"
)

type textReqBodyInput struct {
	Path  string `path:"path"`
	Query int    `query:"query"`
	text  []byte
	err   error
}

func (c *textReqBodyInput) SetRequest(r *http.Request) {
	c.text, c.err = io.ReadAll(r.Body)
	clErr := r.Body.Close()

	if c.err == nil {
		c.err = clErr
	}
}

func textReqBody() usecase.Interactor {
	type output struct {
		Path  string `json:"path"`
		Query int    `json:"query"`
		Text  string `json:"text"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in textReqBodyInput, out *output) (err error) {
		out.Text = string(in.text)
		out.Path = in.Path
		out.Query = in.Query

		return nil
	})

	u.SetTitle("Request With Text Body")
	u.SetDescription("This usecase allows direct access to original `*http.Request` while keeping automated decoding of parameters.")
	u.SetTags("Request")

	return u
}

func textReqBodyPtr() usecase.Interactor {
	type output struct {
		Path  string `json:"path"`
		Query int    `json:"query"`
		Text  string `json:"text"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in *textReqBodyInput, out *output) (err error) {
		out.Text = string(in.text)
		out.Path = in.Path
		out.Query = in.Query

		return nil
	})

	u.SetTitle("Request With Text Body (ptr input)")
	u.SetDescription("This usecase allows direct access to original `*http.Request` while keeping automated decoding of parameters.")
	u.SetTags("Request")

	return u
}
