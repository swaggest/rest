//go:build appengine
// +build appengine

package request

func b2s(b []byte) string {
	return string(b)
}
