package gzip

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

// Writer writes gzip data into suitable stream or returns 0, nil.
type Writer interface {
	GzipWrite(d []byte) (int, error)
}

// JSONContainer contains compressed JSON.
type JSONContainer struct {
	gz       []byte
	hash     string
	gzipHash string
}

// WriteCompressedBytes writes compressed bytes to response.
//
// Bytes are unpacked if response writer does not support direct gzip writing.
func WriteCompressedBytes(compressed []byte, w io.Writer) (int, error) {
	if gw, ok := w.(Writer); ok {
		n, err := gw.GzipWrite(compressed)
		if n != 0 {
			return n, err
		}
	}

	// Decompress bytes before writing into not instrumented response writer.
	gr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return 0, err
	}

	n, err := io.Copy(w, gr) //nolint:gosec // The origin of compressed data supposed to be app itself, safe to copy.

	return int(n), err
}

// UnpackJSON unmarshals data from JSON container into a Go value.
func (jc JSONContainer) UnpackJSON(v interface{}) error {
	return UnmarshalJSON(jc.gz, v)
}

// PackJSON puts Go value in JSON container.
func (jc *JSONContainer) PackJSON(v interface{}) error {
	res, hash, gzipHash, err := marshalJSON(v)
	if err != nil {
		return err
	}

	jc.gz = res
	jc.hash = hash
	jc.gzipHash = gzipHash

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

// ETag returns hash of uncompressed JSON content, identifying the identity representation.
//
// Unlike hashing compressed bytes, this is stable across changes to the gzip/flate
// implementation (e.g. across Go versions), which do not guarantee stable output bytes
// for the same input.
func (jc JSONContainer) ETag() string {
	return jc.hash
}

// GzipETag returns a hash of uncompressed JSON content salted for the gzip-compressed
// representation, so it differs from ETag.
//
// The gzip and identity representations are byte-distinct, so gzip-compression middleware
// (see response/gzip) uses this instead of ETag when serving the gzip body, to avoid one
// strong validator being shared between two different representations.
func (jc JSONContainer) GzipETag() string {
	return jc.gzipHash
}

// MarshalJSON encodes Go value as JSON and compresses result with gzip.
func MarshalJSON(v interface{}) ([]byte, error) {
	res, _, _, err := marshalJSON(v)

	return res, err
}

// marshalJSON encodes Go value as JSON, compressing it with gzip and hashing its uncompressed
// content in a single pass. It computes two digests of that same content, one plain (hash) and
// one with a "gzip" marker mixed in (gzipHash), so the gzip and identity representations get
// distinct ETags without a second pass over the (potentially much larger, pre-compression) content.
func marshalJSON(v interface{}) (compressed []byte, hash, gzipHash string, err error) {
	b := bytes.Buffer{}
	w := gzip.NewWriter(&b)
	h := xxhash.New()
	gh := xxhash.New()

	enc := json.NewEncoder(io.MultiWriter(w, h, gh))

	if err := enc.Encode(v); err != nil {
		return nil, "", "", err
	}

	if err := w.Close(); err != nil {
		return nil, "", "", err
	}

	_, _ = gh.Write(gzipETagSalt) // xxhash.Digest.Write never returns an error.

	// Copying result slice to reduce dynamic capacity.
	res := make([]byte, len(b.Bytes()))
	copy(res, b.Bytes())

	return res, strconv.FormatUint(h.Sum64(), 36), strconv.FormatUint(gh.Sum64(), 36), nil
}

// gzipETagSalt is mixed into the gzip-representation digest so it differs from the plain
// content digest.
var gzipETagSalt = []byte("gzip")

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
	// The gzip and identity bodies are byte-distinct representations of the same content, so a
	// validator shared between them would let a cache serve the wrong variant. If w can take the
	// gzip-compressed bytes directly (see WriteCompressedBytes), swap in the ETag already
	// computed for that representation, in place of whatever ETag a response encoder set before
	// the compression decision was made.
	if rw, ok := w.(http.ResponseWriter); ok {
		if _, ok := w.(Writer); ok && rw.Header().Get("Etag") != "" {
			rw.Header().Set("Etag", jc.gzipHash)
		}
	}

	return WriteCompressedBytes(jc.gz, w)
}
