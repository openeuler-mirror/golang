// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa

import (
	"cmd/compile/internal/types"
	"testing"
)

func checkBlockOrder(expected []ID, wanted []*Block) bool {
	if len(expected) != len(wanted) {
		return false
	}
	for i, b := range wanted {
		if b.ID != expected[i] {
			return false
		}
	}
	return true
}

func TestLoopRotateSimpleLoop1(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("b1",
		Bloc("b1",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Goto("b2")),
		Bloc("b2",
			Goto("b3")),
		Bloc("b3",
			Goto("b4")),
		Bloc("b4",
			Valu("cond", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond", "b2", "b5")),
		Bloc("b5",
			Exit("mem")),
	)

	// b1 -> b2 -> b3 -> b4 -> b5
	//        ↑_ _ _ _ _ _|

	blocks := make([]*Block, len(fun.f.Blocks))
	copy(blocks, fun.f.Blocks)

	// check loopRotate
	CheckFunc(fun.f)
	loopRotate(fun.f)
	CheckFunc(fun.f)

	expectedOrder := []ID{1, 2, 3, 4, 5}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("loopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}

	// check aggressiveLoopRotate
	// restore original block order
	fun.f.Blocks = blocks
	CheckFunc(fun.f)
	aggressiveLoopRotate(fun.f)
	CheckFunc(fun.f)

	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("aggressiveLoopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}
}

func TestLoopRotateSimple2(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("b1",
		Bloc("b1",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Goto("b2")),
		Bloc("b2",
			Valu("cond", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond", "b3", "b6")),
		Bloc("b3",
			Valu("cond1", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond1", "b4", "b5")),
		Bloc("b4",
			Goto("b2")),
		Bloc("b5",
			Goto("b2")),
		Bloc("b6",
			Exit("mem")),
	)

	//       |‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾|
	//       |      |‾ ‾ ‾ ‾ ‾↓    ↓
	// b1 -> b2 -> b3 -> b4   b5   b6
	//       ↑↑_ _ _ _ _ _|   |
	//       | _ _ _ _ _ _ _ _|

	// Transformed to:

	//
	//       ↓‾ ‾ ‾ ‾ ‾ ‾|
	// b1   b5 -> b2 -> b3 -> b4   b6
	//  |_ _ _ _ ↑|↑_ _ _ _ _ |    ↑
	//            |_ _ _ _ _ _ _ _ |

	blocks := make([]*Block, len(fun.f.Blocks))
	copy(blocks, fun.f.Blocks)

	// check loopRotate
	CheckFunc(fun.f)
	loopRotate(fun.f)
	CheckFunc(fun.f)

	expectedOrder := []ID{1, 5, 2, 3, 4, 6}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("loopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}

	// check aggressiveLoopRotate
	// restore original block order
	fun.f.Blocks = blocks
	CheckFunc(fun.f)
	aggressiveLoopRotate(fun.f)
	CheckFunc(fun.f)

	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("aggressiveLoopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}
}

func TestLoopRotateSimpleLoop3(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("b1",
		Bloc("b1",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Goto("b2")),
		Bloc("b2",
			Valu("cond", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond", "b3", "b5")),
		Bloc("b3",
			Goto("b4")),
		Bloc("b4",
			Goto("b2")),
		Bloc("b5",
			Exit("mem")),
	)

	//        ↓‾ ‾ ‾ ‾ ‾ ‾|
	// b1 -> b2 -> b3 -> b4    b5
	//        |_ _ _ _ _ _ _ _ _↑

	// Transformed to:

	//       ↓‾ ‾ ‾ ‾ ‾ ‾|
	// b1   b4 -> b2 -> b3    b5
	// |_ _ _ _ _ ↑|_ _ _ _ _ _↑

	blocks := make([]*Block, len(fun.f.Blocks))
	copy(blocks, fun.f.Blocks)

	// check loopRotate
	CheckFunc(fun.f)
	loopRotate(fun.f)
	CheckFunc(fun.f)

	expectedOrder := []ID{1, 4, 2, 3, 5}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("loopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}

	// check aggressiveLoopRotate
	// restore original block order
	fun.f.Blocks = blocks
	CheckFunc(fun.f)
	aggressiveLoopRotate(fun.f)
	CheckFunc(fun.f)

	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("aggressiveLoopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}
}

func TestLoopRotateNestLoop1(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("b1",
		Bloc("b1",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Goto("b2")),
		Bloc("b2",
			Goto("b3")),
		Bloc("b3",
			Goto("b4")),
		Bloc("b4",
			Valu("cond1", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond1", "b3", "b5")),
		Bloc("b5",
			Valu("cond2", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond2", "b2", "b6")),
		Bloc("b6",
			Exit("mem")),
	)

	//              ↓‾ ‾ ‾|
	// b1 -> b2 -> b3 -> b4 -> b5 -> b6
	//        ↑_ _ _ _ _ _ _ _ _|

	blocks := make([]*Block, len(fun.f.Blocks))
	copy(blocks, fun.f.Blocks)

	// check loopRotate
	CheckFunc(fun.f)
	loopRotate(fun.f)
	CheckFunc(fun.f)

	expectedOrder := []ID{1, 2, 3, 4, 5, 6}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("loopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}

	// check aggressiveLoopRotate
	// restore original block order
	fun.f.Blocks = blocks
	CheckFunc(fun.f)
	aggressiveLoopRotate(fun.f)
	CheckFunc(fun.f)

	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("aggressiveLoopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}
}

func TestLoopRotateNestLoop2(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("b1",
		Bloc("b1",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Goto("b2")),
		Bloc("b2",
			Valu("cond1", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond1", "b3", "b7")),
		Bloc("b3",
			Valu("cond2", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond2", "b4", "b5")),
		Bloc("b4",
			Goto("b3")),
		Bloc("b5",
			Goto("b6")),
		Bloc("b6",
			Goto("b2")),
		Bloc("b7",
			Exit("mem")),
	)

	//        |‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ |
	//        ↓     ↓‾ ‾ |           |
	// b1 -> b2 -> b3 -> b4   b5 -> b6   b7
	//        |     |_ _ _ _ _ ↑         ↑
	//        |_ _ _ _ _ _ _ _ _ _ _ _ _ |

	blocks := make([]*Block, len(fun.f.Blocks))
	copy(blocks, fun.f.Blocks)

	// check loopRotate
	CheckFunc(fun.f)
	loopRotate(fun.f)
	CheckFunc(fun.f)

	//        |‾ ‾ ‾ ‾ ‾ ||‾ ‾ ‾ ‾ ‾ |
	//        ↓          |↓   ↓‾ ‾ ‾||
	// b1    b6 -> b2    b5   b4 -> b3   b7
	//  |_ _ _ _ _↑||_ _ _ _ _ _ _ _ ↑   ↑
	//             |_ _ _ _ _ _ _ _ _ _ _|

	expectedOrder := []ID{1, 6, 2, 5, 4, 3, 7}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("loopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}

	// check aggressiveLoopRotate
	// restore original block order
	fun.f.Blocks = blocks
	CheckFunc(fun.f)
	aggressiveLoopRotate(fun.f)
	CheckFunc(fun.f)

	//       |‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾|
	//       ↓           ↓‾ ‾ ‾|     |
	// b1    b6 -> b2    b4 -> b3 -> b5   b7
	//  |_ _ _ _ _↑||_ _ _ _ _ ↑          ↑
	//             |_ _ _ _ _ _ _ _ _ _ _ |

	expectedOrder = []ID{1, 6, 2, 4, 3, 5, 7}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("aggressiveLoopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}
}

func TestLoopRotateNestLoop3(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("b1",
		Bloc("b1",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Goto("b2")),
		Bloc("b2",
			Valu("cond1", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond1", "b3", "b6")),
		Bloc("b3",
			Goto("b4")),
		Bloc("b4",
			Valu("cond2", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond2", "b3", "b5")),
		Bloc("b5",
			Goto("b2")),
		Bloc("b6",
			Exit("mem")),
	)

	//        |‾ ‾ ‾ ‾ ‾ ‾ ‾  ‾ |
	//        ↓     ↓‾ ‾ |      |
	// b1 -> b2 -> b3 -> b4 -> b5   b6
	//        |                     ↑
	//        |_ _ _ _ _ _ _ _ _ _ _|

	// Transformed to:

	//      |‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾|
	//      ↓            ↓‾ ‾ |
	// b1   b5 -> b2 -> b3 -> b4   b6
	// |          ↑|               ↑
	// | _ _ _ _ _||_ _ _ _ _ _ _ _|

	blocks := make([]*Block, len(fun.f.Blocks))
	copy(blocks, fun.f.Blocks)

	// check loopRotate
	CheckFunc(fun.f)
	loopRotate(fun.f)
	CheckFunc(fun.f)

	expectedOrder := []ID{1, 5, 2, 3, 4, 6}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("loopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}

	// check aggressiveLoopRotate
	// restore original block order
	fun.f.Blocks = blocks
	CheckFunc(fun.f)
	aggressiveLoopRotate(fun.f)
	CheckFunc(fun.f)

	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("aggressiveLoopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}
}

func TestLoopRotateNestLoop4(t *testing.T) {
	c := testConfig(t)
	fun := c.Fun("b1",
		Bloc("b1",
			Valu("mem", OpInitMem, types.TypeMem, 0, nil),
			Goto("b2")),
		Bloc("b2",
			Valu("cond1", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond1", "b3", "b9")),
		Bloc("b3",
			Goto("b4")),
		Bloc("b4",
			Valu("cond2", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond2", "b5", "b6")),
		Bloc("b5",
			Goto("b4")),
		Bloc("b6",
			Valu("cond3", OpConstBool, c.config.Types.Bool, 0, nil),
			If("cond3", "b8", "b7")),
		Bloc("b7",
			Goto("b2")),
		Bloc("b8",
			Goto("b10")),
		Bloc("b9",
			Goto("b10")),
		Bloc("b10",
			Exit("mem")),
	)

	//        |‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾|
	//        ↓          ↓‾ ‾ ‾|          |     |‾ ‾ ‾ ‾ ‾ ↓
	// b1 -> b2 -> b3 -> b4 -> b5   b6 -> b7   b8   b9 -> b10
	//        |          |_ _ _ _ _ ↑|_ _ _ _ _ ↑   ↑
	//        |_ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _|

	blocks := make([]*Block, len(fun.f.Blocks))
	copy(blocks, fun.f.Blocks)

	// check loopRotate
	CheckFunc(fun.f)
	loopRotate(fun.f)
	CheckFunc(fun.f)

	//                         |‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾|
	//                         ||‾ ‾ ‾ ‾ ‾ |   |
	//        ↓‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾||↓     ↓‾ ‾||   ↓|‾ ‾ ‾ ‾ ‾↓
	// b1    b7 -> b2 -> b3    b6   b5 -> b4   b8   b9 -> b10
	//  |_ _ _ _ _ ↑|     |_ _ _ _ _ _ _ _↑         ↑
	//              |_ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _|

	expectedOrder := []ID{1, 7, 2, 3, 6, 5, 4, 8, 9, 10}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("loopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}

	// Blocks b7 and b5 move before the corresponding loop headers b2 and b4.
	// Nested loop is kept inside its parent loop (isn't moved after b6).

	//        |‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾ ‾|
	//        ↓                  ↓‾ ‾ |     |     |‾ ‾ ‾ ‾ ‾↓
	// b1    b7 -> b2 -> b3    b5 -> b4 -> b6 -> b8   b9 -> b10
	//  |_ _ _ _ _ ↑|     |_ _ _ _ _ ↑                ↑
	//              |_ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _|

	// check aggressiveLoopRotate
	// restore original block order
	fun.f.Blocks = blocks
	CheckFunc(fun.f)
	aggressiveLoopRotate(fun.f)
	CheckFunc(fun.f)

	expectedOrder = []ID{1, 7, 2, 3, 5, 4, 6, 8, 9, 10}
	if !checkBlockOrder(expectedOrder, fun.f.Blocks) {
		t.Errorf("aggressiveLoopRotate: expected block order is %v, but got %v", expectedOrder, fun.f.Blocks)
	}
}
