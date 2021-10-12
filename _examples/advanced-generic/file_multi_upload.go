//go:build go1.18
// +build go1.18

package main

import (
	"context"
	"mime/multipart"
	"net/textproto"

	"github.com/swaggest/usecase"
)

func fileMultiUploader() usecase.Interactor {
	type upload struct {
		Simple   string                  `formData:"simple" description:"Simple scalar value in body."`
		Query    int                     `query:"in_query" description:"Simple scalar value in query."`
		Uploads1 []*multipart.FileHeader `formData:"uploads1" description:"Uploads with *multipart.FileHeader."`
		Uploads2 []multipart.File        `formData:"uploads2" description:"Uploads with multipart.File."`
	}

	type info struct {
		Filenames    []string               `json:"filenames"`
		Headers      []textproto.MIMEHeader `json:"headers"`
		Sizes        []int64                `json:"sizes"`
		Upload1Peeks []string               `json:"peeks1"`
		Upload2Peeks []string               `json:"peeks2"`
		Simple       string                 `json:"simple"`
		Query        int                    `json:"inQuery"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in upload, out *info) (err error) {
		out.Query = in.Query
		out.Simple = in.Simple
		for _, o := range in.Uploads1 {
			out.Filenames = append(out.Filenames, o.Filename)
			out.Headers = append(out.Headers, o.Header)
			out.Sizes = append(out.Sizes, o.Size)

			f, err := o.Open()
			if err != nil {
				return err
			}
			p := make([]byte, 100)
			_, err = f.Read(p)
			if err != nil {
				return err
			}

			out.Upload1Peeks = append(out.Upload1Peeks, string(p))

			err = f.Close()
			if err != nil {
				return err
			}
		}

		for _, o := range in.Uploads2 {
			p := make([]byte, 100)
			_, err = o.Read(p)
			if err != nil {
				return err
			}

			out.Upload2Peeks = append(out.Upload2Peeks, string(p))
			err = o.Close()
			if err != nil {
				return err
			}
		}

		return nil
	})

	u.SetTitle("Files Uploads With 'multipart/form-data'")

	return u
}
