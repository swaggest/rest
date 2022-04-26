package main

import (
	"context"
	"encoding/json"
	"net"

	"github.com/google/uuid"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/usecase"
)

type URL string

func (URL) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithFormat("uri")

	return nil
}

type JSONMapPayload map[string]float64

type jsonMapReq struct {
	Header string `header:"X-Header" description:"Simple scalar value in header."`
	Query  int    `query:"in_query" description:"Simple scalar value in query."`
	JSONMapPayload
}

func (j *jsonMapReq) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.JSONMapPayload)
}

type jsonOutput struct {
	Header string         `json:"inHeader"`
	Query  int            `json:"inQuery"`
	Data   JSONMapPayload `json:"data"`
	UUIDs  []uuid.UUID    `json:"uuids"` // Schema provided via type mapping globally.
	IPs    []net.IP       `json:"ips"`   // Schema provided via type mapping globally.
	URLs1  []URL          `json:"urls1"` // Schema prepared for named type.
	URLs2  []string       `json:"urls2"` // Schema updated in parent, no named type needed.
}

func (jsonOutput) PrepareJSONSchema(schema *jsonschema.Schema) error {
	prop := schema.Properties["urls2"]
	items := prop.TypeObjectEns().ItemsEns()
	items.SchemaOrBoolEns().TypeObjectEns().WithFormat("uri").WithMinLength(5)

	return nil
}

func jsonMapBody() usecase.Interactor {
	u := usecase.NewIOI(new(jsonMapReq), new(jsonOutput), func(ctx context.Context, input, output interface{}) (err error) {
		var (
			in  = input.(*jsonMapReq)
			out = output.(*jsonOutput)
		)

		out.Query = in.Query
		out.Header = in.Header
		out.Data = in.JSONMapPayload

		return nil
	})

	u.SetTitle("Request With JSON Map In Body")

	return u
}
