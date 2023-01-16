package main

import (
	"context"
	"encoding/json"

	"github.com/swaggest/usecase"
)

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

	u := usecase.NewIOI(new(jsonSliceReq), new(jsonOutput), func(ctx context.Context, input, output any) (err error) {
		var (
			in  = input.(*jsonSliceReq)
			out = output.(*jsonOutput)
		)

		out.Query = in.Query
		out.Header = in.Header
		out.Data = in.JSONSlicePayload

		return nil
	})

	u.SetTitle("Request With JSON Array In Body")

	return u
}
