package rest

import "io"

// ETagged exposes specific version of resource.
type ETagged interface {
	ETag() string
}

// JSONWriterTo writes JSON payload.
type JSONWriterTo interface {
	JSONWriteTo(w io.Writer) (int, error)
}
