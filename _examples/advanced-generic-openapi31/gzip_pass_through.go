//go:build go1.18

package main

import (
	"context"

	"github.com/swaggest/rest/gzip"
	"github.com/swaggest/usecase"
)

type gzipPassThroughInput struct {
	PlainStruct bool `query:"plainStruct" description:"Output plain structure instead of gzip container."`
	CountItems  bool `query:"countItems" description:"Invokes internal decoding of compressed data."`
}

// gzipPassThroughOutput defers data to an accessor function instead of using struct directly.
// This is necessary to allow containers that can data in binary wire-friendly format.
type gzipPassThroughOutput interface {
	// Data should be accessed though an accessor to allow container interface.
	gzipPassThroughStruct() gzipPassThroughStruct
}

// gzipPassThroughStruct represents the actual structure that is held in the container
// and implements gzipPassThroughOutput to be directly useful in output.
type gzipPassThroughStruct struct {
	Header string   `header:"X-Header" json:"-"`
	ID     int      `json:"id"`
	Text   []string `json:"text"`
}

func (d gzipPassThroughStruct) gzipPassThroughStruct() gzipPassThroughStruct {
	return d
}

// gzipPassThroughContainer is wrapping gzip.JSONContainer and implements gzipPassThroughOutput.
type gzipPassThroughContainer struct {
	Header string `header:"X-Header" json:"-"`
	gzip.JSONContainer
}

func (dc gzipPassThroughContainer) gzipPassThroughStruct() gzipPassThroughStruct {
	var p gzipPassThroughStruct

	err := dc.UnpackJSON(&p)
	if err != nil {
		panic(err)
	}

	return p
}

func directGzip() usecase.Interactor {
	// Prepare moderately big JSON, resulting JSON payload is ~67KB.
	rawData := gzipPassThroughStruct{
		ID: 123,
	}
	for i := 0; i < 400; i++ {
		rawData.Text = append(rawData.Text, "Quis autem vel eum iure reprehenderit, qui in ea voluptate velit esse, "+
			"quam nihil molestiae consequatur, vel illum, qui dolorem eum fugiat, quo voluptas nulla pariatur?")
	}

	// Precompute compressed data container. Generally this step should be owned by a caching storage of data.
	dataFromCache := gzipPassThroughContainer{}

	err := dataFromCache.PackJSON(rawData)
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in gzipPassThroughInput, out *gzipPassThroughOutput) error {
		if in.PlainStruct {
			o := rawData
			o.Header = "cba"
			*out = o
		} else {
			o := dataFromCache
			o.Header = "abc"
			*out = o
		}

		// Imitating an internal read operation on data in container.
		if in.CountItems {
			cnt := len((*out).gzipPassThroughStruct().Text)
			println("items: ", cnt)
		}

		return nil
	})
	u.SetTags("Response")

	return u
}
