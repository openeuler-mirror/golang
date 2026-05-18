// asmcheck -gcflags=-aggressivedse

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codegen

import "unsafe"

// This file contains code generation tests that only trigger when the
// -aggressivedse compiler flag is enabled. Default-path expectations belong in
// stack.go (and other non-gcflags-qualified codegen files) so that the
// conservative DSE coverage is not silently rerouted through aggressiveDse.

// mySlice mirrors the runtime slice header so we can construct a slice via
// a bytewise struct assignment and then reinterpret-cast it.
type mySlice struct {
	array unsafe.Pointer
	len   int
	cap   int
}

// Aggressive DSE is expected to collapse the mySlice struct initialization
// into a direct slice value, leaving no stack frame and no SP manipulation.
// arm64:"TEXT\t.*, [$]0-"
func sliceInit(base uintptr) []uintptr {
	const ptrSize = 8
	size := uintptr(4096)
	bitmapSize := size / ptrSize / 8
	elements := int(bitmapSize / ptrSize)
	var sl mySlice
	sl = mySlice{
		unsafe.Pointer(base + size - bitmapSize),
		elements,
		elements,
	}
	return *(*[]uintptr)(unsafe.Pointer(&sl))
}
