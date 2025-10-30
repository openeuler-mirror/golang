// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goexperiment.pagenum

package runtime

import (
	"internal/goarch"
	"unsafe"
	"internal/cpu"
)

var minSizeForMallocHeader uintptr

func memMinSizeCalKp(arr *[68]uint16, len uint64) int

func findMinSize(class uintptr, size uint64, min uint64) int64

func checkMultPage(pages uintptr, idx uint64, size uint64) bool

func checkMinimumSize() {
	minSizeForMallocHeader = goarch.PtrSize * ptrBits
	switch GOARCH {
	case "arm64":
		if cpu.ARM64.HasSVE {
			memMinSizeKp := memMinSizeCalKp(&class_to_size, _NumSizeClasses)
			minSizeForMallocHeader = minSizeForMallocHeader / (uintptr)(memMinSizeKp) * 2
		}
	}

	minSizeForMallocHeaderIsSizeClassIdx := findMinSize(uintptr(unsafe.Pointer(&class_to_size[0])),
		uint64(len(class_to_size)), uint64(minSizeForMallocHeader))
	if minSizeForMallocHeaderIsSizeClassIdx == -1 {
		throw("min size of malloc header is not a size class boundary")
	}
	sizeClassesUpToMinSizeForMallocHeaderAreMultPage := checkMultPage(uintptr(unsafe.Pointer(&class_to_allocnpages[0])),
		uint64(minSizeForMallocHeaderIsSizeClassIdx), uint64(pageNumberMultiplierForMinSize))
	if !sizeClassesUpToMinSizeForMallocHeaderAreMultPage {
		throw("expected all size classes up to min size for malloc header to fit in (pageNumberMultiplierForMinSize" +
			")-page spans")
	}
}
