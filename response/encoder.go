package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"sync"

	"github.com/swaggest/form/v5"
	"github.com/swaggest/refl"
	"github.com/swaggest/rest"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// Encoder prepares and writes http response.
type Encoder struct {
	JSONWriter func(v interface{})

	outputBufferType     reflect.Type
	outputHeadersEncoder *form.Encoder
	skipRendering        bool
	outputWithWriter     bool
	unwrapInterface      bool
}

type noContent interface {
	NoContent() bool
}

// addressable makes a pointer from a non-pointer values.
func addressable(output interface{}) interface{} {
	if reflect.ValueOf(output).Kind() != reflect.Ptr {
		o := reflect.New(reflect.TypeOf(output))
		o.Elem().Set(reflect.ValueOf(output))

		output = o.Interface()
	}

	return output
}

// SetupOutput configures encoder with and instance of use case output.
func (h *Encoder) SetupOutput(output interface{}, ht *rest.HandlerTrait) {
	h.outputBufferType = reflect.TypeOf(output)
	h.outputHeadersEncoder = nil
	h.skipRendering = true

	if output == nil {
		return
	}

	output = addressable(output)

	// Enable dynamic headers check in interface mode.
	if h.unwrapInterface = reflect.ValueOf(output).Elem().Kind() == reflect.Interface; h.unwrapInterface {
		enc := form.NewEncoder()
		enc.SetMode(form.ModeExplicit)
		enc.SetTagName(string(rest.ParamInHeader))

		h.outputHeadersEncoder = enc
	}

	respHeaderMapping := ht.RespHeaderMapping
	if len(respHeaderMapping) == 0 && refl.HasTaggedFields(output, string(rest.ParamInHeader)) {
		respHeaderMapping = make(map[string]string)

		refl.WalkTaggedFields(reflect.ValueOf(output), func(v reflect.Value, sf reflect.StructField, tag string) {
			respHeaderMapping[sf.Name] = tag
		}, string(rest.ParamInHeader))
	}

	if len(respHeaderMapping) > 0 {
		enc := form.NewEncoder()
		enc.SetMode(form.ModeExplicit)
		enc.RegisterTagNameFunc(func(field reflect.StructField) string {
			return respHeaderMapping[field.Name]
		})

		h.outputHeadersEncoder = enc
	}

	if h.outputBufferType.Kind() == reflect.Ptr {
		h.outputBufferType = h.outputBufferType.Elem()
	}

	if !rest.OutputHasNoContent(output) {
		h.skipRendering = false
	}

	if _, ok := output.(usecase.OutputWithWriter); ok {
		h.skipRendering = true
		h.outputWithWriter = true
	}

	if ht.SuccessStatus != 0 {
		return
	}

	if h.skipRendering && !h.outputWithWriter {
		ht.SuccessStatus = http.StatusNoContent
	} else {
		ht.SuccessStatus = http.StatusOK
	}
}

type jsonEncoder struct {
	enc *json.Encoder
	buf *bytes.Buffer
}

var jsonEncoderPool = sync.Pool{
	New: func() interface{} {
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)

		return &jsonEncoder{
			enc: enc,
			buf: buf,
		}
	},
}

