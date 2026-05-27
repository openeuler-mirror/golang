// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa

import (
	"cmd/compile/internal/base"
	"cmd/compile/internal/types"
	"cmd/internal/src"
	"internal/buildcfg"
	"testing"
)

func TestDeadStore(t *testing.T) {
	c := testConfig(t)
	ptrType := c.config.Types.BytePtr
	t.Logf("PTRTYPE %v", ptrType)
	fun := c.Fun("entry",
		Bloc("entry",
			Valu("start", OpInitMem, types.TypeMem, 0, nil),
			Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
			Valu("v", OpConstBool, c.config.Types.Bool, 1, nil),
			Valu("addr1", OpAddr, ptrType, 0, nil, "sb"),
			Valu("addr2", OpAddr, ptrType, 0, nil, "sb"),
			Valu("addr3", OpAddr, ptrType, 0, nil, "sb"),
			Valu("zero1", OpZero, types.TypeMem, 1, c.config.Types.Bool, "addr3", "start"),
			Valu("store1", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr1", "v", "zero1"),
			Valu("store2", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr2", "v", "store1"),
			Valu("store3", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr1", "v", "store2"),
			Valu("store4", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr3", "v", "store3"),
			Goto("exit")),
		Bloc("exit",
			Exit("store3")))

	CheckFunc(fun.f)
	dse(fun.f)
	CheckFunc(fun.f)

	v1 := fun.values["store1"]
	if v1.Op != OpCopy {
		t.Errorf("dead store not removed")
	}

	v2 := fun.values["zero1"]
	if v2.Op != OpCopy {
		t.Errorf("dead store (zero) not removed")
	}
}

func TestDeadStorePhi(t *testing.T) {
	// make sure we don't get into an infinite loop with phi values.
	c := testConfig(t)
	ptrType := c.config.Types.BytePtr
	fun := c.Fun("entry",
		Bloc("entry",
			Valu("start", OpInitMem, types.TypeMem, 0, nil),
			Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
			Valu("v", OpConstBool, c.config.Types.Bool, 1, nil),
			Valu("addr", OpAddr, ptrType, 0, nil, "sb"),
			Goto("loop")),
		Bloc("loop",
			Valu("phi", OpPhi, types.TypeMem, 0, nil, "start", "store"),
			Valu("store", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr", "v", "phi"),
			If("v", "loop", "exit")),
		Bloc("exit",
			Exit("store")))

	CheckFunc(fun.f)
	dse(fun.f)
	CheckFunc(fun.f)
}

func TestDeadStoreTypes(t *testing.T) {
	// Make sure a narrow store can't shadow a wider one. We test an even
	// stronger restriction, that one store can't shadow another unless the
	// types of the address fields are identical (where identicalness is
	// decided by the CSE pass).
	c := testConfig(t)
	t1 := c.config.Types.UInt64.PtrTo()
	t2 := c.config.Types.UInt32.PtrTo()
	fun := c.Fun("entry",
		Bloc("entry",
			Valu("start", OpInitMem, types.TypeMem, 0, nil),
			Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
			Valu("v", OpConstBool, c.config.Types.Bool, 1, nil),
			Valu("addr1", OpAddr, t1, 0, nil, "sb"),
			Valu("addr2", OpAddr, t2, 0, nil, "sb"),
			Valu("store1", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr1", "v", "start"),
			Valu("store2", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr2", "v", "store1"),
			Goto("exit")),
		Bloc("exit",
			Exit("store2")))

	CheckFunc(fun.f)
	cse(fun.f)
	dse(fun.f)
	CheckFunc(fun.f)

	v := fun.values["store1"]
	if v.Op == OpCopy {
		t.Errorf("store %s incorrectly removed", v)
	}
}

func TestDeadStoreUnsafe(t *testing.T) {
	// Make sure a narrow store can't shadow a wider one. The test above
	// covers the case of two different types, but unsafe pointer casting
	// can get to a point where the size is changed but type unchanged.
	c := testConfig(t)
	ptrType := c.config.Types.UInt64.PtrTo()
	fun := c.Fun("entry",
		Bloc("entry",
			Valu("start", OpInitMem, types.TypeMem, 0, nil),
			Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
			Valu("v", OpConstBool, c.config.Types.Bool, 1, nil),
			Valu("addr1", OpAddr, ptrType, 0, nil, "sb"),
			Valu("store1", OpStore, types.TypeMem, 0, c.config.Types.Int64, "addr1", "v", "start"), // store 8 bytes
			Valu("store2", OpStore, types.TypeMem, 0, c.config.Types.Bool, "addr1", "v", "store1"), // store 1 byte
			Goto("exit")),
		Bloc("exit",
			Exit("store2")))

	CheckFunc(fun.f)
	cse(fun.f)
	dse(fun.f)
	CheckFunc(fun.f)

	v := fun.values["store1"]
	if v.Op == OpCopy {
		t.Errorf("store %s incorrectly removed", v)
	}
}

// withAggressiveDse toggles base.Flag.AggressiveDse for the duration of a
// test, restoring the prior value on return. Tests use this rather than
// setting the flag directly so that a panic or t.Fatal inside the test body
// cannot leak the flag to subsequent tests.
func withAggressiveDse(t *testing.T, on bool) {
	prev := base.Flag.AggressiveDse
	base.Flag.AggressiveDse = on
	t.Cleanup(func() { base.Flag.AggressiveDse = prev })
}

// TestAggressiveDseIntSignednessForward: int32 store forwards to uint32 load
// under -aggressivedse (copyCompatibleType accepts same-size int↔int), but
// NOT under the conservative rule (CMPeq fails on distinct signedness).
//
// After the (Load (Store ...)) rule rewrites ld to OpCopy, copy-input
// elimination rewrites sOut.Args[1] from ld to x, at which point ld.Uses==0
// and applyRewrite resets ld to OpInvalid. So the success check is on
// sOut.Args[1] == x, not on ld.Op itself.
func TestAggressiveDseIntSignednessForward(t *testing.T) {
	if buildcfg.GOARCH != "arm64" {
		t.Skip("aggressivedse tests only run on arm64")
	}
	c := testConfig(t)
	tt := c.config.Types
	build := func() fun {
		name := c.Temp(tt.Int32)
		nameOut := c.Temp(tt.UInt32)
		return c.Fun("entry",
			Bloc("entry",
				Valu("mem0", OpInitMem, types.TypeMem, 0, nil),
				Valu("sp", OpSP, tt.Uintptr, 0, nil),
				Valu("addr", OpLocalAddr, types.NewPtr(tt.Int32), 0, name, "sp", "mem0"),
				Valu("addrOut", OpLocalAddr, types.NewPtr(tt.UInt32), 0, nameOut, "sp", "mem0"),
				Valu("x", OpConst32, tt.Int32, 42, nil),
				Valu("s", OpStore, types.TypeMem, 0, tt.Int32, "addr", "x", "mem0"),
				Valu("ld", OpLoad, tt.UInt32, 0, nil, "addr", "s"),
				Valu("sOut", OpStore, types.TypeMem, 0, tt.UInt32, "addrOut", "ld", "s"),
				Goto("exit")),
			Bloc("exit",
				Exit("sOut")))
	}

	// Conservative: must NOT forward — sOut's stored value stays as ld.
	withAggressiveDse(t, false)
	fun1 := build()
	CheckFunc(fun1.f)
	opt(fun1.f)
	if sOut1 := fun1.values["sOut"]; sOut1.Args[1] == fun1.values["x"] {
		t.Errorf("conservative mode: load was forwarded but int32↔uint32 should fail CMPeq")
	}

	// Aggressive: must forward — sOut's stored value is rewritten to x directly.
	withAggressiveDse(t, true)
	fun2 := build()
	CheckFunc(fun2.f)
	opt(fun2.f)
	sOut2 := fun2.values["sOut"]
	if sOut2.Args[1] != fun2.values["x"] {
		t.Errorf("aggressive mode: expected sOut to store x directly, got %v", sOut2.Args[1])
	}
}

// TestAggressiveDseUnsafePtrForward: *int32 store forwards to *byte load
// under aggressive mode via the pointer-shaped branch of copyCompatibleType.
func TestAggressiveDseUnsafePtrForward(t *testing.T) {
	if buildcfg.GOARCH != "arm64" {
		t.Skip("aggressivedse tests only run on arm64")
	}
	c := testConfig(t)
	tt := c.config.Types
	int32Ptr := types.NewPtr(tt.Int32)
	build := func() fun {
		name := c.Temp(int32Ptr)
		nameOut := c.Temp(tt.BytePtr)
		return c.Fun("entry",
			Bloc("entry",
				Valu("mem0", OpInitMem, types.TypeMem, 0, nil),
				Valu("sp", OpSP, tt.Uintptr, 0, nil),
				Valu("addr", OpLocalAddr, types.NewPtr(int32Ptr), 0, name, "sp", "mem0"),
				Valu("addrOut", OpLocalAddr, types.NewPtr(tt.BytePtr), 0, nameOut, "sp", "mem0"),
				Valu("x", OpConstNil, int32Ptr, 0, nil),
				Valu("s", OpStore, types.TypeMem, 0, int32Ptr, "addr", "x", "mem0"),
				Valu("ld", OpLoad, tt.BytePtr, 0, nil, "addr", "s"),
				Valu("sOut", OpStore, types.TypeMem, 0, tt.BytePtr, "addrOut", "ld", "s"),
				Goto("exit")),
			Bloc("exit",
				Exit("sOut")))
	}

	withAggressiveDse(t, true)
	fun := build()
	CheckFunc(fun.f)
	opt(fun.f)
	sOut := fun.values["sOut"]
	if sOut.Args[1] != fun.values["x"] {
		t.Errorf("aggressive mode: expected sOut to store x directly, got %v", sOut.Args[1])
	}
}

// TestAggressiveDseNoFloatIntMix: a same-size float→int reinterpret must NOT
// be forwarded even under -aggressivedse, because bit patterns carry
// different semantic meaning.
func TestAggressiveDseNoFloatIntMix(t *testing.T) {
	if buildcfg.GOARCH != "arm64" {
		t.Skip("aggressivedse tests only run on arm64")
	}
	c := testConfig(t)
	tt := c.config.Types
	build := func() fun {
		name := c.Temp(tt.Float32)
		nameOut := c.Temp(tt.Int32)
		return c.Fun("entry",
			Bloc("entry",
				Valu("mem0", OpInitMem, types.TypeMem, 0, nil),
				Valu("sp", OpSP, tt.Uintptr, 0, nil),
				Valu("addr", OpLocalAddr, types.NewPtr(tt.Float32), 0, name, "sp", "mem0"),
				Valu("addrOut", OpLocalAddr, types.NewPtr(tt.Int32), 0, nameOut, "sp", "mem0"),
				Valu("x", OpConst32F, tt.Float32, 0, nil),
				Valu("s", OpStore, types.TypeMem, 0, tt.Float32, "addr", "x", "mem0"),
				Valu("ld", OpLoad, tt.Int32, 0, nil, "addr", "s"),
				Valu("sOut", OpStore, types.TypeMem, 0, tt.Int32, "addrOut", "ld", "s"),
				Goto("exit")),
			Bloc("exit",
				Exit("sOut")))
	}

	withAggressiveDse(t, true)
	fun := build()
	CheckFunc(fun.f)
	opt(fun.f)
	if sOut := fun.values["sOut"]; sOut.Args[1] == fun.values["x"] {
		t.Errorf("aggressive mode must not forward float→int reinterpret")
	}
}

// TestAggressiveDseSizeMismatchNoForward: even in aggressive mode, a size
// mismatch between load and source-store must block forwarding — this
// directly exercises the t1.Size() == t3.Size() check that replaced the
// upstream t1.Size() == t2.Size() in the nested rules.
func TestAggressiveDseSizeMismatchNoForward(t *testing.T) {
	if buildcfg.GOARCH != "arm64" {
		t.Skip("aggressivedse tests only run on arm64")
	}
	c := testConfig(t)
	tt := c.config.Types
	build := func() fun {
		name := c.Temp(tt.Int32)
		nameOut := c.Temp(tt.Int8)
		return c.Fun("entry",
			Bloc("entry",
				Valu("mem0", OpInitMem, types.TypeMem, 0, nil),
				Valu("sp", OpSP, tt.Uintptr, 0, nil),
				Valu("addr", OpLocalAddr, types.NewPtr(tt.Int32), 0, name, "sp", "mem0"),
				Valu("addrOut", OpLocalAddr, types.NewPtr(tt.Int8), 0, nameOut, "sp", "mem0"),
				Valu("x", OpConst32, tt.Int32, 0xDEAD, nil),
				Valu("s", OpStore, types.TypeMem, 0, tt.Int32, "addr", "x", "mem0"),
				// Narrow load at the same address — different size.
				Valu("ld", OpLoad, tt.Int8, 0, nil, "addr", "s"),
				Valu("sOut", OpStore, types.TypeMem, 0, tt.Int8, "addrOut", "ld", "s"),
				Goto("exit")),
			Bloc("exit",
				Exit("sOut")))
	}

	withAggressiveDse(t, true)
	fun := build()
	CheckFunc(fun.f)
	opt(fun.f)
	if sOut := fun.values["sOut"]; sOut.Args[1] == fun.values["x"] {
		t.Errorf("aggressive mode must not forward across size mismatch")
	}
}

func TestDeadStoreSmallStructInit(t *testing.T) {
	base.Flag.AggressiveDse = true
	defer func() {
		base.Flag.AggressiveDse = false
	}()
	c := testConfig(t)
	ptrType := c.config.Types.BytePtr
	typ := types.NewStruct([]*types.Field{
		types.NewField(src.NoXPos, &types.Sym{Name: "A"}, c.config.Types.Int),
		types.NewField(src.NoXPos, &types.Sym{Name: "B"}, c.config.Types.Int),
	})
	name := c.Temp(typ)
	fun := c.Fun("entry",
		Bloc("entry",
			Valu("start", OpInitMem, types.TypeMem, 0, nil),
			Valu("sp", OpSP, c.config.Types.Uintptr, 0, nil),
			Valu("zero", OpConst64, c.config.Types.Int, 0, nil),
			Valu("v6", OpLocalAddr, ptrType, 0, name, "sp", "start"),
			Valu("v3", OpOffPtr, ptrType, 8, nil, "v6"),
			Valu("v22", OpOffPtr, ptrType, 0, nil, "v6"),
			Valu("zerostore1", OpStore, types.TypeMem, 0, c.config.Types.Int, "v22", "zero", "start"),
			Valu("zerostore2", OpStore, types.TypeMem, 0, c.config.Types.Int, "v3", "zero", "zerostore1"),
			Valu("v8", OpLocalAddr, ptrType, 0, name, "sp", "zerostore2"),
			Valu("v23", OpOffPtr, ptrType, 8, nil, "v8"),
			Valu("v25", OpOffPtr, ptrType, 0, nil, "v8"),
			Valu("zerostore3", OpStore, types.TypeMem, 0, c.config.Types.Int, "v25", "zero", "zerostore2"),
			Valu("zerostore4", OpStore, types.TypeMem, 0, c.config.Types.Int, "v23", "zero", "zerostore3"),
			Goto("exit")),
		Bloc("exit",
			Exit("zerostore4")))

	fun.f.Name = "smallstructinit"
	CheckFunc(fun.f)
	cse(fun.f)
	dse(fun.f)
	CheckFunc(fun.f)

	v1 := fun.values["zerostore1"]
	if v1.Op != OpCopy {
		t.Errorf("dead store not removed")
	}
	v2 := fun.values["zerostore2"]
	if v2.Op != OpCopy {
		t.Errorf("dead store not removed")
	}
}
