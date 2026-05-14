// asmcheck -gcflags=-aggressivepatterns

// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codegen

// --------------------------------------- //
//   Test ARM64 folding for slice masks:   //
//      NEG+SUB -> SUB                     //
//      LSL+(AND>>63)+ADD ->               //
//                 (AND>>63)+ADDshiftLL    //
//   useful mostly for slice operations.   //
// --------------------------------------- //

func SliceOpts(a []int, b int) int {
	// arm64:-"NEG",-"LSL"
	return a[b:][b]
}

func SliceAndIndex(a []int, b int) int {
	// arm64:"AND\tR[0-9]+->63","ADD\tR[0-9]+<<3"
	return a[b:][b]
}

// --------------------------------------- //
//
//	Test ARM64 folding SUB into CMP:      //
//	   SUB+CMP[0]+BLS -> CMP+BEQ          //
//
// --------------------------------------- //
func SlicePut(a []byte, c uint8) []byte {
	// arm64:`CBZ\tR1`
	a[0] = c
	a = a[1:]
	// arm64:`CMP\t\$1, R1`
	a[0] = c
	a = a[1:]
	// arm64:`CMP\t\$2, R1`
	a[0] = c
	a = a[1:]
	return a
}
