package main

import (
	"context"

	"github.com/swaggest/usecase"
)

type formOrJSONInput struct {
	Field1 string `json:"field1" formData:"field1" required:"true"`
	Field2 int    `json:"field2" formData:"field2" required:"true"`
	Field3 string `path:"path" required:"true"`
}

func (formOrJSONInput) ForceJSONRequestBody() {}

func formOrJSON() usecase.Interactor {
	type formOrJSONOutput struct {
		F1 string `json:"f1"`
		F2 int    `json:"f2"`
		F3 string `json:"f3"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input formOrJSONInput, output *formOrJSONOutput) error {
		output.F1 = input.Field1
		output.F2 = input.Field2
		output.F3 = input.Field3

		return nil
	})

	u.SetTags("Request")

	return u
}