func (h *Encoder) writeJSONResponse(
	w http.ResponseWriter,
	r *http.Request,
	v interface{},
	ht rest.HandlerTrait,
) {
	if ht.SuccessContentType == "" {
		ht.SuccessContentType = "application/json; charset=utf-8"
	}

	if jw, ok := v.(rest.JSONWriterTo); ok {
		w.Header().Set("Content-Type", ht.SuccessContentType)

		_, err := jw.JSONWriteTo(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		return
	}

	e := jsonEncoderPool.Get().(*jsonEncoder) //nolint:errcheck

	e.buf.Reset()
	defer jsonEncoderPool.Put(e)

	err := e.enc.Encode(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if ht.RespValidator != nil {
		err = ht.RespValidator.ValidateJSONBody(e.buf.Bytes())
		if err != nil {
			code, er := rest.Err(status.Wrap(fmt.Errorf("bad response: %w", err), status.Internal))
			h.WriteErrResponse(w, r, code, er)

			return
		}
	}

	w.Header().Set("Content-Length", strconv.Itoa(e.buf.Len()))
	w.Header().Set("Content-Type", ht.SuccessContentType)
	w.WriteHeader(ht.SuccessStatus)

	if r.Method == http.MethodHead {
		return
	}

	_, err = w.Write(e.buf.Bytes())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// WriteErrResponse encodes and writes error to response.
func (h *Encoder) WriteErrResponse(w http.ResponseWriter, r *http.Request, statusCode int, response interface{}) {
	contentType := "application/json; charset=utf-8"

	e := jsonEncoderPool.Get().(*jsonEncoder) //nolint:errcheck

	e.buf.Reset()
	defer jsonEncoderPool.Put(e)

	err := e.enc.Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(e.buf.Len()))
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)

	if r.Method == http.MethodHead {
		return
	}

	_, err = w.Write(e.buf.Bytes())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// WriteSuccessfulResponse encodes and writes successful output of use case interactor to http response.
func (h *Encoder) WriteSuccessfulResponse(
	w http.ResponseWriter,
	r *http.Request,
	output interface{},
	ht rest.HandlerTrait,
) {
	if h.unwrapInterface {
		output = reflect.ValueOf(output).Elem().Interface()
	}

	if etagged, ok := output.(rest.ETagged); ok {
		etag := etagged.ETag()
		if etag != "" {
			w.Header().Set("Etag", etag)
		}
	}

	if h.outputHeadersEncoder != nil && !h.whiteHeader(w, r, output, ht) {
		return
	}

	skipRendering := h.skipRendering
	if !skipRendering {
		if nc, ok := output.(noContent); ok {
			skipRendering = nc.NoContent()
			if skipRendering && ht.SuccessStatus == 0 {
				ht.SuccessStatus = http.StatusNoContent
			}
		}
	}

	if ht.SuccessStatus == 0 {
		ht.SuccessStatus = http.StatusOK
	}

	if skipRendering {
		if ht.SuccessStatus != http.StatusOK {
			w.WriteHeader(ht.SuccessStatus)
		}

		return
	}

	h.writeJSONResponse(w, r, output, ht)
}

func (h *Encoder) whiteHeader(w http.ResponseWriter, r *http.Request, output interface{}, ht rest.HandlerTrait) bool {
	var headerValues map[string]interface{}
	if ht.RespValidator != nil {
		headerValues = make(map[string]interface{})
	}

	headers, err := h.outputHeadersEncoder.Encode(output, headerValues)
	if err != nil {
		code, er := rest.Err(err)
		h.WriteErrResponse(w, r, code, er)

		return false
	}

	if ht.RespValidator != nil {
		err = ht.RespValidator.ValidateData(rest.ParamInHeader, headerValues)
		if err != nil {
			code, er := rest.Err(status.Wrap(fmt.Errorf("bad response: %w", err), status.Internal))
			h.WriteErrResponse(w, r, code, er)

			return false
		}
	}

	for header, val := range headers {
		if len(val) == 1 {
			w.Header().Set(header, val[0])
		}
	}

	return true
}

// MakeOutput instantiates a value for use case output port.
func (h *Encoder) MakeOutput(w http.ResponseWriter, ht rest.HandlerTrait) interface{} {
	if h.outputBufferType == nil {
		return nil
	}

	output := reflect.New(h.outputBufferType).Interface()

	if h.outputWithWriter {
		if withWriter, ok := output.(usecase.OutputWithWriter); ok {
			if h.outputHeadersEncoder != nil || ht.SuccessContentType != "" {
				withWriter.SetWriter(&writerWithHeaders{
					ResponseWriter: w,
					responseWriter: h,
					trait:          ht,
					output:         output,
				})
			} else {
				withWriter.SetWriter(w)
			}
		}
	}

	return output
}

type writerWithHeaders struct {
	http.ResponseWriter

	responseWriter *Encoder
	trait          rest.HandlerTrait
	output         interface{}
	headersSet     bool
}

func (w *writerWithHeaders) setHeaders() error {
	if w.responseWriter.outputHeadersEncoder == nil {
		return nil
	}

	headers, err := w.responseWriter.outputHeadersEncoder.Encode(w.output)
	if err != nil {
		return err
	}

	for header, val := range headers {
		if len(val) == 1 {
			w.Header().Set(header, val[0])
		}
	}

	return err
}

func (w *writerWithHeaders) Write(data []byte) (int, error) {
	if !w.headersSet {
		if err := w.setHeaders(); err != nil {
			return 0, err
		}

		if w.trait.SuccessContentType != "" {
			w.Header().Set("Content-Type", w.trait.SuccessContentType)
		}

		w.headersSet = true
	}

	return w.ResponseWriter.Write(data)
}
