//go:build !appengine
// +build !appengine

package request

import (
	"reflect"
	"unsafe"
)

/*
Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// b2s returns a string that refers to the data backing the slice s.
//
// The caller must ensure that the contents of the slice are never again
// mutated, and that its memory either is managed by the Go garbage collector or
// remains valid for the remainder of this process's lifetime.
//
// Programs that use b2s should be tested under the race detector to flag
// erroneous mutations.
//
// Programs that have been adequately tested and shown to be safe may be
// recompiled with the "unsafe" tag to significantly reduce the overhead of this
// function, at the cost of reduced safety checks. Programs built under the race
// detector always have safety checks enabled, even when the "unsafe" tag is
// set.
//
// Copied from https://github.com/bcmills/unsafeslice/blob/v0.2.0/unsafeslice.go#L143.
func b2s(b []byte) string {
	p := unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&b)).Data)

	var s string
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	hdr.Data = uintptr(p)
	hdr.Len = len(b)

	return s
}
