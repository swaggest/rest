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
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.SetTitle("Request With JSON Map In Body")

	type jsonOutput struct {
		Header string         `json:"inHeader"`
		Query  int            `json:"inQuery"`
		Data   JSONMapPayload `json:"data"`
	}

	u.Input = new(jsonMapReq)
	u.Output = new(jsonOutput)

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) (err error) {
		var (
			in  = input.(*jsonMapReq)
			out = output.(*jsonOutput)
		)

		out.Query = in.Query
		out.Header = in.Header
		out.Data = in.JSONMapPayload

		return nil
	})

	return u
}
