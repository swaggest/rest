//go:build go1.18

package main

import (
	"context"
	"encoding/json"

	"github.com/swaggest/usecase"
)

// JSONSlicePayload is an example non-scalar type without `json` tags.
type JSONSlicePayload []int

type jsonSliceReq struct {
	Header string `header:"X-Header" description:"Simple scalar value in header."`
	Query  int    `query:"in_query" description:"Simple scalar value in query."`
	JSONSlicePayload
}

func (j *jsonSliceReq) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.JSONSlicePayload)
}

func jsonSliceBody() usecase.Interactor {
	type jsonOutput struct {
		Header string           `json:"inHeader"`
		Query  int              `json:"inQuery"`
		Data   JSONSlicePayload `json:"data"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in jsonSliceReq, out *jsonOutput) (err error) {
		out.Query = in.Query
		out.Header = in.Header
		out.Data = in.JSONSlicePayload

		return nil
	})

	u.SetTitle("Request With JSON Array In Body")
	u.SetTags("Request")

	return u
}
