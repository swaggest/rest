package nethttp

import (
	"reflect"
	"runtime"

	"github.com/swaggest/fchi"
)

// WrapHandler wraps fchi.Handler with an unwrappable middleware.
//
// Wrapping order is reversed, e.g. if you call WrapHandler(h, mw1, mw2, mw3) middlewares will be
// invoked in order of mw1(mw2(mw3(h))), mw3 first and mw1 last. So that request processing is first
// affected by mw1.
func WrapHandler(h fchi.Handler, mw ...func(fchi.Handler) fchi.Handler) fchi.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		w := mw[i](h)
		if w == nil {
			panic("nil handler returned from middleware: " + runtime.FuncForPC(reflect.ValueOf(mw[i]).Pointer()).Name())
		}

		h = &wrappedHandler{
			Handler: w,
			wrapped: h,
			mwName:  runtime.FuncForPC(reflect.ValueOf(mw[i]).Pointer()).Name(),
		}
	}

	return h
}

// HandlerAs finds the first fchi.Handler in fchi.Handler's chain that matches target, and if so, sets
// target to that fchi.Handler value and returns true.
//
// An fchi.Handler matches target if the fchi.Handler's concrete value is assignable to the value
// pointed to by target.
//
// HandlerAs will panic if target is not a non-nil pointer to either a type that implements
// fchi.Handler, or to any interface type.
func HandlerAs(handler fchi.Handler, target interface{}) bool {
	if target == nil {
		panic("target cannot be nil")
	}

	val := reflect.ValueOf(target)
	typ := val.Type()

	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("target must be a non-nil pointer")
	}

	if e := typ.Elem(); e.Kind() != reflect.Interface && !e.Implements(handlerType) {
		panic("*target must be interface or implement fchi.Handler")
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

var handlerType = reflect.TypeOf((*fchi.Handler)(nil)).Elem()

type wrappedHandler struct {
	fchi.Handler
	wrapped fchi.Handler
	mwName  string
}

func (w *wrappedHandler) String() string {
	if h, ok := w.wrapped.(*wrappedHandler); ok {
		return w.mwName + "(" + h.String() + ")"
	}

	return "handler"
}
