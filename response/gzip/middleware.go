package gzip

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	gz "github.com/swaggest/rest/gzip"
)

const (
	contentTypeHeader     = "Content-Type"
	contentLengthHeader   = "Content-Length"
	contentEncodingHeader = "Content-Encoding"
	acceptEncodingHeader  = "Accept-Encoding"

	defaultBufferSize = 8 * 1024
)

// Middleware enables gzip compression of handler response for requests that accept gzip encoding.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = maybeGzipResponseWriter(w, r)
		if grw, ok := w.(*gzipResponseWriter); ok {
			defer func() {
				err := grw.Close()
				if err != nil {
					panic(fmt.Sprintf("BUG: cannot close gzip writer: %s", err))
				}
			}()
		}

		next.ServeHTTP(w, r)
	})
}

var (
	gzipWriterPool sync.Pool
	bufWriterPool  sync.Pool
)

func getGzipWriter(w io.Writer) *gzip.Writer {
	v := gzipWriterPool.Get()
	if v == nil {
		zw, err := gzip.NewWriterLevel(w, flate.BestSpeed)
		if err != nil {
			panic(fmt.Sprintf("BUG: cannot create gzip writer: %s", err))
		}

		return zw
	}

	// nolint:errcheck // OK to panic here.
	zw := v.(*gzip.Writer)

	zw.Reset(w)

	return zw
}

func getBufWriter(w io.Writer) *bufio.Writer {
	v := bufWriterPool.Get()
	if v == nil {
		return bufio.NewWriterSize(w, defaultBufferSize)
	}

	// nolint:errcheck // OK to panic here.
	bw := v.(*bufio.Writer)

	bw.Reset(w)

	return bw
}

func maybeGzipResponseWriter(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	ae := r.Header.Get(acceptEncodingHeader)
	if ae == "" {
		return w
	}

	ae = strings.ToLower(ae)

	if n := strings.Index(ae, "gzip"); n < 0 {
		return w
	}

	zrw := &gzipResponseWriter{
		ResponseWriter: w,
	}

	return zrw
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
	bufWriter  *bufio.Writer

	expectCompressedBytes bool
	headersWritten        bool
	disableCompression    bool
}

var _ gz.Writer = &gzipResponseWriter{}

func (rw *gzipResponseWriter) GzipWrite(data []byte) (int, error) {
	if rw.headersWritten {
		return 0, nil
	}

	rw.expectCompressedBytes = true

	return rw.Write(data)
}

func (rw *gzipResponseWriter) writeHeader(statusCode int) {
	if rw.headersWritten {
		return
	}

	if statusCode == http.StatusNoContent ||
		statusCode == http.StatusNotModified ||
		(statusCode >= http.StatusContinue && statusCode < http.StatusOK) {
		rw.disableCompression = true
	}

	h := rw.Header()

	if h.Get(contentEncodingHeader) != "" || rw.disableCompression {
		// The request handler disabled gzip encoding.
		// Send uncompressed response body.
		rw.disableCompression = true
	} else {
		h.Set(contentEncodingHeader, "gzip")

		if !rw.expectCompressedBytes {
			rw.gzipWriter = getGzipWriter(rw.ResponseWriter)
			rw.bufWriter = getBufWriter(rw.gzipWriter)
		}

		h.Del(contentLengthHeader)

		if h.Get(contentTypeHeader) == "" {
			// Disable auto-detection of content-type, since it
			// is incorrectly detected after the compression.
			h.Set(contentTypeHeader, "text/html")
		}
	}

	rw.ResponseWriter.WriteHeader(statusCode)
	rw.headersWritten = true
}

func (rw *gzipResponseWriter) Write(p []byte) (int, error) {
	if !rw.headersWritten {
		rw.writeHeader(http.StatusOK)
	}

	if rw.disableCompression || rw.expectCompressedBytes {
		return rw.ResponseWriter.Write(p)
	}

	return rw.bufWriter.Write(p)
}

func (rw *gzipResponseWriter) WriteHeader(statusCode int) {
	rw.writeHeader(statusCode)
}

func isTrivialNetworkError(err error) bool {
	s := err.Error()
	if strings.Contains(s, "broken pipe") || strings.Contains(s, "reset by peer") {
		return true
	}

	return false
}

// Flush implements http.Flusher.
func (rw *gzipResponseWriter) Flush() {
	if rw.bufWriter == nil || rw.gzipWriter == nil {
		return
	}

	if err := rw.bufWriter.Flush(); err != nil && !isTrivialNetworkError(err) {
		panic(fmt.Sprintf("BUG: cannot flush bufio.Writer: %s", err))
	}

	if err := rw.gzipWriter.Flush(); err != nil && !isTrivialNetworkError(err) {
		panic(fmt.Sprintf("BUG: cannot flush gzip.Writer: %s", err))
	}

	if fw, ok := rw.ResponseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

// Close flushes and closes response.
func (rw *gzipResponseWriter) Close() error {
	if !rw.headersWritten {
		rw.disableCompression = true

		return nil
	}

	if rw.bufWriter == nil || rw.gzipWriter == nil {
		return nil
	}

	rw.Flush()

	err := rw.gzipWriter.Close()

	putBufWriter(rw.bufWriter)
	rw.bufWriter = nil

	putGzipWriter(rw.gzipWriter)
	rw.gzipWriter = nil

	return err
}

func putGzipWriter(zw *gzip.Writer) {
	gzipWriterPool.Put(zw)
}

func putBufWriter(bw *bufio.Writer) {
	bufWriterPool.Put(bw)
}
