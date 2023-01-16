package main

import (
	"context"
	"encoding/csv"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func outputCSVWriter() usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithOutput
	}{}

	u.SetTitle("Output With Stream Writer")
	u.SetDescription("Output with stream writer.")
	u.SetExpectedErrors(status.Internal)

	type writerOutput struct {
		Header string `header:"X-Header" description:"Sample response header."`
		usecase.OutputWithEmbeddedWriter
	}

	u.Output = new(writerOutput)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output any) (err error) {
		out := output.(*writerOutput)

		out.Header = "abc"

		c := csv.NewWriter(out)
		return c.WriteAll([][]string{{"abc", "def", "hij"}, {"klm", "nop", "qrs"}})
	})

	return u
}
