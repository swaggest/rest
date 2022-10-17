//go:build go1.18

package main

import (
	"context"
	"html/template"
	"io"

	"github.com/swaggest/usecase"
)

type htmlResponseOutput struct {
	ID         int
	Filter     string
	Title      string
	Items      []string
	AntiHeader bool `header:"X-Anti-Header"`

	writer io.Writer
}

func (o *htmlResponseOutput) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *htmlResponseOutput) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

func htmlResponse() usecase.Interactor {
	type htmlResponseInput struct {
		ID     int    `path:"id"`
		Filter string `query:"filter"`
		Header bool   `header:"X-Header"`
	}

	const tpl = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
		<a href="/html-response/{{.ID}}?filter={{.Filter}}">Next {{.Title}}</a><br />
		{{range .Items}}<div>{{ . }}</div>{{else}}<div><strong>no rows</strong></div>{{end}}
	</body>
</html>`

	tmpl, err := template.New("htmlResponse").Parse(tpl)
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in htmlResponseInput, out *htmlResponseOutput) (err error) {
		out.AntiHeader = !in.Header
		out.Filter = in.Filter
		out.ID = in.ID + 1
		out.Title = "Foo"
		out.Items = []string{"foo", "bar", "baz"}

		return out.Render(tmpl)
	})

	u.SetTitle("Request With HTML Response")
	u.SetDescription("Request with templated HTML response.")

	return u
}
