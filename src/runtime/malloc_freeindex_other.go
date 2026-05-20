// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !arm64

package runtime

// atomicStoreFreeIndexForScan is a stub for non-arm64 platforms.
// goexperiment.AtomicVar is always false outside arm64, so this function is
// never reached at runtime; it exists solely to keep malloc.go compilable on
// other architectures without requiring atomic.Store16 to be declared there.
//
//go:nosplit
func atomicStoreFreeIndexForScan(span *mspan) {
	span.freeIndexForScan = span.freeindex
}
