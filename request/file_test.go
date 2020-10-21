package request_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
)

type ReqEmb struct {
	UploadHeader *multipart.FileHeader `formData:"upload"`
}

type fileReqTest struct {
	ReqEmb
	Upload multipart.File `file:"upload"`
}

func TestMapper_Decode_fileUploadTag(t *testing.T) {
	r := chirouter.NewWrapper(chi.NewRouter())
	apiSchema := openapi.Collector{}
	decoderFactory := request.NewDecoderFactory()
	validatorFactory := jsonschema.NewFactory(&apiSchema, &apiSchema)

	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	r.Use(
		nethttp.OpenAPIMiddleware(&apiSchema),
		request.DecoderMiddleware(decoderFactory),
		request.ValidatorMiddleware(validatorFactory),
		response.EncoderMiddleware,
	)

	u := struct {
		usecase.Interactor
		usecase.WithInput
	}{}

	u.Input = new(fileReqTest)
	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		inp, ok := input.(*fileReqTest)
		assert.True(t, ok)
		assert.NotNil(t, inp.Upload)
		assert.NotNil(t, inp.UploadHeader)
		assert.Equal(t, "my.csv", inp.UploadHeader.Filename)
		assert.Equal(t, int64(6), inp.UploadHeader.Size)
		content, err := ioutil.ReadAll(inp.Upload)
		assert.NoError(t, err)
		assert.Equal(t, "Hello!", string(content))

		return nil
	})

	h := nethttp.NewHandler(u)
	r.Method(http.MethodPost, "/receive", h)

	srv := httptest.NewServer(r)
	defer srv.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	writer, err := w.CreateFormFile("upload", "my.csv")
	assert.NoError(t, err)

	_, err = writer.Write([]byte(`Hello!`))
	assert.NoError(t, err)

	assert.NoError(t, w.Close())

	hreq := httptest.NewRequest(http.MethodPost, srv.URL+"/receive", &b)
	hreq.RequestURI = ""
	hreq.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := srv.Client().Do(hreq)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
}
