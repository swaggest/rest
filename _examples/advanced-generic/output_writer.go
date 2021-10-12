//go:build go1.18
// +build go1.18

package main

import (
	"context"
	"encoding/csv"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func outputCSVWriter() usecase.Interactor {
	type writerOutput struct {
		Header string `header:"X-Header" description:"Sample response header."`
		usecase.OutputWithEmbeddedWriter
	}

	u := usecase.NewInteractor(func(ctx context.Context, _ interface{}, out *writerOutput) (err error) {
		out.Header = "abc"

		c := csv.NewWriter(out)
		return c.WriteAll([][]string{{"abc", "def", "hij"}, {"klm", "nop", "qrs"}})
	})

	u.SetTitle("Output With Stream Writer")
	u.SetDescription("Output with stream writer.")
	u.SetExpectedErrors(status.Internal)

	return u
}
