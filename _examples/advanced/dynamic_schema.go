package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

type dynamicInput struct {
	jsonschema.Struct
	request.EmbeddedSetter

	// Type is a static field example.
	Type string `query:"type"`
}

type dynamicOutput struct {
	// Embedded jsonschema.Struct exposes dynamic fields for documentation.
	jsonschema.Struct

	jsonFields   map[string]interface{}
	headerFields map[string]string

	// Status is a static field example.
	Status string `json:"status"`
}

func (o dynamicOutput) SetupResponseHeader(h http.Header) {
	for k, v := range o.headerFields {
		h.Set(k, v)
	}
}

func (o dynamicOutput) MarshalJSON() ([]byte, error) {
	if o.jsonFields == nil {
		o.jsonFields = map[string]interface{}{}
	}

	o.jsonFields["status"] = o.Status

	return json.Marshal(o.jsonFields)
}

func dynamicSchema() usecase.Interactor {
	dynIn := dynamicInput{}
	dynIn.DefName = "DynIn123"
	dynIn.Struct.Fields = []jsonschema.Field{
		{Name: "Foo", Value: 123, Tag: `header:"foo" enum:"123,456,789"`},
		{Name: "Bar", Value: "abc", Tag: `query:"bar"`},
	}

	dynOut := dynamicOutput{}
	dynOut.DefName = "DynOut123"
	dynOut.Struct.Fields = []jsonschema.Field{
		{Name: "Foo", Value: 123, Tag: `header:"foo" enum:"123,456,789"`},
		{Name: "Bar", Value: "abc", Tag: `json:"bar"`},
	}

	u := usecase.NewIOI(dynIn, dynOut, func(ctx context.Context, input, output interface{}) (err error) {
		var (
			in  = input.(dynamicInput)
			out = output.(*dynamicOutput)
		)

		switch in.Type {
		case "ok":
			out.Status = "ok"
			out.jsonFields = map[string]interface{}{
				"bar": in.Request().URL.Query().Get("bar"),
			}
			out.headerFields = map[string]string{
				"foo": in.Request().Header.Get("foo"),
			}
		case "invalid_argument":
			return status.Wrap(errors.New("bad value for foo"), status.InvalidArgument)
		case "conflict":
			return status.Wrap(ctxd.NewError(ctx, "conflict", "foo", "bar"),
				status.AlreadyExists)
		}

		return nil
	})

	u.SetTitle("Dynamic Request Schema")
	u.SetDescription("This use case demonstrates documentation of types that are only known at runtime.")
	u.SetExpectedErrors(status.InvalidArgument, status.FailedPrecondition, status.AlreadyExists)

	return u
}
