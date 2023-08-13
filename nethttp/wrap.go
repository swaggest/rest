package nethttp

import (
	"net/http"
	"reflect"
	"runtime"
)

// WrapHandler wraps http.Handler with an unwrappable middleware.
//
// Wrapping order is reversed, e.g. if you call WrapHandler(h, mw1, mw2, mw3) middlewares will be
// invoked in order of mw1(mw2(mw3(h))), mw3 first and mw1 last. So that request processing is first
// affected by mw1.
func WrapHandler(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		w := mw[i](h)
		if w == nil {
			panic("nil handler returned from middleware: " + runtime.FuncForPC(reflect.ValueOf(mw[i]).Pointer()).Name())
		}

		fp := reflect.ValueOf(mw[i]).Pointer()
		mwName := runtime.FuncForPC(fp).Name()

		h = &wrappedHandler{
			Handler: w,
			wrapped: h,
			mwName:  mwName,
		}
	}

	return h
}

// HandlerAs finds the first http.Handler in http.Handler's chain that matches target, and if so, sets
// target to that http.Handler value and returns true.
//
// An http.Handler matches target if the http.Handler's concrete value is assignable to the value
// pointed to by target.
//
// HandlerAs will panic if target is not a non-nil pointer to either a type that implements
// http.Handler, or to any interface type.
func HandlerAs(handler http.Handler, target interface{}) bool {
	if target == nil {
		panic("target cannot be nil")
	}

	val := reflect.ValueOf(target)
	typ := val.Type()

	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("target must be a non-nil pointer")
	}

	if e := typ.Elem(); e.Kind() != reflect.Interface && !e.Implements(handlerType) {
		panic("*target must be interface or implement http.Handler")
	}

	targetType := typ.Elem()

	for {
		wrap, isWrap := handler.(*wrappedHandler)

		if isWrap {
			handler = wrap.Handler
		}

		if handler == nil {
			break
		}

		if reflect.TypeOf(handler).AssignableTo(targetType) {
			val.Elem().Set(reflect.ValueOf(handler))

			return true
		}

		if !isWrap {
			break
		}

		handler = wrap.wrapped
	}

	return false
}

var handlerType = reflect.TypeOf((*http.Handler)(nil)).Elem()

type wrappedHandler struct {
	http.Handler
	wrapped http.Handler
	mwName  string
}

func (w *wrappedHandler) String() string {
	if h, ok := w.wrapped.(*wrappedHandler); ok {
		return w.mwName + "(" + h.String() + ")"
	}

	return "handler"
}
