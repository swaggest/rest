//go:build go1.18
// +build go1.18

package main

import (
	"context"
	"mime/multipart"
	"net/textproto"

	"github.com/swaggest/usecase"
)

func fileUploader() usecase.Interactor {
	type upload struct {
		Simple  string                `formData:"simple" description:"Simple scalar value in body."`
		Query   int                   `query:"in_query" description:"Simple scalar value in query."`
		Upload1 *multipart.FileHeader `formData:"upload1" description:"Upload with *multipart.FileHeader."`
		Upload2 multipart.File        `formData:"upload2" description:"Upload with multipart.File."`
	}

	type info struct {
		Filename    string               `json:"filename"`
		Header      textproto.MIMEHeader `json:"header"`
		Size        int64                `json:"size"`
		Upload1Peek string               `json:"peek1"`
		Upload2Peek string               `json:"peek2"`
		Simple      string               `json:"simple"`
		Query       int                  `json:"inQuery"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in upload, out *info) (err error) {
		out.Query = in.Query
		out.Simple = in.Simple
		out.Filename = in.Upload1.Filename
		out.Header = in.Upload1.Header
		out.Size = in.Upload1.Size

		f, err := in.Upload1.Open()
		if err != nil {
			return err
		}

		defer func() {
			clErr := f.Close()
			if clErr != nil && err == nil {
				err = clErr
			}

			clErr = in.Upload2.Close()
			if clErr != nil && err == nil {
				err = clErr
			}
		}()

		p := make([]byte, 100)
		_, err = f.Read(p)
		if err != nil {
			return err
		}

		out.Upload1Peek = string(p)

		p = make([]byte, 100)
		_, err = in.Upload2.Read(p)
		if err != nil {
			return err
		}

		out.Upload2Peek = string(p)

		return nil
	})

	u.SetTitle("File Upload With 'multipart/form-data'")

	return u
}
