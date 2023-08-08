//go:build go1.18

package main

import (
	"context"
	"encoding/json"

	"github.com/swaggest/usecase"
)

type JSONMapPayload map[string]float64

type jsonMapReq struct {
	Header string `header:"X-Header" description:"Simple scalar value in header."`
	Query  int    `query:"in_query" description:"Simple scalar value in query."`
	JSONMapPayload
}

func (j *jsonMapReq) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.JSONMapPayload)
}

func jsonMapBody() usecase.Interactor {
	type jsonOutput struct {
		Header string         `json:"inHeader"`
		Query  int            `json:"inQuery"`
		Data   JSONMapPayload `json:"data"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in jsonMapReq, out *jsonOutput) (err error) {
		out.Query = in.Query
		out.Header = in.Header
		out.Data = in.JSONMapPayload

		return nil
	})

	u.SetTitle("Request With JSON Map In Body")
	u.SetTags("Request")

	return u
}
