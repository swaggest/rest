package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/swaggest/form/v5"
	"github.com/swaggest/refl"
	"github.com/swaggest/rest"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

type (
	// Setter captures original http.ResponseWriter.
	//
	// Implement this interface on a pointer to your output structure to get access to http.ResponseWriter.
	Setter interface {
		SetResponseWriter(rw http.ResponseWriter)
	}
)

// Encoder prepares and writes http response.
type Encoder struct {
	JSONWriter func(v interface{})

	outputBufferType     reflect.Type
	outputHeadersEncoder *form.Encoder
	outputCookiesEncoder *form.Encoder
	outputCookieBase     []http.Cookie
	skipRendering        bool
	outputWithWriter     bool
	unwrapInterface      bool

	dynamicWithHeadersSetup bool
	dynamicSetter           bool
	dynamicETagged          bool
	dynamicNoContent        bool
}

type noContent interface {
	// NoContent controls whether status 204 should be used in response to current request.
	NoContent() bool
}

type outputWithHeadersSetup interface {
	// SetupResponseHeader gives access to response headers of current request.
	SetupResponseHeader(h http.Header)
}

// DefaultSuccessResponseContentType is a package-level variable set to
// default success response content type.
var DefaultSuccessResponseContentType = "application/json"

// DefaultErrorResponseContentType is a package-level variable set to
// default error response content type.
var DefaultErrorResponseContentType = "application/json"

// addressable makes a pointer from a non-pointer values.
func addressable(output interface{}) interface{} {
	if reflect.ValueOf(output).Kind() != reflect.Ptr {
		o := reflect.New(reflect.TypeOf(output))
		o.Elem().Set(reflect.ValueOf(output))

		output = o.Interface()
	}

	return output
}

func (h *Encoder) setupHeadersEncoder(output interface{}, ht *rest.HandlerTrait) {
	// Enable dynamic headers check in interface mode.
	if h.unwrapInterface {
		enc := form.NewEncoder()
		enc.SetMode(form.ModeExplicit)
		enc.SetTagName(string(rest.ParamInHeader))

		h.outputHeadersEncoder = enc

		return
	}

	respHeaderMapping := ht.RespHeaderMapping
	if len(respHeaderMapping) == 0 && refl.HasTaggedFields(output, string(rest.ParamInHeader)) {
		respHeaderMapping = make(map[string]string)

		refl.WalkTaggedFields(reflect.ValueOf(output), func(_ reflect.Value, sf reflect.StructField, _ string) {
			// Converting name to canonical form, while keeping omitempty and any other options.
			t := sf.Tag.Get(string(rest.ParamInHeader))
			parts := strings.Split(t, ",")
			parts[0] = http.CanonicalHeaderKey(parts[0])
			t = strings.Join(parts, ",")

			respHeaderMapping[sf.Name] = t
		}, string(rest.ParamInHeader))
	}

	if len(respHeaderMapping) > 0 {
		enc := form.NewEncoder()
		enc.SetMode(form.ModeExplicit)
		enc.RegisterTagNameFunc(func(field reflect.StructField) string {
			if name, ok := respHeaderMapping[field.Name]; ok {
				return name
			}

			if field.Anonymous {
				return ""
			}

			return "-"
		})

		h.outputHeadersEncoder = enc
	}
}

func (h *Encoder) setupCookiesEncoder(output interface{}, ht *rest.HandlerTrait) {
	// Enable dynamic headers check in interface mode.
	if h.unwrapInterface {
		enc := form.NewEncoder()
		enc.SetMode(form.ModeExplicit)
		enc.SetTagName(string(rest.ParamInCookie))

		h.outputCookiesEncoder = enc

		return
	}

	respCookieMapping := ht.RespCookieMapping
	if len(respCookieMapping) == 0 && refl.HasTaggedFields(output, string(rest.ParamInCookie)) {
		respCookieMapping = make(map[string]http.Cookie)
		h.outputCookieBase = make([]http.Cookie, 0)

		refl.WalkTaggedFields(reflect.ValueOf(output), func(_ reflect.Value, sf reflect.StructField, tag string) {
			c := http.Cookie{
				Name: tag,
			}

			options := strings.Split(sf.Tag.Get("cookie"), ",")[1:]
			if len(options) > 0 {
				resp := http.Response{}
				resp.Header = make(http.Header)
				resp.Header.Add("Set-Cookie", tag+"=x;"+strings.Join(options, ";"))

				cc := resp.Cookies()
				if len(cc) == 1 {
					c = *cc[0]
				}
			}

			c.Value = ""
			c.Raw = ""

			h.outputCookieBase = append(h.outputCookieBase, c)
			respCookieMapping[sf.Name] = c
		}, string(rest.ParamInCookie))
	}

	if len(respCookieMapping) > 0 {
		enc := form.NewEncoder()
		enc.SetMode(form.ModeExplicit)
		enc.RegisterTagNameFunc(func(field reflect.StructField) string {
			if c, ok := respCookieMapping[field.Name]; ok {
				return c.Name
			}

			if field.Anonymous {
				return ""
			}

			return "-"
		})

		h.outputCookiesEncoder = enc
	}
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

	h.unwrapInterface = reflect.ValueOf(output).Elem().Kind() == reflect.Interface

	if _, ok := output.(outputWithHeadersSetup); ok || h.unwrapInterface {
		h.dynamicWithHeadersSetup = true
	}

	if _, ok := output.(Setter); ok || h.unwrapInterface {
		h.dynamicSetter = true
	}

	if _, ok := output.(rest.ETagged); ok || h.unwrapInterface {
		h.dynamicETagged = true
	}

	if _, ok := output.(noContent); ok || h.unwrapInterface {
		h.dynamicNoContent = true
	}

	h.setupHeadersEncoder(output, ht)
	h.setupCookiesEncoder(output, ht)

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

	ht.SuccessStatus = h.successStatus(output)
}

