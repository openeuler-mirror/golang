// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa

import (
	"cmd/compile/internal/types"
	"testing"
)

func TestLayoutPredicatedBranch(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("entry",
		Bloc("entry",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Valu("branch", OpConstBool, types.Types[types.TBOOL], 1, nil),
			If("branch", "likely", "unlikely")),
		Bloc("likely",
			Goto("successor")),
		Bloc("unlikely",
			Goto("end")),
		Bloc("successor",
			Goto("end")),
		Bloc("end",
			Exit("mem")),
	)
	fun.blocks["entry"].Likely = BranchLikely
	expectedOrder := [5]ID{1, 2, 4, 5, 3}
	CheckFunc(fun.f)
	layout(fun.f)
	CheckFunc(fun.f)

	for i, b := range fun.f.Blocks {
		if b.ID != expectedOrder[i] {
			t.Errorf("block layout order want %d, got %d", expectedOrder[i], b.ID)
		}
	}
}
