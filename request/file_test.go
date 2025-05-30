package request_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
)

type ReqEmb struct {
	Simple         string                  `formData:"simple"`
	UploadHeader   *multipart.FileHeader   `formData:"upload"`
	UploadsHeaders []*multipart.FileHeader `formData:"uploads"`
}

type fileReqTest struct {
	ReqEmb
	Upload  multipart.File   `file:"upload"`
	Uploads []multipart.File `formData:"uploads"`
}

func TestDecoder_Decode_fileUploadOptional(t *testing.T) {
	u := usecase.NewIOI(new(ReqEmb), nil, func(_ context.Context, _, _ interface{}) error {
		return nil
	})

	s := web.NewService(openapi3.NewReflector())
	s.Post("/", u)

	b := bytes.NewBuffer(nil)
	w := multipart.NewWriter(b)
	require.NoError(t, w.WriteField("simple", "def"))
	require.NoError(t, w.Close())

	req, err := http.NewRequest(http.MethodPost, "/", b)
	require.NoError(t, err)

	req.Header.Set("Content-Type", w.FormDataContentType())

	rw := httptest.NewRecorder()
	s.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusNoContent, rw.Code)
	assert.Equal(t, ``, rw.Body.String())
}

func TestDecoder_Decode_fileUploadTag(t *testing.T) {
	r := chirouter.NewWrapper(chi.NewRouter())
	apiSchema := openapi.NewCollector(openapi3.NewReflector())
	decoderFactory := request.NewDecoderFactory()
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)

	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	ws := []func(handler http.Handler) http.Handler{
		nethttp.OpenAPIMiddleware(apiSchema),
		request.DecoderMiddleware(decoderFactory),
		request.ValidatorMiddleware(validatorFactory),
		response.EncoderMiddleware,
	}

	r.Wrap(ws...)

	for i, w := range ws {
		assert.True(t, nethttp.MiddlewareIsWrapper(w), i)
	}

	u := struct {
		usecase.Interactor
		usecase.WithInput
	}{}

	u.Input = new(fileReqTest)
	u.Interactor = usecase.Interact(func(_ context.Context, input, _ interface{}) error {
		in, ok := input.(*fileReqTest)
		assert.True(t, ok)

		assert.Equal(t, "def", in.Simple)

		assert.NotNil(t, in.Upload)
		assert.NotNil(t, in.UploadHeader)
		assert.Equal(t, "my.csv", in.UploadHeader.Filename)
		assert.Equal(t, int64(6), in.UploadHeader.Size)
		content, err := ioutil.ReadAll(in.Upload)
		assert.NoError(t, err)
		assert.NoError(t, in.Upload.Close())
		assert.Equal(t, "Hello!", string(content))

		require.Len(t, in.Uploads, 2)
		require.Len(t, in.UploadsHeaders, 2)
		assert.Equal(t, "my1.csv", in.UploadsHeaders[0].Filename)
		assert.Equal(t, int64(7), in.UploadsHeaders[0].Size)
		assert.Equal(t, "my2.csv", in.UploadsHeaders[1].Filename)
		assert.Equal(t, int64(7), in.UploadsHeaders[1].Size)

		content, err = ioutil.ReadAll(in.Uploads[0])
		assert.NoError(t, err)
		assert.NoError(t, in.Uploads[0].Close())
		assert.Equal(t, "Hello1!", string(content))

		content, err = ioutil.ReadAll(in.Uploads[1])
		assert.NoError(t, err)
		assert.NoError(t, in.Uploads[1].Close())
		assert.Equal(t, "Hello2!", string(content))

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

	writer, err = w.CreateFormFile("uploads", "my1.csv")
	assert.NoError(t, err)

	_, err = writer.Write([]byte(`Hello1!`))
	assert.NoError(t, err)

	writer, err = w.CreateFormFile("uploads", "my2.csv")
	assert.NoError(t, err)

	_, err = writer.Write([]byte(`Hello2!`))
	assert.NoError(t, err)

	assert.NoError(t, w.WriteField("simple", "def"))

	assert.NoError(t, w.Close())

	hreq := httptest.NewRequest(http.MethodPost, srv.URL+"/receive", &b)
	hreq.RequestURI = ""
	hreq.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := srv.Client().Do(hreq)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
}
