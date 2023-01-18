//go:build go1.18

package main

import (
	"context"

	"github.com/swaggest/usecase"
)

func form() usecase.Interactor {
	type form struct {
		ID   int    `form:"id"`
		Name string `form:"name"`
	}

	type output struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in form, out *output) error {
		out.ID = in.ID
		out.Name = in.Name

		return nil
	})

	u.SetTitle("Request With Form")
	u.SetDescription("The `form` field tag acts as `query` and `formData`, with priority on `formData`.\n\n" +
		"It is decoded with `http.Request.Form` values.")

	return u
}
