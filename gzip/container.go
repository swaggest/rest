package gzip

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/valyala/fasthttp"
)

// Writer writes gzip data into suitable stream or returns 0, nil.
type Writer interface {
	GzipWrite([]byte) (int, error)
}

// JSONContainer contains compressed JSON.
type JSONContainer struct {
	gz   []byte
	hash string
}

// WriteCompressedBytes writes compressed bytes to response.
//
// Bytes are unpacked if response writer does not support direct gzip writing.
func WriteCompressedBytes(compressed []byte, w io.Writer) (int, error) {
	if rc, ok := w.(*fasthttp.RequestCtx); ok {
		if rc.Request.Header.HasAcceptEncoding("gzip") {
			rc.Request.Header.Del("Accept-Encoding")
			rc.Response.Header.Set("Content-Encoding", "gzip")
			return rc.Write(compressed)
		}
	}

	// Decompress bytes before writing into not instrumented response writer.
	gr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return 0, err
	}

	n, err := io.Copy(w, gr) // nolint:gosec // The origin of compressed data supposed to be app itself, safe to copy.

	return int(n), err
}

// UnpackJSON unmarshals data from JSON container into a Go value.
func (jc JSONContainer) UnpackJSON(v interface{}) error {
	return UnmarshalJSON(jc.gz, v)
}

// PackJSON puts Go value in JSON container.
func (jc *JSONContainer) PackJSON(v interface{}) error {
	res, err := MarshalJSON(v)
	if err != nil {
		return err
	}

	jc.gz = res
	jc.hash = strconv.FormatUint(xxhash.Sum64(res), 36)

	return nil
}

// GzipCompressedJSON returns JSON compressed with gzip.
func (jc JSONContainer) GzipCompressedJSON() []byte {
	return jc.gz
}

// MarshalJSON returns uncompressed JSON.
func (jc JSONContainer) MarshalJSON() (j []byte, err error) {
	b := bytes.NewReader(jc.gz)

	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}

	defer func() {
		clErr := r.Close()
		if err == nil && clErr != nil {
			err = clErr
		}
	}()

	return ioutil.ReadAll(r)
}

// ETag returns hash of compressed bytes.
func (jc JSONContainer) ETag() string {
	return jc.hash
}

// MarshalJSON encodes Go value as JSON and compresses result with gzip.
func MarshalJSON(v interface{}) ([]byte, error) {
	b := bytes.Buffer{}
	w := gzip.NewWriter(&b)

	enc := json.NewEncoder(w)

	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	// Copying result slice to reduce dynamic capacity.
	res := make([]byte, len(b.Bytes()))
	copy(res, b.Bytes())

	return res, nil
}

// UnmarshalJSON decodes compressed JSON bytes into a Go value.
func UnmarshalJSON(data []byte, v interface{}) error {
	b := bytes.NewReader(data)

	r, err := gzip.NewReader(b)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(r)

	err = dec.Decode(v)
	if err != nil {
		return err
	}

	return r.Close()
}

// JSONWriteTo writes JSON payload to writer.
func (jc JSONContainer) JSONWriteTo(w io.Writer) (int, error) {
	return WriteCompressedBytes(jc.gz, w)
}