func (h *Encoder) successStatus(output interface{}) int {
	if outputWithStatus, ok := output.(rest.OutputWithHTTPStatus); ok {
		return outputWithStatus.HTTPStatus()
	}

	if h.skipRendering && !h.outputWithWriter {
		return http.StatusNoContent
	}

	return http.StatusOK
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
		ht.SuccessContentType = DefaultSuccessResponseContentType
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
			h.writeError(status.Wrap(fmt.Errorf("bad response: %w", err), status.Internal), w, r, ht)

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
	e := jsonEncoderPool.Get().(*jsonEncoder) //nolint:errcheck

	e.buf.Reset()
	defer jsonEncoderPool.Put(e)

	err := e.enc.Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// Skip statuses that do not allow response body (1xx, 204, 304).
	if !(statusCode < http.StatusOK || statusCode == http.StatusNoContent || statusCode == http.StatusNotModified) {
		w.Header().Set("Content-Length", strconv.Itoa(e.buf.Len()))

		contentType := DefaultErrorResponseContentType
		w.Header().Set("Content-Type", contentType)
	}

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

	if h.dynamicETagged {
		if etagged, ok := output.(rest.ETagged); ok {
			etag := etagged.ETag()
			if etag != "" {
				w.Header().Set("Etag", etag)
			}
		}
	}

	if !h.whiteHeader(w, r, output, ht) {
		return
	}

	if !h.writeCookies(w, r, output, ht) {
		return
	}

	skipRendering := h.skipRendering
	if !skipRendering && h.dynamicNoContent {
		if nc, ok := output.(noContent); ok {
			skipRendering = nc.NoContent()
			if skipRendering && ht.SuccessStatus == 0 {
				ht.SuccessStatus = http.StatusNoContent
			}
		}
	}

	if ht.SuccessStatus == 0 {
		ht.SuccessStatus = h.successStatus(output)
	}

	if skipRendering {
		if !h.outputWithWriter && !h.dynamicSetter && ht.SuccessStatus != http.StatusOK {
			w.WriteHeader(ht.SuccessStatus)
		}

		return
	}

	h.writeJSONResponse(w, r, output, ht)
}

func (h *Encoder) writeError(err error, w http.ResponseWriter, r *http.Request, ht rest.HandlerTrait) {
	if ht.MakeErrResp != nil {
		code, er := ht.MakeErrResp(r.Context(), err)
		h.WriteErrResponse(w, r, code, er)
	} else {
		code, er := rest.Err(err)
		h.WriteErrResponse(w, r, code, er)
	}
}

func (h *Encoder) whiteHeader(w http.ResponseWriter, r *http.Request, output interface{}, ht rest.HandlerTrait) bool {
	if h.dynamicWithHeadersSetup {
		if sh, ok := output.(outputWithHeadersSetup); ok {
			sh.SetupResponseHeader(w.Header())
		}
	}

	if h.outputHeadersEncoder == nil {
		return true
	}

	var goValues map[string]interface{}
	if ht.RespValidator != nil {
		goValues = make(map[string]interface{})
	}

	headers, err := h.outputHeadersEncoder.Encode(output, goValues)
	if err != nil {
		h.writeError(err, w, r, ht)

		return false
	}

	if ht.RespValidator != nil {
		if err := ht.RespValidator.ValidateData(rest.ParamInHeader, goValues); err != nil {
			h.writeError(status.Wrap(fmt.Errorf("bad response: %w", err), status.Internal), w, r, ht)

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

func (h *Encoder) writeCookies(w http.ResponseWriter, r *http.Request, output interface{}, ht rest.HandlerTrait) bool {
	if h.outputCookiesEncoder == nil {
		return true
	}

	cookies, err := h.outputCookiesEncoder.Encode(output, nil)
	if err != nil {
		h.writeError(err, w, r, ht)

		return false
	}

	if h.outputCookieBase != nil {
		for _, c := range h.outputCookieBase {
			if val, ok := cookies[c.Name]; ok && len(val) == 1 && val[0] != "" {
				c := c
				c.Value = val[0]

				http.SetCookie(w, &c)
			}
		}
	} else {
		for cookie, val := range cookies {
			c := http.Cookie{}
			c.Name = cookie
			c.Value = val[0]

			http.SetCookie(w, &c)
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

	if h.dynamicSetter {
		if setter, ok := output.(Setter); ok {
			setter.SetResponseWriter(w)
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

// EmbeddedSetter can capture http.ResponseWriter in your output structure.
type EmbeddedSetter struct {
	rw http.ResponseWriter
}

// SetResponseWriter implements Setter.
func (e *EmbeddedSetter) SetResponseWriter(rw http.ResponseWriter) {
	e.rw = rw
}

// ResponseWriter is an accessor.
func (e *EmbeddedSetter) ResponseWriter() http.ResponseWriter {
	return e.rw
}
