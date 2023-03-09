//go:build go1.18

package main

import (
	"context"
	"encoding/csv"
	"net/http"

	"github.com/swaggest/rest"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func outputCSVWriter() usecase.Interactor {
	type writerOutput struct {
		Header      string `header:"X-Header" description:"Sample response header."`
		ContentHash string `header:"ETag" description:"Content hash."`
		usecase.OutputWithEmbeddedWriter
	}

	type writerInput struct {
		ContentHash string `header:"If-None-Match" description:"Content hash."`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in writerInput, out *writerOutput) (err error) {
		contentHash := "abc123" // Pretending this is an actual content hash.

		if in.ContentHash == contentHash {
			return rest.HTTPCodeAsError(http.StatusNotModified)
		}

		out.Header = "abc"
		out.ContentHash = contentHash

		c := csv.NewWriter(out)
		return c.WriteAll([][]string{{"abc", "def", "hij"}, {"klm", "nop", "qrs"}})
	})

	u.SetTitle("Output With Stream Writer")
	u.SetDescription("Output with stream writer.")
	u.SetExpectedErrors(status.Internal, rest.HTTPCodeAsError(http.StatusNotModified))
	u.SetTags("Response")

	return u
}
