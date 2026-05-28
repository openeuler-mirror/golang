// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goexperiment.pagenum

package runtime

import (
	"internal/goarch"
)

// Keeping this const helps optimizations (e.g. aggressiveprove)
// to reason about its value and eliminate bound checks.
const minSizeForMallocHeader = goarch.PtrSize * ptrBits

func checkMinimumSize() {
	minSizeForMallocHeaderIsSizeClass := false
	sizeClassesUpToMinSizeForMallocHeaderAreMultPage := true
	for i := 0; i < len(class_to_size); i++ {
		if class_to_allocnpages[i] > pageNumberMultiplierForMinSize {
			sizeClassesUpToMinSizeForMallocHeaderAreMultPage = false
		}
		if minSizeForMallocHeader == uintptr(class_to_size[i]) {
			minSizeForMallocHeaderIsSizeClass = true
			break
		}
	}
	if !minSizeForMallocHeaderIsSizeClass {
		throw("min size of malloc header is not a size class boundary")
	}
	if !sizeClassesUpToMinSizeForMallocHeaderAreMultPage {
		throw("expected all size classes up to min size for malloc header to fit in (pageNumberMultiplierForMinSize" +
			")-page spans")
	}
}
