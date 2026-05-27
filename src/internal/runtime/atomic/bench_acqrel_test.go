// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atomic_test

import (
	"internal/runtime/atomic"
	"testing"
)

// paddedVal occupies at least 64 bytes so that two instances
// allocated on the stack land in different cache lines.
type paddedVal struct {
	a64 atomic.Uint64
	a32 atomic.Uint32
	a8  atomic.Uint8
	_   [51]byte
}

func BenchmarkStoreReleaseThenLoad64(b *testing.B) {
	var x, y paddedVal
	var sum uint64
	for i := 0; i < b.N; i++ {
		x.a64.StoreRelease(uint64(i))
		sum += y.a64.Load()
	}
	_ = sum
}

func BenchmarkStoreReleaseThenLoadAcquire64(b *testing.B) {
	var x, y paddedVal
	var sum uint64
	for i := 0; i < b.N; i++ {
		x.a64.StoreRelease(uint64(i))
		sum += y.a64.LoadAcquire()
	}
	_ = sum
}
