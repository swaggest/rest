package nethttp

import "net/http"

type wrapperChecker struct {
	found bool
}

func (*wrapperChecker) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}

// IsWrapperChecker is a hack to mark middleware as a handler wrapper.
// See chirouter.Wrapper Wrap() documentation for more details on the difference.
//
// Wrappers should invoke the check and do early return if it succeeds.
func IsWrapperChecker(h http.Handler) bool {
	if wm, ok := h.(*wrapperChecker); ok {
		wm.found = true

		return true
	}

	return false
}

// MiddlewareIsWrapper is a hack to detect whether middleware is a handler wrapper.
// See chirouter.Wrapper Wrap() documentation for more details on the difference.
func MiddlewareIsWrapper(mw func(h http.Handler) http.Handler) bool {
	wm := &wrapperChecker{}
	mw(wm)

	return wm.found
}
