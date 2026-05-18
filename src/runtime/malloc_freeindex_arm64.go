// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build arm64

package runtime

import "internal/runtime/atomic"

// atomicStoreFreeIndexForScan stores span.freeindex into span.freeIndexForScan
// with release semantics (STLRH on arm64), replacing the publicationBarrier+store
// sequence used on the AtomicVar=false path.
//
// It is only referenced from the AtomicVar=true branch of mallocgc* on arm64;
// on default builds the call is in a statically dead branch and is removed
// before SSA generation, leaving the surrounding mallocgc* code byte-identical
// to upstream.
//
//go:nosplit
func atomicStoreFreeIndexForScan(span *mspan) {
	atomic.Store16(&span.freeIndexForScan, span.freeindex)
}
