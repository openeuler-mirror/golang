// cmd/7l/asm.c, cmd/7l/asmout.c, cmd/7l/optab.c, cmd/7l/span.c, cmd/ld/sub.c, cmd/ld/mod.c, from Vita Nuova.
// https://bitbucket.org/plan9-from-bell-labs/9-cc/src/master/
//
// 	Copyright © 1994-1999 Lucent Technologies Inc. All rights reserved.
// 	Portions Copyright © 1995-1997 C H Forsyth (forsyth@terzarima.net)
// 	Portions Copyright © 1997-1999 Vita Nuova Limited
// 	Portions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com)
// 	Portions Copyright © 2004,2006 Bruce Ellis
// 	Portions Copyright © 2005-2007 C H Forsyth (forsyth@terzarima.net)
// 	Revisions Copyright © 2000-2007 Lucent Technologies Inc. and others
// 	Portions Copyright © 2009 The Go Authors. All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package arm64

import (
	"cmd/internal/obj"
	"cmd/internal/objabi"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"slices"
	"strings"
)

// ctxt7 holds state while assembling a single function.
// Each function gets a fresh ctxt7.
// This allows for multiple functions to be safely concurrently assembled.
type ctxt7 struct {
	ctxt       *obj.Link
	newprog    obj.ProgAlloc
	cursym     *obj.LSym
	blitrl     *obj.Prog
	elitrl     *obj.Prog
	autosize   int32
	extrasize  int32
	instoffset int64
	pc         int64
	pool       struct {
		start uint32
		size  uint32
	}
}

const (
	funcAlign = 16
)

const (
	REGFROM = 1
)

type Optab struct {
	as    obj.As
	a1    uint8 // Prog.From
	a2    uint8 // 2nd source operand, Prog.Reg or Prog.RestArgs[XXX]
	a3    uint8 // 3rd source operand, Prog.RestArgs[XXX]
	a4    uint8 // Prog.To
	a5    uint8 // 2nd destination operand, Prog.RegTo2 or Prog.RestArgs[XXX]
	type_ int8
	size_ int8 // the value of this field is not static, use the size() method to return the value
	param int16
	flag  int8
	scond uint8
}

func IsAtomicInstruction(as obj.As) bool {
	if _, ok := atomicLDADD[as]; ok {
		return true
	}
	if _, ok := atomicSWP[as]; ok {
		return true
	}
	return false
}

// known field values of an instruction.
var atomicLDADD = map[obj.As]uint32{
	ALDADDAD:  3<<30 | 0x1c5<<21 | 0x00<<10,
	ALDADDAW:  2<<30 | 0x1c5<<21 | 0x00<<10,
	ALDADDAH:  1<<30 | 0x1c5<<21 | 0x00<<10,
	ALDADDAB:  0<<30 | 0x1c5<<21 | 0x00<<10,
	ALDADDALD: 3<<30 | 0x1c7<<21 | 0x00<<10,
	ALDADDALW: 2<<30 | 0x1c7<<21 | 0x00<<10,
	ALDADDALH: 1<<30 | 0x1c7<<21 | 0x00<<10,
	ALDADDALB: 0<<30 | 0x1c7<<21 | 0x00<<10,
	ALDADDD:   3<<30 | 0x1c1<<21 | 0x00<<10,
	ALDADDW:   2<<30 | 0x1c1<<21 | 0x00<<10,
	ALDADDH:   1<<30 | 0x1c1<<21 | 0x00<<10,
	ALDADDB:   0<<30 | 0x1c1<<21 | 0x00<<10,
	ALDADDLD:  3<<30 | 0x1c3<<21 | 0x00<<10,
	ALDADDLW:  2<<30 | 0x1c3<<21 | 0x00<<10,
	ALDADDLH:  1<<30 | 0x1c3<<21 | 0x00<<10,
	ALDADDLB:  0<<30 | 0x1c3<<21 | 0x00<<10,
	ALDCLRAD:  3<<30 | 0x1c5<<21 | 0x04<<10,
	ALDCLRAW:  2<<30 | 0x1c5<<21 | 0x04<<10,
	ALDCLRAH:  1<<30 | 0x1c5<<21 | 0x04<<10,
	ALDCLRAB:  0<<30 | 0x1c5<<21 | 0x04<<10,
	ALDCLRALD: 3<<30 | 0x1c7<<21 | 0x04<<10,
	ALDCLRALW: 2<<30 | 0x1c7<<21 | 0x04<<10,
	ALDCLRALH: 1<<30 | 0x1c7<<21 | 0x04<<10,
	ALDCLRALB: 0<<30 | 0x1c7<<21 | 0x04<<10,
	ALDCLRD:   3<<30 | 0x1c1<<21 | 0x04<<10,
	ALDCLRW:   2<<30 | 0x1c1<<21 | 0x04<<10,
	ALDCLRH:   1<<30 | 0x1c1<<21 | 0x04<<10,
	ALDCLRB:   0<<30 | 0x1c1<<21 | 0x04<<10,
	ALDCLRLD:  3<<30 | 0x1c3<<21 | 0x04<<10,
	ALDCLRLW:  2<<30 | 0x1c3<<21 | 0x04<<10,
	ALDCLRLH:  1<<30 | 0x1c3<<21 | 0x04<<10,
	ALDCLRLB:  0<<30 | 0x1c3<<21 | 0x04<<10,
	ALDEORAD:  3<<30 | 0x1c5<<21 | 0x08<<10,
	ALDEORAW:  2<<30 | 0x1c5<<21 | 0x08<<10,
	ALDEORAH:  1<<30 | 0x1c5<<21 | 0x08<<10,
	ALDEORAB:  0<<30 | 0x1c5<<21 | 0x08<<10,
	ALDEORALD: 3<<30 | 0x1c7<<21 | 0x08<<10,
	ALDEORALW: 2<<30 | 0x1c7<<21 | 0x08<<10,
	ALDEORALH: 1<<30 | 0x1c7<<21 | 0x08<<10,
	ALDEORALB: 0<<30 | 0x1c7<<21 | 0x08<<10,
	ALDEORD:   3<<30 | 0x1c1<<21 | 0x08<<10,
	ALDEORW:   2<<30 | 0x1c1<<21 | 0x08<<10,
	ALDEORH:   1<<30 | 0x1c1<<21 | 0x08<<10,
	ALDEORB:   0<<30 | 0x1c1<<21 | 0x08<<10,
	ALDEORLD:  3<<30 | 0x1c3<<21 | 0x08<<10,
	ALDEORLW:  2<<30 | 0x1c3<<21 | 0x08<<10,
	ALDEORLH:  1<<30 | 0x1c3<<21 | 0x08<<10,
	ALDEORLB:  0<<30 | 0x1c3<<21 | 0x08<<10,
	ALDORAD:   3<<30 | 0x1c5<<21 | 0x0c<<10,
	ALDORAW:   2<<30 | 0x1c5<<21 | 0x0c<<10,
	ALDORAH:   1<<30 | 0x1c5<<21 | 0x0c<<10,
	ALDORAB:   0<<30 | 0x1c5<<21 | 0x0c<<10,
	ALDORALD:  3<<30 | 0x1c7<<21 | 0x0c<<10,
	ALDORALW:  2<<30 | 0x1c7<<21 | 0x0c<<10,
	ALDORALH:  1<<30 | 0x1c7<<21 | 0x0c<<10,
	ALDORALB:  0<<30 | 0x1c7<<21 | 0x0c<<10,
	ALDORD:    3<<30 | 0x1c1<<21 | 0x0c<<10,
	ALDORW:    2<<30 | 0x1c1<<21 | 0x0c<<10,
	ALDORH:    1<<30 | 0x1c1<<21 | 0x0c<<10,
	ALDORB:    0<<30 | 0x1c1<<21 | 0x0c<<10,
	ALDORLD:   3<<30 | 0x1c3<<21 | 0x0c<<10,
	ALDORLW:   2<<30 | 0x1c3<<21 | 0x0c<<10,
	ALDORLH:   1<<30 | 0x1c3<<21 | 0x0c<<10,
	ALDORLB:   0<<30 | 0x1c3<<21 | 0x0c<<10,
}

var atomicSWP = map[obj.As]uint32{
	ASWPAD:  3<<30 | 0x1c5<<21 | 0x20<<10,
	ASWPAW:  2<<30 | 0x1c5<<21 | 0x20<<10,
	ASWPAH:  1<<30 | 0x1c5<<21 | 0x20<<10,
	ASWPAB:  0<<30 | 0x1c5<<21 | 0x20<<10,
	ASWPALD: 3<<30 | 0x1c7<<21 | 0x20<<10,
	ASWPALW: 2<<30 | 0x1c7<<21 | 0x20<<10,
	ASWPALH: 1<<30 | 0x1c7<<21 | 0x20<<10,
	ASWPALB: 0<<30 | 0x1c7<<21 | 0x20<<10,
	ASWPD:   3<<30 | 0x1c1<<21 | 0x20<<10,
	ASWPW:   2<<30 | 0x1c1<<21 | 0x20<<10,
	ASWPH:   1<<30 | 0x1c1<<21 | 0x20<<10,
	ASWPB:   0<<30 | 0x1c1<<21 | 0x20<<10,
	ASWPLD:  3<<30 | 0x1c3<<21 | 0x20<<10,
	ASWPLW:  2<<30 | 0x1c3<<21 | 0x20<<10,
	ASWPLH:  1<<30 | 0x1c3<<21 | 0x20<<10,
	ASWPLB:  0<<30 | 0x1c3<<21 | 0x20<<10,
	ACASD:   3<<30 | 0x45<<21 | 0x1f<<10,
	ACASW:   2<<30 | 0x45<<21 | 0x1f<<10,
	ACASH:   1<<30 | 0x45<<21 | 0x1f<<10,
	ACASB:   0<<30 | 0x45<<21 | 0x1f<<10,
	ACASAD:  3<<30 | 0x47<<21 | 0x1f<<10,
	ACASAW:  2<<30 | 0x47<<21 | 0x1f<<10,
	ACASLD:  3<<30 | 0x45<<21 | 0x3f<<10,
	ACASLW:  2<<30 | 0x45<<21 | 0x3f<<10,
	ACASALD: 3<<30 | 0x47<<21 | 0x3f<<10,
	ACASALW: 2<<30 | 0x47<<21 | 0x3f<<10,
	ACASALH: 1<<30 | 0x47<<21 | 0x3f<<10,
	ACASALB: 0<<30 | 0x47<<21 | 0x3f<<10,
}
var atomicCASP = map[obj.As]uint32{
	ACASPD: 1<<30 | 0x41<<21 | 0x1f<<10,
	ACASPW: 0<<30 | 0x41<<21 | 0x1f<<10,
}

var oprange [ALAST & obj.AMask][]Optab

var xcmp [C_NCLASS][C_NCLASS]bool

const (
	S32     = 0 << 31
	S64     = 1 << 31
	Sbit    = 1 << 29
	LSL0_32 = 2 << 13
	LSL0_64 = 3 << 13
)

func OPDP2(x uint32) uint32 {
	return 0<<30 | 0<<29 | 0xd6<<21 | x<<10
}

func OPDP3(sf uint32, op54 uint32, op31 uint32, o0 uint32) uint32 {
	return sf<<31 | op54<<29 | 0x1B<<24 | op31<<21 | o0<<15
}

func OPBcc(x uint32) uint32 {
	return 0x2A<<25 | 0<<24 | 0<<4 | x&15
}

func OPBLR(x uint32) uint32 {
	/* x=0, JMP; 1, CALL; 2, RET */
	return 0x6B<<25 | 0<<23 | x<<21 | 0x1F<<16 | 0<<10
}

func SYSOP(l uint32, op0 uint32, op1 uint32, crn uint32, crm uint32, op2 uint32, rt uint32) uint32 {
	return 0x354<<22 | l<<21 | op0<<19 | op1<<16 | crn&15<<12 | crm&15<<8 | op2<<5 | rt
}

func SYSHINT(x uint32) uint32 {
	return SYSOP(0, 0, 3, 2, 0, x, 0x1F)
}

func LDSTR(sz uint32, v uint32, opc uint32) uint32 {
	return sz<<30 | 7<<27 | v<<26 | opc<<22
}

func LD2STR(o uint32) uint32 {
	return o &^ (3 << 22)
}

func LDSTX(sz uint32, o2 uint32, l uint32, o1 uint32, o0 uint32) uint32 {
	return sz<<30 | 0x8<<24 | o2<<23 | l<<22 | o1<<21 | o0<<15
}

func LDAPR(sz uint32) uint32 {
	return sz<<30 | 7<<27 | 0x2FF0<<10
}

func FPCMP(m uint32, s uint32, type_ uint32, op uint32, op2 uint32) uint32 {
	return m<<31 | s<<29 | 0x1E<<24 | type_<<22 | 1<<21 | op<<14 | 8<<10 | op2
}

func FPCCMP(m uint32, s uint32, type_ uint32, op uint32) uint32 {
	return m<<31 | s<<29 | 0x1E<<24 | type_<<22 | 1<<21 | 1<<10 | op<<4
}

func FPOP1S(m uint32, s uint32, type_ uint32, op uint32) uint32 {
	return m<<31 | s<<29 | 0x1E<<24 | type_<<22 | 1<<21 | op<<15 | 0x10<<10
}

func FPOP2S(m uint32, s uint32, type_ uint32, op uint32) uint32 {
	return m<<31 | s<<29 | 0x1E<<24 | type_<<22 | 1<<21 | op<<12 | 2<<10
}

func FPOP3S(m uint32, s uint32, type_ uint32, op uint32, op2 uint32) uint32 {
	return m<<31 | s<<29 | 0x1F<<24 | type_<<22 | op<<21 | op2<<15
}

func FPCVTI(sf uint32, s uint32, type_ uint32, rmode uint32, op uint32) uint32 {
	return sf<<31 | s<<29 | 0x1E<<24 | type_<<22 | 1<<21 | rmode<<19 | op<<16 | 0<<10
}

func ADR(p uint32, o uint32, rt uint32) uint32 {
	return p<<31 | (o&3)<<29 | 0x10<<24 | ((o>>2)&0x7FFFF)<<5 | rt&31
}

func OPBIT(x uint32) uint32 {
	return 1<<30 | 0<<29 | 0xD6<<21 | 0<<16 | x<<10
}

func MOVCONST(d int64, s int, rt int) uint32 {
	return uint32(((d>>uint(s*16))&0xFFFF)<<5) | uint32(s)&3<<21 | uint32(rt&31)
}

const (
	// Optab.flag
	LFROM        = 1 << iota // p.From uses constant pool
	LTO                      // p.To uses constant pool
	NOTUSETMP                // p expands to multiple instructions, but does NOT use REGTMP
	BRANCH14BITS             // branch instruction encodes 14 bits
	BRANCH19BITS             // branch instruction encodes 19 bits
	SVE
)

var optab = []Optab{
	/* struct Optab:
	OPCODE, from, prog->reg, from3, to, to2, type,size,param,flag,scond */
	{obj.ATEXT, C_ADDR, C_NONE, C_NONE, C_TEXTSIZE, C_NONE, 0, 0, 0, 0, 0},

	/* arithmetic operations */
	{AADD, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AADD, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AADC, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AADC, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{ANEG, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 25, 4, 0, 0, 0},
	{ANEG, C_NONE, C_NONE, C_NONE, C_REGZR, C_NONE, 25, 4, 0, 0, 0},
	{ANGC, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 17, 4, 0, 0, 0},
	{ACMP, C_REGZR, C_REGZR, C_NONE, C_NONE, C_NONE, 1, 4, 0, 0, 0},
	{AADD, C_ADDCON, C_RSP, C_NONE, C_RSP, C_NONE, 2, 4, 0, 0, 0},
	{AADD, C_ADDCON, C_NONE, C_NONE, C_RSP, C_NONE, 2, 4, 0, 0, 0},
	{ACMP, C_ADDCON, C_RSP, C_NONE, C_NONE, C_NONE, 2, 4, 0, 0, 0},
	{AADD, C_MOVCON, C_RSP, C_NONE, C_RSP, C_NONE, 62, 8, 0, 0, 0},
	{AADD, C_MOVCON, C_NONE, C_NONE, C_RSP, C_NONE, 62, 8, 0, 0, 0},
	{ACMP, C_MOVCON, C_RSP, C_NONE, C_NONE, C_NONE, 62, 8, 0, 0, 0},
	{AADD, C_BITCON, C_RSP, C_NONE, C_RSP, C_NONE, 62, 8, 0, 0, 0},
	{AADD, C_BITCON, C_NONE, C_NONE, C_RSP, C_NONE, 62, 8, 0, 0, 0},
	{ACMP, C_BITCON, C_RSP, C_NONE, C_NONE, C_NONE, 62, 8, 0, 0, 0},
	{AADD, C_ADDCON2, C_RSP, C_NONE, C_RSP, C_NONE, 48, 8, 0, NOTUSETMP, 0},
	{AADD, C_ADDCON2, C_NONE, C_NONE, C_RSP, C_NONE, 48, 8, 0, NOTUSETMP, 0},
	{AADD, C_MOVCON2, C_RSP, C_NONE, C_RSP, C_NONE, 13, 12, 0, 0, 0},
	{AADD, C_MOVCON2, C_NONE, C_NONE, C_RSP, C_NONE, 13, 12, 0, 0, 0},
	{AADD, C_MOVCON3, C_RSP, C_NONE, C_RSP, C_NONE, 13, 16, 0, 0, 0},
	{AADD, C_MOVCON3, C_NONE, C_NONE, C_RSP, C_NONE, 13, 16, 0, 0, 0},
	{AADD, C_VCON, C_RSP, C_NONE, C_RSP, C_NONE, 13, 20, 0, 0, 0},
	{AADD, C_VCON, C_NONE, C_NONE, C_RSP, C_NONE, 13, 20, 0, 0, 0},
	{ACMP, C_MOVCON2, C_REGZR, C_NONE, C_NONE, C_NONE, 13, 12, 0, 0, 0},
	{ACMP, C_MOVCON3, C_REGZR, C_NONE, C_NONE, C_NONE, 13, 16, 0, 0, 0},
	{ACMP, C_VCON, C_REGZR, C_NONE, C_NONE, C_NONE, 13, 20, 0, 0, 0},
	{AADD, C_SHIFT, C_REGZR, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{AADD, C_SHIFT, C_NONE, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{AMVN, C_SHIFT, C_NONE, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{ACMP, C_SHIFT, C_REGZR, C_NONE, C_NONE, C_NONE, 3, 4, 0, 0, 0},
	{ANEG, C_SHIFT, C_NONE, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{AADD, C_REGZR, C_RSP, C_NONE, C_RSP, C_NONE, 27, 4, 0, 0, 0},
	{AADD, C_REGZR, C_NONE, C_NONE, C_RSP, C_NONE, 27, 4, 0, 0, 0},
	{ACMP, C_REGZR, C_RSP, C_NONE, C_NONE, C_NONE, 27, 4, 0, 0, 0},
	{AADD, C_EXTREG, C_RSP, C_NONE, C_RSP, C_NONE, 27, 4, 0, 0, 0},
	{AADD, C_EXTREG, C_NONE, C_NONE, C_RSP, C_NONE, 27, 4, 0, 0, 0},
	{ACMP, C_EXTREG, C_RSP, C_NONE, C_NONE, C_NONE, 27, 4, 0, 0, 0},
	{AADD, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AADD, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AMUL, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 15, 4, 0, 0, 0},
	{AMUL, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 15, 4, 0, 0, 0},
	{AMADD, C_REGZR, C_REGZR, C_REGZR, C_REGZR, C_NONE, 15, 4, 0, 0, 0},
	{AREM, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 16, 8, 0, 0, 0},
	{AREM, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 16, 8, 0, 0, 0},
	{ASDIV, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{ASDIV, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},

	{AFADDS, C_FREG, C_NONE, C_NONE, C_FREG, C_NONE, 54, 4, 0, 0, 0},
	{AFADDS, C_FREG, C_FREG, C_NONE, C_FREG, C_NONE, 54, 4, 0, 0, 0},
	{AFMSUBD, C_FREG, C_FREG, C_FREG, C_FREG, C_NONE, 15, 4, 0, 0, 0},
	{AFCMPS, C_FREG, C_FREG, C_NONE, C_NONE, C_NONE, 56, 4, 0, 0, 0},
	{AFCMPS, C_FCON, C_FREG, C_NONE, C_NONE, C_NONE, 56, 4, 0, 0, 0},
	{AVADDP, C_ARNG, C_ARNG, C_NONE, C_ARNG, C_NONE, 72, 4, 0, 0, 0},
	{AVADD, C_ARNG, C_ARNG, C_NONE, C_ARNG, C_NONE, 72, 4, 0, 0, 0},
	{AVADD, C_VREG, C_VREG, C_NONE, C_VREG, C_NONE, 89, 4, 0, 0, 0},
	{AVADD, C_VREG, C_NONE, C_NONE, C_VREG, C_NONE, 89, 4, 0, 0, 0},
	{AVADDV, C_ARNG, C_NONE, C_NONE, C_VREG, C_NONE, 85, 4, 0, 0, 0},

	/* logical operations */
	{AAND, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AAND, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AANDS, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{AANDS, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 1, 4, 0, 0, 0},
	{ATST, C_REGZR, C_REGZR, C_NONE, C_NONE, C_NONE, 1, 4, 0, 0, 0},
	{AAND, C_MBCON, C_REGZR, C_NONE, C_RSP, C_NONE, 53, 4, 0, 0, 0},
	{AAND, C_MBCON, C_NONE, C_NONE, C_RSP, C_NONE, 53, 4, 0, 0, 0},
	{AANDS, C_MBCON, C_REGZR, C_NONE, C_REGZR, C_NONE, 53, 4, 0, 0, 0},
	{AANDS, C_MBCON, C_NONE, C_NONE, C_REGZR, C_NONE, 53, 4, 0, 0, 0},
	{ATST, C_MBCON, C_REGZR, C_NONE, C_NONE, C_NONE, 53, 4, 0, 0, 0},
	{AAND, C_BITCON, C_REGZR, C_NONE, C_RSP, C_NONE, 53, 4, 0, 0, 0},
	{AAND, C_BITCON, C_NONE, C_NONE, C_RSP, C_NONE, 53, 4, 0, 0, 0},
	{AANDS, C_BITCON, C_REGZR, C_NONE, C_REGZR, C_NONE, 53, 4, 0, 0, 0},
	{AANDS, C_BITCON, C_NONE, C_NONE, C_REGZR, C_NONE, 53, 4, 0, 0, 0},
	{ATST, C_BITCON, C_REGZR, C_NONE, C_NONE, C_NONE, 53, 4, 0, 0, 0},
	{AAND, C_MOVCON, C_REGZR, C_NONE, C_REGZR, C_NONE, 62, 8, 0, 0, 0},
	{AAND, C_MOVCON, C_NONE, C_NONE, C_REGZR, C_NONE, 62, 8, 0, 0, 0},
	{AANDS, C_MOVCON, C_REGZR, C_NONE, C_REGZR, C_NONE, 62, 8, 0, 0, 0},
	{AANDS, C_MOVCON, C_NONE, C_NONE, C_REGZR, C_NONE, 62, 8, 0, 0, 0},
	{ATST, C_MOVCON, C_REGZR, C_NONE, C_NONE, C_NONE, 62, 8, 0, 0, 0},
	{AAND, C_MOVCON2, C_REGZR, C_NONE, C_REGZR, C_NONE, 28, 12, 0, 0, 0},
	{AAND, C_MOVCON2, C_NONE, C_NONE, C_REGZR, C_NONE, 28, 12, 0, 0, 0},
	{AAND, C_MOVCON3, C_REGZR, C_NONE, C_REGZR, C_NONE, 28, 16, 0, 0, 0},
	{AAND, C_MOVCON3, C_NONE, C_NONE, C_REGZR, C_NONE, 28, 16, 0, 0, 0},
	{AAND, C_VCON, C_REGZR, C_NONE, C_REGZR, C_NONE, 28, 20, 0, 0, 0},
	{AAND, C_VCON, C_NONE, C_NONE, C_REGZR, C_NONE, 28, 20, 0, 0, 0},
	{AANDS, C_MOVCON2, C_REGZR, C_NONE, C_REGZR, C_NONE, 28, 12, 0, 0, 0},
	{AANDS, C_MOVCON2, C_NONE, C_NONE, C_REGZR, C_NONE, 28, 12, 0, 0, 0},
	{AANDS, C_MOVCON3, C_REGZR, C_NONE, C_REGZR, C_NONE, 28, 16, 0, 0, 0},
	{AANDS, C_MOVCON3, C_NONE, C_NONE, C_REGZR, C_NONE, 28, 16, 0, 0, 0},
	{AANDS, C_VCON, C_REGZR, C_NONE, C_REGZR, C_NONE, 28, 20, 0, 0, 0},
	{AANDS, C_VCON, C_NONE, C_NONE, C_REGZR, C_NONE, 28, 20, 0, 0, 0},
	{ATST, C_MOVCON2, C_REGZR, C_NONE, C_NONE, C_NONE, 28, 12, 0, 0, 0},
	{ATST, C_MOVCON3, C_REGZR, C_NONE, C_NONE, C_NONE, 28, 16, 0, 0, 0},
	{ATST, C_VCON, C_REGZR, C_NONE, C_NONE, C_NONE, 28, 20, 0, 0, 0},
	{AAND, C_SHIFT, C_REGZR, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{AAND, C_SHIFT, C_NONE, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{AANDS, C_SHIFT, C_REGZR, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{AANDS, C_SHIFT, C_NONE, C_NONE, C_REGZR, C_NONE, 3, 4, 0, 0, 0},
	{ATST, C_SHIFT, C_REGZR, C_NONE, C_NONE, C_NONE, 3, 4, 0, 0, 0},
	{AMOVD, C_RSP, C_NONE, C_NONE, C_RSP, C_NONE, 24, 4, 0, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 24, 4, 0, 0, 0},
	{AMVN, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 24, 4, 0, 0, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 45, 4, 0, 0, 0}, /* also MOVBU */
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 45, 4, 0, 0, 0}, /* also MOVHU */
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 45, 4, 0, 0, 0}, /* also MOVWU */
	/* TODO: MVN C_SHIFT */

	/* MOVs that become MOVK/MOVN/MOVZ/ADD/SUB/OR */
	{AMOVW, C_MBCON, C_NONE, C_NONE, C_REGZR, C_NONE, 32, 4, 0, 0, 0},
	{AMOVD, C_MBCON, C_NONE, C_NONE, C_REGZR, C_NONE, 32, 4, 0, 0, 0},
	{AMOVW, C_MOVCON, C_NONE, C_NONE, C_REGZR, C_NONE, 32, 4, 0, 0, 0},
	{AMOVD, C_MOVCON, C_NONE, C_NONE, C_REGZR, C_NONE, 32, 4, 0, 0, 0},
	{AMOVW, C_BITCON, C_NONE, C_NONE, C_RSP, C_NONE, 32, 4, 0, 0, 0},
	{AMOVD, C_BITCON, C_NONE, C_NONE, C_RSP, C_NONE, 32, 4, 0, 0, 0},
	{AMOVW, C_MOVCON2, C_NONE, C_NONE, C_REGZR, C_NONE, 12, 8, 0, NOTUSETMP, 0},
	{AMOVD, C_MOVCON2, C_NONE, C_NONE, C_REGZR, C_NONE, 12, 8, 0, NOTUSETMP, 0},
	{AMOVD, C_MOVCON3, C_NONE, C_NONE, C_REGZR, C_NONE, 12, 12, 0, NOTUSETMP, 0},
	{AMOVD, C_VCON, C_NONE, C_NONE, C_REGZR, C_NONE, 12, 16, 0, NOTUSETMP, 0},

	{AMOVK, C_VCON, C_NONE, C_NONE, C_REGZR, C_NONE, 33, 4, 0, 0, 0},
	{AMOVD, C_AACON, C_NONE, C_NONE, C_RSP, C_NONE, 4, 4, REGFROM, 0, 0},
	{AMOVD, C_AACON2, C_NONE, C_NONE, C_RSP, C_NONE, 4, 8, REGFROM, NOTUSETMP, 0},

	/* load long effective stack address (load int32 offset and add) */
	{AMOVD, C_LACON, C_NONE, C_NONE, C_RSP, C_NONE, 34, 8, REGSP, LFROM, 0},

	// Load a large constant into a vector register.
	{AVMOVS, C_ADDR, C_NONE, C_NONE, C_VREG, C_NONE, 65, 12, 0, 0, 0},
	{AVMOVD, C_ADDR, C_NONE, C_NONE, C_VREG, C_NONE, 65, 12, 0, 0, 0},
	{AVMOVQ, C_ADDR, C_NONE, C_NONE, C_VREG, C_NONE, 65, 12, 0, 0, 0},

	/* jump operations */
	{AB, C_NONE, C_NONE, C_NONE, C_SBRA, C_NONE, 5, 4, 0, 0, 0},
	{ABL, C_NONE, C_NONE, C_NONE, C_SBRA, C_NONE, 5, 4, 0, 0, 0},
	{AB, C_NONE, C_NONE, C_NONE, C_ZOREG, C_NONE, 6, 4, 0, 0, 0},
	{ABL, C_NONE, C_NONE, C_NONE, C_REGZR, C_NONE, 6, 4, 0, 0, 0},
	{ABL, C_NONE, C_NONE, C_NONE, C_ZOREG, C_NONE, 6, 4, 0, 0, 0},
	{obj.ARET, C_NONE, C_NONE, C_NONE, C_REGZR, C_NONE, 6, 4, 0, 0, 0},
	{obj.ARET, C_NONE, C_NONE, C_NONE, C_ZOREG, C_NONE, 6, 4, 0, 0, 0},
	{ABEQ, C_NONE, C_NONE, C_NONE, C_SBRA, C_NONE, 7, 4, 0, BRANCH19BITS, 0},
	{ACBZ, C_REGZR, C_NONE, C_NONE, C_SBRA, C_NONE, 39, 4, 0, BRANCH19BITS, 0},
	{ATBZ, C_VCON, C_REGZR, C_NONE, C_SBRA, C_NONE, 40, 4, 0, BRANCH14BITS, 0},
	{AERET, C_NONE, C_NONE, C_NONE, C_NONE, C_NONE, 41, 4, 0, 0, 0},

	// get a PC-relative address
	{AADRP, C_SBRA, C_NONE, C_NONE, C_REGZR, C_NONE, 60, 4, 0, 0, 0},
	{AADR, C_SBRA, C_NONE, C_NONE, C_REGZR, C_NONE, 61, 4, 0, 0, 0},

	{ACLREX, C_NONE, C_NONE, C_NONE, C_VCON, C_NONE, 38, 4, 0, 0, 0},
	{ACLREX, C_NONE, C_NONE, C_NONE, C_NONE, C_NONE, 38, 4, 0, 0, 0},
	{ABFM, C_VCON, C_REGZR, C_VCON, C_REGZR, C_NONE, 42, 4, 0, 0, 0},
	{ABFI, C_VCON, C_REGZR, C_VCON, C_REGZR, C_NONE, 43, 4, 0, 0, 0},
	{AEXTR, C_VCON, C_REGZR, C_REGZR, C_REGZR, C_NONE, 44, 4, 0, 0, 0},
	{ASXTB, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 45, 4, 0, 0, 0},
	{ACLS, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 46, 4, 0, 0, 0},
	{ALSL, C_VCON, C_REGZR, C_NONE, C_REGZR, C_NONE, 8, 4, 0, 0, 0},
	{ALSL, C_VCON, C_NONE, C_NONE, C_REGZR, C_NONE, 8, 4, 0, 0, 0},
	{ALSL, C_REGZR, C_NONE, C_NONE, C_REGZR, C_NONE, 9, 4, 0, 0, 0},
	{ALSL, C_REGZR, C_REGZR, C_NONE, C_REGZR, C_NONE, 9, 4, 0, 0, 0},
	{ASVC, C_VCON, C_NONE, C_NONE, C_NONE, C_NONE, 10, 4, 0, 0, 0},
	{ASVC, C_NONE, C_NONE, C_NONE, C_NONE, C_NONE, 10, 4, 0, 0, 0},
	{ADWORD, C_NONE, C_NONE, C_NONE, C_VCON, C_NONE, 11, 8, 0, NOTUSETMP, 0},
	{ADWORD, C_NONE, C_NONE, C_NONE, C_LEXT, C_NONE, 11, 8, 0, NOTUSETMP, 0},
	{ADWORD, C_NONE, C_NONE, C_NONE, C_ADDR, C_NONE, 11, 8, 0, NOTUSETMP, 0},
	{ADWORD, C_NONE, C_NONE, C_NONE, C_LACON, C_NONE, 11, 8, 0, NOTUSETMP, 0},
	{AWORD, C_NONE, C_NONE, C_NONE, C_LCON, C_NONE, 14, 4, 0, 0, 0},
	{AWORD, C_NONE, C_NONE, C_NONE, C_LEXT, C_NONE, 14, 4, 0, 0, 0},
	{AWORD, C_NONE, C_NONE, C_NONE, C_ADDR, C_NONE, 14, 4, 0, 0, 0},
	{AMOVW, C_VCONADDR, C_NONE, C_NONE, C_REGZR, C_NONE, 68, 8, 0, NOTUSETMP, 0},
	{AMOVD, C_VCONADDR, C_NONE, C_NONE, C_REGZR, C_NONE, 68, 8, 0, NOTUSETMP, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_ADDR, C_NONE, 64, 12, 0, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_ADDR, C_NONE, 64, 12, 0, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_ADDR, C_NONE, 64, 12, 0, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_ADDR, C_NONE, 64, 12, 0, 0, 0},
	{AMOVB, C_ADDR, C_NONE, C_NONE, C_REGZR, C_NONE, 65, 12, 0, 0, 0},
	{AMOVH, C_ADDR, C_NONE, C_NONE, C_REGZR, C_NONE, 65, 12, 0, 0, 0},
	{AMOVW, C_ADDR, C_NONE, C_NONE, C_REGZR, C_NONE, 65, 12, 0, 0, 0},
	{AMOVD, C_ADDR, C_NONE, C_NONE, C_REGZR, C_NONE, 65, 12, 0, 0, 0},
	{AMOVD, C_GOTADDR, C_NONE, C_NONE, C_REGZR, C_NONE, 71, 8, 0, 0, 0},
	{AMOVD, C_TLS_LE, C_NONE, C_NONE, C_REGZR, C_NONE, 69, 4, 0, 0, 0},
	{AMOVD, C_TLS_IE, C_NONE, C_NONE, C_REGZR, C_NONE, 70, 8, 0, 0, 0},

	{AFMOVS, C_FREG, C_NONE, C_NONE, C_ADDR, C_NONE, 64, 12, 0, 0, 0},
	{AFMOVS, C_ADDR, C_NONE, C_NONE, C_FREG, C_NONE, 65, 12, 0, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_ADDR, C_NONE, 64, 12, 0, 0, 0},
	{AFMOVD, C_ADDR, C_NONE, C_NONE, C_FREG, C_NONE, 65, 12, 0, 0, 0},
	{AFMOVS, C_FCON, C_NONE, C_NONE, C_FREG, C_NONE, 55, 4, 0, 0, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_FREG, C_NONE, 54, 4, 0, 0, 0},
	{AFMOVD, C_FCON, C_NONE, C_NONE, C_FREG, C_NONE, 55, 4, 0, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_FREG, C_NONE, 54, 4, 0, 0, 0},
	{AFMOVS, C_REGZR, C_NONE, C_NONE, C_FREG, C_NONE, 29, 4, 0, 0, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_REGZR, C_NONE, 29, 4, 0, 0, 0},
	{AFMOVD, C_REGZR, C_NONE, C_NONE, C_FREG, C_NONE, 29, 4, 0, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_REGZR, C_NONE, 29, 4, 0, 0, 0},
	{AFCVTZSD, C_FREG, C_NONE, C_NONE, C_REGZR, C_NONE, 29, 4, 0, 0, 0},
	{ASCVTFD, C_REGZR, C_NONE, C_NONE, C_FREG, C_NONE, 29, 4, 0, 0, 0},
	{AFCVTSD, C_FREG, C_NONE, C_NONE, C_FREG, C_NONE, 29, 4, 0, 0, 0},
	{AVMOV, C_ELEM, C_NONE, C_NONE, C_REGZR, C_NONE, 73, 4, 0, 0, 0},
	{AVMOV, C_ELEM, C_NONE, C_NONE, C_ELEM, C_NONE, 92, 4, 0, 0, 0},
	{AVMOV, C_ELEM, C_NONE, C_NONE, C_VREG, C_NONE, 80, 4, 0, 0, 0},
	{AVMOV, C_REGZR, C_NONE, C_NONE, C_ARNG, C_NONE, 82, 4, 0, 0, 0},
	{AVMOV, C_REGZR, C_NONE, C_NONE, C_ELEM, C_NONE, 78, 4, 0, 0, 0},
	{AVMOV, C_ARNG, C_NONE, C_NONE, C_ARNG, C_NONE, 83, 4, 0, 0, 0},
	{AVDUP, C_ELEM, C_NONE, C_NONE, C_ARNG, C_NONE, 79, 4, 0, 0, 0},
	{AVDUP, C_ELEM, C_NONE, C_NONE, C_VREG, C_NONE, 80, 4, 0, 0, 0},
	{AVDUP, C_REGZR, C_NONE, C_NONE, C_ARNG, C_NONE, 82, 4, 0, 0, 0},
	{AVMOVI, C_ADDCON, C_NONE, C_NONE, C_ARNG, C_NONE, 86, 4, 0, 0, 0},
	{AVFMLA, C_ARNG, C_ARNG, C_NONE, C_ARNG, C_NONE, 72, 4, 0, 0, 0},
	{AVEXT, C_VCON, C_ARNG, C_ARNG, C_ARNG, C_NONE, 94, 4, 0, 0, 0},
	{AVTBL, C_ARNG, C_NONE, C_LIST, C_ARNG, C_NONE, 100, 4, 0, 0, 0},
	{AVUSHR, C_VCON, C_ARNG, C_NONE, C_ARNG, C_NONE, 95, 4, 0, 0, 0},
	{AVZIP1, C_ARNG, C_ARNG, C_NONE, C_ARNG, C_NONE, 72, 4, 0, 0, 0},
	{AVUSHLL, C_VCON, C_ARNG, C_NONE, C_ARNG, C_NONE, 102, 4, 0, 0, 0},
	{AVUXTL, C_ARNG, C_NONE, C_NONE, C_ARNG, C_NONE, 102, 4, 0, 0, 0},
	{AVUADDW, C_ARNG, C_ARNG, C_NONE, C_ARNG, C_NONE, 105, 4, 0, 0, 0},

	/* conditional operations */
	{ACSEL, C_COND, C_REGZR, C_REGZR, C_REGZR, C_NONE, 18, 4, 0, 0, 0},
	{ACINC, C_COND, C_REGZR, C_NONE, C_REGZR, C_NONE, 18, 4, 0, 0, 0},
	{ACSET, C_COND, C_NONE, C_NONE, C_REGZR, C_NONE, 18, 4, 0, 0, 0},
	{AFCSELD, C_COND, C_FREG, C_FREG, C_FREG, C_NONE, 18, 4, 0, 0, 0},
	{ACCMN, C_COND, C_REGZR, C_REGZR, C_VCON, C_NONE, 19, 4, 0, 0, 0},
	{ACCMN, C_COND, C_REGZR, C_VCON, C_VCON, C_NONE, 19, 4, 0, 0, 0},
	{AFCCMPS, C_COND, C_FREG, C_FREG, C_VCON, C_NONE, 57, 4, 0, 0, 0},

	/* scaled 12-bit unsigned displacement store */
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_UAUTO4K, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_UOREG4K, C_NONE, 20, 4, 0, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_UAUTO8K, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_UOREG8K, C_NONE, 20, 4, 0, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_UAUTO16K, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_UOREG16K, C_NONE, 20, 4, 0, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_UAUTO32K, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_UOREG32K, C_NONE, 20, 4, 0, 0, 0},

	{AFMOVS, C_FREG, C_NONE, C_NONE, C_UAUTO16K, C_NONE, 20, 4, REGSP, 0, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_UOREG16K, C_NONE, 20, 4, 0, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_UAUTO32K, C_NONE, 20, 4, REGSP, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_UOREG32K, C_NONE, 20, 4, 0, 0, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_UAUTO64K, C_NONE, 20, 4, REGSP, 0, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_UOREG64K, C_NONE, 20, 4, 0, 0, 0},

	/* unscaled 9-bit signed displacement store */
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_NSAUTO, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_NSOREG, C_NONE, 20, 4, 0, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_NSAUTO, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_NSOREG, C_NONE, 20, 4, 0, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_NSAUTO, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_NSOREG, C_NONE, 20, 4, 0, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_NSAUTO, C_NONE, 20, 4, REGSP, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_NSOREG, C_NONE, 20, 4, 0, 0, 0},

	{AFMOVS, C_FREG, C_NONE, C_NONE, C_NSAUTO, C_NONE, 20, 4, REGSP, 0, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_NSOREG, C_NONE, 20, 4, 0, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_NSAUTO, C_NONE, 20, 4, REGSP, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_NSOREG, C_NONE, 20, 4, 0, 0, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_NSAUTO, C_NONE, 20, 4, REGSP, 0, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_NSOREG, C_NONE, 20, 4, 0, 0, 0},

	/* scaled 12-bit unsigned displacement load */
	{AMOVB, C_UAUTO4K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVB, C_UOREG4K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},
	{AMOVH, C_UAUTO8K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVH, C_UOREG8K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},
	{AMOVW, C_UAUTO16K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVW, C_UOREG16K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},
	{AMOVD, C_UAUTO32K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVD, C_UOREG32K, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},

	{AFMOVS, C_UAUTO16K, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, REGSP, 0, 0},
	{AFMOVS, C_UOREG16K, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, 0, 0, 0},
	{AFMOVD, C_UAUTO32K, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, REGSP, 0, 0},
	{AFMOVD, C_UOREG32K, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, 0, 0, 0},
	{AFMOVQ, C_UAUTO64K, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, REGSP, 0, 0},
	{AFMOVQ, C_UOREG64K, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, 0, 0, 0},

	/* unscaled 9-bit signed displacement load */
	{AMOVB, C_NSAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVB, C_NSOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},
	{AMOVH, C_NSAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVH, C_NSOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},
	{AMOVW, C_NSAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVW, C_NSOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},
	{AMOVD, C_NSAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, REGSP, 0, 0},
	{AMOVD, C_NSOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 21, 4, 0, 0, 0},

	{AFMOVS, C_NSAUTO, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, REGSP, 0, 0},
	{AFMOVS, C_NSOREG, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, 0, 0, 0},
	{AFMOVD, C_NSAUTO, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, REGSP, 0, 0},
	{AFMOVD, C_NSOREG, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, 0, 0, 0},
	{AFMOVQ, C_NSAUTO, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, REGSP, 0, 0},
	{AFMOVQ, C_NSOREG, C_NONE, C_NONE, C_FREG, C_NONE, 21, 4, 0, 0, 0},

	/* long displacement store */
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_LAUTO, C_NONE, 30, 8, REGSP, 0, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 30, 8, REGSP, LTO, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 30, 8, 0, 0, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 30, 8, 0, LTO, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_LAUTO, C_NONE, 30, 8, REGSP, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 30, 8, REGSP, LTO, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 30, 8, 0, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 30, 8, 0, LTO, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_LAUTO, C_NONE, 30, 8, REGSP, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 30, 8, REGSP, LTO, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 30, 8, 0, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 30, 8, 0, LTO, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_LAUTO, C_NONE, 30, 8, REGSP, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 30, 8, REGSP, LTO, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 30, 8, 0, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 30, 8, 0, LTO, 0},

	{AFMOVS, C_FREG, C_NONE, C_NONE, C_LAUTO, C_NONE, 30, 8, REGSP, 0, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 30, 8, REGSP, LTO, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 30, 8, 0, 0, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 30, 8, 0, LTO, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_LAUTO, C_NONE, 30, 8, REGSP, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 30, 8, REGSP, LTO, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 30, 8, 0, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 30, 8, 0, LTO, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_LAUTO, C_NONE, 30, 8, REGSP, 0, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 30, 8, REGSP, LTO, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 30, 8, 0, 0, 0},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 30, 8, 0, LTO, 0},

	/* long displacement load */
	{AMOVB, C_LAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, 0, 0},
	{AMOVB, C_LAUTOPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, LFROM, 0},
	{AMOVB, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, 0, 0},
	{AMOVB, C_LOREGPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, LFROM, 0},
	{AMOVH, C_LAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, 0, 0},
	{AMOVH, C_LAUTOPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, LFROM, 0},
	{AMOVH, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, 0, 0},
	{AMOVH, C_LOREGPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, LFROM, 0},
	{AMOVW, C_LAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, 0, 0},
	{AMOVW, C_LAUTOPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, LFROM, 0},
	{AMOVW, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, 0, 0},
	{AMOVW, C_LOREGPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, LFROM, 0},
	{AMOVD, C_LAUTO, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, 0, 0},
	{AMOVD, C_LAUTOPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, REGSP, LFROM, 0},
	{AMOVD, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, 0, 0},
	{AMOVD, C_LOREGPOOL, C_NONE, C_NONE, C_REGZR, C_NONE, 31, 8, 0, LFROM, 0},

	{AFMOVS, C_LAUTO, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, REGSP, 0, 0},
	{AFMOVS, C_LAUTOPOOL, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, REGSP, LFROM, 0},
	{AFMOVS, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, 0, 0, 0},
	{AFMOVS, C_LOREGPOOL, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, 0, LFROM, 0},
	{AFMOVD, C_LAUTO, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, REGSP, 0, 0},
	{AFMOVD, C_LAUTOPOOL, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, REGSP, LFROM, 0},
	{AFMOVD, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, 0, 0, 0},
	{AFMOVD, C_LOREGPOOL, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, 0, LFROM, 0},
	{AFMOVQ, C_LAUTO, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, REGSP, 0, 0},
	{AFMOVQ, C_LAUTOPOOL, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, REGSP, LFROM, 0},
	{AFMOVQ, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, 0, 0, 0},
	{AFMOVQ, C_LOREGPOOL, C_NONE, C_NONE, C_FREG, C_NONE, 31, 8, 0, LFROM, 0},

	/* pre/post-indexed load (unscaled, signed 9-bit offset) */
	{AMOVD, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPOST},
	{AMOVW, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPOST},
	{AMOVH, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPOST},
	{AMOVB, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPOST},
	{AFMOVS, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 22, 4, 0, 0, C_XPOST},
	{AFMOVD, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 22, 4, 0, 0, C_XPOST},
	{AFMOVQ, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 22, 4, 0, 0, C_XPOST},

	{AMOVD, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPRE},
	{AMOVW, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPRE},
	{AMOVH, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPRE},
	{AMOVB, C_LOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 22, 4, 0, 0, C_XPRE},
	{AFMOVS, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 22, 4, 0, 0, C_XPRE},
	{AFMOVD, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 22, 4, 0, 0, C_XPRE},
	{AFMOVQ, C_LOREG, C_NONE, C_NONE, C_FREG, C_NONE, 22, 4, 0, 0, C_XPRE},

	/* pre/post-indexed store (unscaled, signed 9-bit offset) */
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPOST},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPOST},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPOST},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPOST},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPOST},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPOST},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPOST},

	{AMOVD, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPRE},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPRE},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPRE},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPRE},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPRE},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPRE},
	{AFMOVQ, C_FREG, C_NONE, C_NONE, C_LOREG, C_NONE, 23, 4, 0, 0, C_XPRE},

	/* load with shifted or extended register offset */
	{AMOVD, C_ROFF, C_NONE, C_NONE, C_REGZR, C_NONE, 98, 4, 0, 0, 0},
	{AMOVW, C_ROFF, C_NONE, C_NONE, C_REGZR, C_NONE, 98, 4, 0, 0, 0},
	{AMOVH, C_ROFF, C_NONE, C_NONE, C_REGZR, C_NONE, 98, 4, 0, 0, 0},
	{AMOVB, C_ROFF, C_NONE, C_NONE, C_REGZR, C_NONE, 98, 4, 0, 0, 0},
	{AFMOVS, C_ROFF, C_NONE, C_NONE, C_FREG, C_NONE, 98, 4, 0, 0, 0},
	{AFMOVD, C_ROFF, C_NONE, C_NONE, C_FREG, C_NONE, 98, 4, 0, 0, 0},

	/* store with extended register offset */
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_ROFF, C_NONE, 99, 4, 0, 0, 0},
	{AMOVW, C_REGZR, C_NONE, C_NONE, C_ROFF, C_NONE, 99, 4, 0, 0, 0},
	{AMOVH, C_REGZR, C_NONE, C_NONE, C_ROFF, C_NONE, 99, 4, 0, 0, 0},
	{AMOVB, C_REGZR, C_NONE, C_NONE, C_ROFF, C_NONE, 99, 4, 0, 0, 0},
	{AFMOVS, C_FREG, C_NONE, C_NONE, C_ROFF, C_NONE, 99, 4, 0, 0, 0},
	{AFMOVD, C_FREG, C_NONE, C_NONE, C_ROFF, C_NONE, 99, 4, 0, 0, 0},

	/* pre/post-indexed/signed-offset load/store register pair
	   (unscaled, signed 10-bit quad-aligned and long offset).
	The pre/post-indexed format only supports OREG cases because
	the RSP and pseudo registers are not allowed to be modified
	in this way. */
	{AFLDPQ, C_NQAUTO_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, REGSP, 0, 0},
	{AFLDPQ, C_PQAUTO_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, REGSP, 0, 0},
	{AFLDPQ, C_UAUTO4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, REGSP, 0, 0},
	{AFLDPQ, C_NAUTO4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, REGSP, 0, 0},
	{AFLDPQ, C_LAUTO, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, REGSP, 0, 0},
	{AFLDPQ, C_LAUTOPOOL, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, REGSP, LFROM, 0},
	{AFLDPQ, C_NQOREG_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, 0},
	{AFLDPQ, C_NQOREG_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPRE},
	{AFLDPQ, C_NQOREG_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPOST},
	{AFLDPQ, C_PQOREG_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, 0},
	{AFLDPQ, C_PQOREG_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPRE},
	{AFLDPQ, C_PQOREG_16, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPOST},
	{AFLDPQ, C_UOREG4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, 0, 0, 0},
	{AFLDPQ, C_NOREG4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, 0, 0, 0},
	{AFLDPQ, C_LOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, 0, 0, 0},
	{AFLDPQ, C_LOREGPOOL, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, 0, LFROM, 0},
	{AFLDPQ, C_ADDR, C_NONE, C_NONE, C_PAIR, C_NONE, 88, 12, 0, 0, 0},

	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_NQAUTO_16, C_NONE, 67, 4, REGSP, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_PQAUTO_16, C_NONE, 67, 4, REGSP, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_UAUTO4K, C_NONE, 76, 8, REGSP, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_NAUTO4K, C_NONE, 76, 8, REGSP, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_LAUTO, C_NONE, 77, 12, REGSP, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 77, 12, REGSP, LTO, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_NQOREG_16, C_NONE, 67, 4, 0, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_NQOREG_16, C_NONE, 67, 4, 0, 0, C_XPRE},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_NQOREG_16, C_NONE, 67, 4, 0, 0, C_XPOST},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_PQOREG_16, C_NONE, 67, 4, 0, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_PQOREG_16, C_NONE, 67, 4, 0, 0, C_XPRE},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_PQOREG_16, C_NONE, 67, 4, 0, 0, C_XPOST},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_UOREG4K, C_NONE, 76, 8, 0, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_NOREG4K, C_NONE, 76, 8, 0, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_LOREG, C_NONE, 77, 12, 0, 0, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 77, 12, 0, LTO, 0},
	{AFSTPQ, C_PAIR, C_NONE, C_NONE, C_ADDR, C_NONE, 87, 12, 0, 0, 0},

	{ALDP, C_NPAUTO, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, REGSP, 0, 0},
	{ALDP, C_PPAUTO, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, REGSP, 0, 0},
	{ALDP, C_UAUTO4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, REGSP, 0, 0},
	{ALDP, C_NAUTO4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, REGSP, 0, 0},
	{ALDP, C_LAUTO, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, REGSP, 0, 0},
	{ALDP, C_LAUTOPOOL, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, REGSP, LFROM, 0},
	{ALDP, C_NPOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, 0},
	{ALDP, C_NPOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPRE},
	{ALDP, C_NPOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPOST},
	{ALDP, C_PPOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, 0},
	{ALDP, C_PPOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPRE},
	{ALDP, C_PPOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPOST},
	{ALDP, C_UOREG4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, 0, 0, 0},
	{ALDP, C_NOREG4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, 0, 0, 0},
	{ALDP, C_LOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, 0, 0, 0},
	{ALDP, C_LOREGPOOL, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, 0, LFROM, 0},
	{ALDP, C_ADDR, C_NONE, C_NONE, C_PAIR, C_NONE, 88, 12, 0, 0, 0},

	{ASTP, C_PAIR, C_NONE, C_NONE, C_NPAUTO, C_NONE, 67, 4, REGSP, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_PPAUTO, C_NONE, 67, 4, REGSP, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_UAUTO4K, C_NONE, 76, 8, REGSP, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_NAUTO4K, C_NONE, 76, 8, REGSP, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_LAUTO, C_NONE, 77, 12, REGSP, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 77, 12, REGSP, LTO, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_NPOREG, C_NONE, 67, 4, 0, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_NPOREG, C_NONE, 67, 4, 0, 0, C_XPRE},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_NPOREG, C_NONE, 67, 4, 0, 0, C_XPOST},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_PPOREG, C_NONE, 67, 4, 0, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_PPOREG, C_NONE, 67, 4, 0, 0, C_XPRE},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_PPOREG, C_NONE, 67, 4, 0, 0, C_XPOST},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_UOREG4K, C_NONE, 76, 8, 0, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_NOREG4K, C_NONE, 76, 8, 0, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_LOREG, C_NONE, 77, 12, 0, 0, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 77, 12, 0, LTO, 0},
	{ASTP, C_PAIR, C_NONE, C_NONE, C_ADDR, C_NONE, 87, 12, 0, 0, 0},

	// differ from LDP/STP for C_NSAUTO_4/C_PSAUTO_4/C_NSOREG_4/C_PSOREG_4
	{ALDPW, C_NSAUTO_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, REGSP, 0, 0},
	{ALDPW, C_PSAUTO_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, REGSP, 0, 0},
	{ALDPW, C_UAUTO4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, REGSP, 0, 0},
	{ALDPW, C_NAUTO4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, REGSP, 0, 0},
	{ALDPW, C_LAUTO, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, REGSP, 0, 0},
	{ALDPW, C_LAUTOPOOL, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, REGSP, LFROM, 0},
	{ALDPW, C_NSOREG_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, 0},
	{ALDPW, C_NSOREG_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPRE},
	{ALDPW, C_NSOREG_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPOST},
	{ALDPW, C_PSOREG_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, 0},
	{ALDPW, C_PSOREG_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPRE},
	{ALDPW, C_PSOREG_4, C_NONE, C_NONE, C_PAIR, C_NONE, 66, 4, 0, 0, C_XPOST},
	{ALDPW, C_UOREG4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, 0, 0, 0},
	{ALDPW, C_NOREG4K, C_NONE, C_NONE, C_PAIR, C_NONE, 74, 8, 0, 0, 0},
	{ALDPW, C_LOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, 0, 0, 0},
	{ALDPW, C_LOREGPOOL, C_NONE, C_NONE, C_PAIR, C_NONE, 75, 12, 0, LFROM, 0},
	{ALDPW, C_ADDR, C_NONE, C_NONE, C_PAIR, C_NONE, 88, 12, 0, 0, 0},

	{ASTPW, C_PAIR, C_NONE, C_NONE, C_NSAUTO_4, C_NONE, 67, 4, REGSP, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_PSAUTO_4, C_NONE, 67, 4, REGSP, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_UAUTO4K, C_NONE, 76, 8, REGSP, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_NAUTO4K, C_NONE, 76, 8, REGSP, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_LAUTO, C_NONE, 77, 12, REGSP, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_LAUTOPOOL, C_NONE, 77, 12, REGSP, LTO, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_NSOREG_4, C_NONE, 67, 4, 0, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_NSOREG_4, C_NONE, 67, 4, 0, 0, C_XPRE},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_NSOREG_4, C_NONE, 67, 4, 0, 0, C_XPOST},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_PSOREG_4, C_NONE, 67, 4, 0, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_PSOREG_4, C_NONE, 67, 4, 0, 0, C_XPRE},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_PSOREG_4, C_NONE, 67, 4, 0, 0, C_XPOST},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_UOREG4K, C_NONE, 76, 8, 0, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_NOREG4K, C_NONE, 76, 8, 0, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_LOREG, C_NONE, 77, 12, 0, 0, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_LOREGPOOL, C_NONE, 77, 12, 0, LTO, 0},
	{ASTPW, C_PAIR, C_NONE, C_NONE, C_ADDR, C_NONE, 87, 12, 0, 0, 0},

	{ASWPD, C_REGZR, C_NONE, C_NONE, C_ZOREG, C_REGZR, 47, 4, 0, 0, 0},
	{ASWPD, C_REGZR, C_NONE, C_NONE, C_ZAUTO, C_REGZR, 47, 4, REGSP, 0, 0},
	{ACASPD, C_PAIR, C_NONE, C_NONE, C_ZOREG, C_PAIR, 106, 4, 0, 0, 0},
	{ACASPD, C_PAIR, C_NONE, C_NONE, C_ZAUTO, C_PAIR, 106, 4, REGSP, 0, 0},
	{ALDAPR, C_ZOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 58, 4, 0, 0, 0},
	{ALDAR, C_ZOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 58, 4, 0, 0, 0},
	{ALDXR, C_ZOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 58, 4, 0, 0, 0},
	{ALDAXR, C_ZOREG, C_NONE, C_NONE, C_REGZR, C_NONE, 58, 4, 0, 0, 0},
	{ALDXP, C_ZOREG, C_NONE, C_NONE, C_PAIR, C_NONE, 58, 4, 0, 0, 0},
	{ASTLR, C_REGZR, C_NONE, C_NONE, C_ZOREG, C_NONE, 59, 4, 0, 0, 0},
	{ASTXR, C_REGZR, C_NONE, C_NONE, C_ZOREG, C_REGZR, 59, 4, 0, 0, 0},
	{ASTLXR, C_REGZR, C_NONE, C_NONE, C_ZOREG, C_REGZR, 59, 4, 0, 0, 0},
	{ASTXP, C_PAIR, C_NONE, C_NONE, C_ZOREG, C_REGZR, 59, 4, 0, 0, 0},

	/* VLD[1-4]/VST[1-4] */
	{AVLD1, C_ZOREG, C_NONE, C_NONE, C_LIST, C_NONE, 81, 4, 0, 0, 0},
	{AVLD1, C_LOREG, C_NONE, C_NONE, C_LIST, C_NONE, 81, 4, 0, 0, C_XPOST},
	{AVLD1, C_ROFF, C_NONE, C_NONE, C_LIST, C_NONE, 81, 4, 0, 0, C_XPOST},
	{AVLD1R, C_ZOREG, C_NONE, C_NONE, C_LIST, C_NONE, 81, 4, 0, 0, 0},
	{AVLD1R, C_LOREG, C_NONE, C_NONE, C_LIST, C_NONE, 81, 4, 0, 0, C_XPOST},
	{AVLD1R, C_ROFF, C_NONE, C_NONE, C_LIST, C_NONE, 81, 4, 0, 0, C_XPOST},
	{AVLD1, C_LOREG, C_NONE, C_NONE, C_ELEM, C_NONE, 97, 4, 0, 0, C_XPOST},
	{AVLD1, C_ROFF, C_NONE, C_NONE, C_ELEM, C_NONE, 97, 4, 0, 0, C_XPOST},
	{AVLD1, C_LOREG, C_NONE, C_NONE, C_ELEM, C_NONE, 97, 4, 0, 0, 0},
	{AVST1, C_LIST, C_NONE, C_NONE, C_ZOREG, C_NONE, 84, 4, 0, 0, 0},
	{AVST1, C_LIST, C_NONE, C_NONE, C_LOREG, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST1, C_LIST, C_NONE, C_NONE, C_ROFF, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST2, C_LIST, C_NONE, C_NONE, C_ZOREG, C_NONE, 84, 4, 0, 0, 0},
	{AVST2, C_LIST, C_NONE, C_NONE, C_LOREG, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST2, C_LIST, C_NONE, C_NONE, C_ROFF, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST3, C_LIST, C_NONE, C_NONE, C_ZOREG, C_NONE, 84, 4, 0, 0, 0},
	{AVST3, C_LIST, C_NONE, C_NONE, C_LOREG, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST3, C_LIST, C_NONE, C_NONE, C_ROFF, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST4, C_LIST, C_NONE, C_NONE, C_ZOREG, C_NONE, 84, 4, 0, 0, 0},
	{AVST4, C_LIST, C_NONE, C_NONE, C_LOREG, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST4, C_LIST, C_NONE, C_NONE, C_ROFF, C_NONE, 84, 4, 0, 0, C_XPOST},
	{AVST1, C_ELEM, C_NONE, C_NONE, C_LOREG, C_NONE, 96, 4, 0, 0, C_XPOST},
	{AVST1, C_ELEM, C_NONE, C_NONE, C_ROFF, C_NONE, 96, 4, 0, 0, C_XPOST},
	{AVST1, C_ELEM, C_NONE, C_NONE, C_LOREG, C_NONE, 96, 4, 0, 0, 0},

	/* special */
	{AMOVD, C_SPR, C_NONE, C_NONE, C_REGZR, C_NONE, 35, 4, 0, 0, 0},
	{AMRS, C_SPR, C_NONE, C_NONE, C_REGZR, C_NONE, 35, 4, 0, 0, 0},
	{AMOVD, C_REGZR, C_NONE, C_NONE, C_SPR, C_NONE, 36, 4, 0, 0, 0},
	{AMSR, C_REGZR, C_NONE, C_NONE, C_SPR, C_NONE, 36, 4, 0, 0, 0},
	{AMOVD, C_VCON, C_NONE, C_NONE, C_SPR, C_NONE, 37, 4, 0, 0, 0},
	{AMSR, C_VCON, C_NONE, C_NONE, C_SPR, C_NONE, 37, 4, 0, 0, 0},
	{AMSR, C_VCON, C_NONE, C_NONE, C_SPOP, C_NONE, 37, 4, 0, 0, 0},
	{APRFM, C_UOREG32K, C_NONE, C_NONE, C_SPOP, C_NONE, 91, 4, 0, 0, 0},
	{APRFM, C_UOREG32K, C_NONE, C_NONE, C_LCON, C_NONE, 91, 4, 0, 0, 0},
	{ARPRFM, C_ZOREG, C_REG, C_NONE, C_SPOP, C_NONE, 109, 4, 0, 0, 0},
	{ARPRFM, C_ZOREG, C_REG, C_NONE, C_LCON, C_NONE, 109, 4, 0, 0, 0},
	{ADMB, C_VCON, C_NONE, C_NONE, C_NONE, C_NONE, 51, 4, 0, 0, 0},
	{AHINT, C_VCON, C_NONE, C_NONE, C_NONE, C_NONE, 52, 4, 0, 0, 0},
	{ASYS, C_VCON, C_NONE, C_NONE, C_NONE, C_NONE, 50, 4, 0, 0, 0},
	{ASYS, C_VCON, C_NONE, C_NONE, C_REGZR, C_NONE, 50, 4, 0, 0, 0},
	{ASYSL, C_VCON, C_NONE, C_NONE, C_REGZR, C_NONE, 50, 4, 0, 0, 0},
	{ATLBI, C_SPOP, C_NONE, C_NONE, C_NONE, C_NONE, 107, 4, 0, 0, 0},
	{ATLBI, C_SPOP, C_NONE, C_NONE, C_REGZR, C_NONE, 107, 4, 0, 0, 0},

	/* encryption instructions */
	{AAESD, C_VREG, C_NONE, C_NONE, C_VREG, C_NONE, 26, 4, 0, 0, 0}, // for compatibility with old code
	{AAESD, C_ARNG, C_NONE, C_NONE, C_ARNG, C_NONE, 26, 4, 0, 0, 0}, // recommend using the new one for better readability
	{ASHA1C, C_VREG, C_VREG, C_NONE, C_VREG, C_NONE, 49, 4, 0, 0, 0},
	{ASHA1C, C_ARNG, C_VREG, C_NONE, C_VREG, C_NONE, 49, 4, 0, 0, 0},
	{ASHA1SU0, C_ARNG, C_ARNG, C_NONE, C_ARNG, C_NONE, 63, 4, 0, 0, 0},
	{AVREV32, C_ARNG, C_NONE, C_NONE, C_ARNG, C_NONE, 83, 4, 0, 0, 0},
	{AVPMULL, C_ARNG, C_ARNG, C_NONE, C_ARNG, C_NONE, 93, 4, 0, 0, 0},
	{AVEOR3, C_ARNG, C_ARNG, C_ARNG, C_ARNG, C_NONE, 103, 4, 0, 0, 0},
	{AVXAR, C_VCON, C_ARNG, C_ARNG, C_ARNG, C_NONE, 104, 4, 0, 0, 0},

	/* sve instructions */
	{AZADD, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 1, 4, 0, SVE, 0},
	{AZADD, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 2, 4, 0, SVE, 0},
	{AZADD, C_VCON, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 3, 4, 0, SVE, 0},
	{AZSUBR, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 2, 4, 0, SVE, 0},
	{AZSUBR, C_VCON, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 3, 4, 0, SVE, 0},
	{AZAND, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 4, 4, 0, SVE, 0},
	{AZAND, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 2, 4, 0, SVE, 0},
	{AZAND, C_BITCON, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 5, 4, 0, SVE, 0},
	{APAND, C_PARNG, C_PARNG, C_PREG_Z, C_PARNG, C_NONE, 6, 4, 0, SVE, 0},
	{AZSEL, C_ZARNG, C_ZARNG, C_PREG, C_ZARNG, C_NONE, 14, 4, 0, SVE, 0},
	{APSEL, C_PARNG, C_PARNG, C_PREG, C_PARNG, C_NONE, 6, 4, 0, SVE, 0},
	{AZORN, C_BITCON, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 5, 4, 0, SVE, 0},
	{AZDUP, C_RSP, C_NONE, C_NONE, C_ZARNG, C_NONE, 7, 4, 0, SVE, 0},
	{AZDUP, C_ZELEM, C_NONE, C_NONE, C_ZARNG, C_NONE, 8, 4, 0, SVE, 0},
	{AZDUP, C_VCON, C_NONE, C_NONE, C_ZARNG, C_NONE, 3, 4, 0, SVE, 0},
	{AWHILELT, C_REGZR, C_REGZR, C_NONE, C_PARNG, C_NONE, 10, 4, 0, SVE, 0},
	{AWHILELT, C_REGZR, C_REGZR, C_NONE, C_LIST, C_NONE, 11, 4, 0, SVE, 0},
	{AWHILELTW, C_REGZR, C_REGZR, C_NONE, C_PARNG, C_NONE, 10, 4, 0, SVE, 0},
	{AZCMPEQ, C_ZARNG, C_ZARNG, C_PREG_Z, C_PARNG, C_NONE, 12, 4, 0, SVE, 0},
	{AZCMPEQ, C_VCON, C_ZARNG, C_PREG_Z, C_PARNG, C_NONE, 13, 4, 0, SVE, 0},
	{AZSUQADD, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 2, 4, 0, SVE, 0},
	{AZASR, C_VCON, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 15, 4, 0, SVE, 0},
	{AZASR, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 16, 4, 0, SVE, 0},
	{AZASR, C_VCON, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 18, 4, 0, SVE, 0},
	{AZASR, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 19, 4, 0, SVE, 0},
	{AZSQSHL, C_VCON, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 15, 4, 0, SVE, 0},
	{AZASRD, C_VCON, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 15, 4, 0, SVE, 0},
	{AZASRR, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 16, 4, 0, SVE, 0},
	{AZSXTB, C_ZARNG, C_PREG_M, C_NONE, C_ZARNG, C_NONE, 17, 4, 0, SVE, 0},
	{APTEST, C_PARNG, C_PREG, C_NONE, C_NONE, C_NONE, 20, 4, 0, SVE, 0},
	{APFIRST, C_PARNG, C_PREG, C_NONE, C_PARNG, C_NONE, 21, 4, 0, SVE, 0},
	{APFALSE, C_NONE, C_NONE, C_NONE, C_PARNG, C_NONE, 22, 4, 0, SVE, 0},
	{APRDFFR, C_PREG_Z, C_NONE, C_NONE, C_PARNG, C_NONE, 23, 4, 0, SVE, 0},
	{APRDFFR, C_NONE, C_NONE, C_NONE, C_PARNG, C_NONE, 23, 4, 0, SVE, 0},
	{APRDFFRS, C_PREG_Z, C_NONE, C_NONE, C_PARNG, C_NONE, 23, 4, 0, SVE, 0},
	{APNEXT, C_PARNG, C_PREG, C_NONE, C_PARNG, C_NONE, 24, 4, 0, SVE, 0},
	{APTRUE, C_VCON, C_NONE, C_NONE, C_PARNG, C_NONE, 25, 4, 0, SVE, 0},
	{APTRUES, C_VCON, C_NONE, C_NONE, C_PARNG, C_NONE, 25, 4, 0, SVE, 0},
	{APTRUE, C_NONE, C_NONE, C_NONE, C_PARNG, C_NONE, 25, 4, 0, SVE, 0},
	{APTRUES, C_NONE, C_NONE, C_NONE, C_PARNG, C_NONE, 25, 4, 0, SVE, 0},
	{AZINDEX, C_VCON, C_NONE, C_VCON, C_ZARNG, C_NONE, 26, 4, 0, SVE, 0},
	{AZINDEX, C_VCON, C_NONE, C_REGZR, C_ZARNG, C_NONE, 26, 4, 0, SVE, 0},
	{AZINDEX, C_REGZR, C_NONE, C_VCON, C_ZARNG, C_NONE, 26, 4, 0, SVE, 0},
	{AZINDEX, C_REGZR, C_NONE, C_REGZR, C_ZARNG, C_NONE, 26, 4, 0, SVE, 0},
	{AZMUL, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 1, 4, 0, SVE, 0},
	{AZMUL, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 2, 4, 0, SVE, 0},
	{AZMUL, C_VCON, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 3, 4, 0, SVE, 0},
	{AZMUL, C_ZELEM, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 9, 4, 0, SVE, 0},
	{AZPMUL, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 1, 4, 0, SVE, 0},
	{AZSQDMULH, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 1, 4, 0, SVE, 0},
	{AZSMULH, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 1, 4, 0, SVE, 0},
	{AZSMULH, C_ZARNG, C_ZARNG, C_PREG_M, C_ZARNG, C_NONE, 2, 4, 0, SVE, 0},
	{APUNPKHI, C_PARNG, C_NONE, C_NONE, C_PARNG, C_NONE, 27, 4, 0, SVE, 0},
	{APZIP1, C_PARNG, C_PARNG, C_NONE, C_PARNG, C_NONE, 28, 4, 0, SVE, 0},
	{AZZIP1, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 29, 4, 0, SVE, 0},
	{APBRKN, C_PARNG, C_PARNG, C_PREG_Z, C_PARNG, C_NONE, 30, 4, 0, SVE, 0},
	{APBRKA, C_PARNG, C_PREG_Z, C_NONE, C_PARNG, C_NONE, 31, 4, 0, SVE, 0},
	{APBRKA, C_PARNG, C_PREG_M, C_NONE, C_PARNG, C_NONE, 31, 4, 0, SVE, 0},
	{APBRKAS, C_PARNG, C_PREG_Z, C_NONE, C_PARNG, C_NONE, 31, 4, 0, SVE, 0},
	{AZLD1B, C_LOREG, C_PREG_Z, C_NONE, C_LIST, C_NONE, 32, 4, 0, SVE, 0},
	{AZLD1B, C_ROFF, C_PREG_Z, C_NONE, C_LIST, C_NONE, 33, 4, 0, SVE, 0},
	{AZLD1SB, C_LOREG, C_PREG_Z, C_NONE, C_LIST, C_NONE, 32, 4, 0, SVE, 0},
	{AZLD1SB, C_ROFF, C_PREG_Z, C_NONE, C_LIST, C_NONE, 33, 4, 0, SVE, 0},
	{AZLDNF1B, C_LOREG, C_PREG_Z, C_NONE, C_LIST, C_NONE, 32, 4, 0, SVE, 0},
	{AZLDFF1B, C_ROFF, C_PREG_Z, C_NONE, C_LIST, C_NONE, 33, 4, 0, SVE, 0},
	{AZLDFF1B, C_LOREG, C_PREG_Z, C_NONE, C_LIST, C_NONE, 33, 4, 0, SVE, 0},
	{AZLD2B, C_LOREG, C_PREG_Z, C_NONE, C_LIST, C_NONE, 34, 4, 0, SVE, 0},
	{AZLD2B, C_ROFF, C_PREG_Z, C_NONE, C_LIST, C_NONE, 35, 4, 0, SVE, 0},
	{AZAESE, C_ZARNG, C_ZARNG, C_NONE, C_ZARNG, C_NONE, 36, 4, 0, SVE, 0},
	{AZAESMC, C_ZARNG, C_NONE, C_NONE, C_ZARNG, C_NONE, 37, 4, 0, SVE, 0},
	{AZST1B, C_LIST, C_PREG, C_NONE, C_LOREG, C_NONE, 38, 4, 0, SVE, 0},
	{AZST1B, C_LIST, C_PREG, C_NONE, C_ROFF, C_NONE, 39, 4, 0, SVE, 0},
	{AZST2B, C_LIST, C_PREG, C_NONE, C_LOREG, C_NONE, 40, 4, 0, SVE, 0},
	{AZST2B, C_LIST, C_PREG, C_NONE, C_ROFF, C_NONE, 41, 4, 0, SVE, 0},
	{AZINCB, C_VCON, C_NONE, C_VCON, C_REGZR, C_NONE, 42, 4, 0, SVE, 0},
	{AZINCB, C_NONE, C_NONE, C_VCON, C_REGZR, C_NONE, 42, 4, 0, SVE, 0},
	{AZINCB, C_NONE, C_NONE, C_NONE, C_REGZR, C_NONE, 42, 4, 0, SVE, 0},
	{AZINCH, C_VCON, C_NONE, C_VCON, C_REGZR, C_NONE, 42, 4, 0, SVE, 0},
	{AZINCH, C_NONE, C_NONE, C_VCON, C_REGZR, C_NONE, 42, 4, 0, SVE, 0},
	{AZINCH, C_NONE, C_NONE, C_NONE, C_REGZR, C_NONE, 42, 4, 0, SVE, 0},
	{AZINCH, C_VCON, C_NONE, C_VCON, C_ZARNG, C_NONE, 43, 4, 0, SVE, 0},
	{AZINCH, C_NONE, C_NONE, C_VCON, C_ZARNG, C_NONE, 43, 4, 0, SVE, 0},
	{AZINCH, C_NONE, C_NONE, C_NONE, C_ZARNG, C_NONE, 43, 4, 0, SVE, 0},
	{AZLASTA, C_ZARNG, C_PREG, C_NONE, C_REGZR, C_NONE, 44, 4, 0, SVE, 0},
	{AZLASTA, C_ZARNG, C_PREG, C_NONE, C_ARNG, C_NONE, 45, 4, 0, SVE, 0},
	{AZCNTP, C_PARNG, C_PREG, C_NONE, C_REGZR, C_NONE, 46, 4, 0, SVE, 0},
	{AZSADDV, C_ZARNG, C_PREG, C_NONE, C_VREG, C_NONE, 47, 4, 0, SVE, 0},

	{obj.AUNDEF, C_NONE, C_NONE, C_NONE, C_NONE, C_NONE, 90, 4, 0, 0, 0},
	{obj.APCDATA, C_VCON, C_NONE, C_NONE, C_VCON, C_NONE, 0, 0, 0, 0, 0},
	{obj.AFUNCDATA, C_VCON, C_NONE, C_NONE, C_ADDR, C_NONE, 0, 0, 0, 0, 0},
	{obj.ANOP, C_NONE, C_NONE, C_NONE, C_NONE, C_NONE, 0, 0, 0, 0, 0},
	{obj.ANOP, C_LCON, C_NONE, C_NONE, C_NONE, C_NONE, 0, 0, 0, 0, 0}, // nop variants, see #40689
	{obj.ANOP, C_REGZR, C_NONE, C_NONE, C_NONE, C_NONE, 0, 0, 0, 0, 0},
	{obj.ANOP, C_VREG, C_NONE, C_NONE, C_NONE, C_NONE, 0, 0, 0, 0, 0},
	{obj.ADUFFZERO, C_NONE, C_NONE, C_NONE, C_SBRA, C_NONE, 5, 4, 0, 0, 0},   // same as AB/ABL
	{obj.ADUFFCOPY, C_NONE, C_NONE, C_NONE, C_SBRA, C_NONE, 5, 4, 0, 0, 0},   // same as AB/ABL
	{obj.APCALIGN, C_LCON, C_NONE, C_NONE, C_NONE, C_NONE, 0, 0, 0, 0, 0},    // align code
	{obj.APCALIGNMAX, C_LCON, C_NONE, C_NONE, C_LCON, C_NONE, 0, 0, 0, 0, 0}, // align code, conditional
}

// Valid pstate field values, and value to use in instruction.
// Doesn't include special registers.
var pstatefield = []struct {
	opd SpecialOperand
	enc uint32
}{
	{SPOP_DAIFSet, 3<<16 | 4<<12 | 6<<5},
	{SPOP_DAIFClr, 3<<16 | 4<<12 | 7<<5},
}

var prfopfield = map[SpecialOperand]uint32{
	SPOP_PLDL1KEEP: 0,
	SPOP_PLDL1STRM: 1,
	SPOP_PLDL2KEEP: 2,
	SPOP_PLDL2STRM: 3,
	SPOP_PLDL3KEEP: 4,
	SPOP_PLDL3STRM: 5,
	SPOP_PLIL1KEEP: 8,
	SPOP_PLIL1STRM: 9,
	SPOP_PLIL2KEEP: 10,
	SPOP_PLIL2STRM: 11,
	SPOP_PLIL3KEEP: 12,
	SPOP_PLIL3STRM: 13,
	SPOP_PSTL1KEEP: 16,
	SPOP_PSTL1STRM: 17,
	SPOP_PSTL2KEEP: 18,
	SPOP_PSTL2STRM: 19,
	SPOP_PSTL3KEEP: 20,
	SPOP_PSTL3STRM: 21,
}

var rprfopfield = map[SpecialOperand]uint32{
	// The 6-bit 'operation' = [option2][option0][S][Rt2][Rt1][Rt0].
	// Four named values (ARM Arch Ref Manual):
	//   PLDKEEP (0b000000), PSTKEEP (0b000001),
	//   PLDSTRM (0b000100), PSTSTRM (0b000101).
	SPOP_PLDKEEP: 0,
	SPOP_PSTKEEP: 1,
	SPOP_PLDSTRM: 4,
	SPOP_PSTSTRM: 5,
}

// sysInstFields helps convert SYS alias instructions to SYS instructions.
// For example, the format of TLBI is: TLBI <tlbi_op>{, <Xt>}.
// It's equivalent to: SYS #<op1>, C8, <Cm>, #<op2>{, <Xt>}.
// The field hasOperand2 indicates whether Xt is required. It helps to check
// some combinations that may be undefined, such as TLBI VMALLE1IS, R0.
var sysInstFields = map[SpecialOperand]struct {
	op1         uint8
	cn          uint8
	cm          uint8
	op2         uint8
	hasOperand2 bool
}{
	// TLBI
	SPOP_VMALLE1IS:    {0, 8, 3, 0, false},
	SPOP_VAE1IS:       {0, 8, 3, 1, true},
	SPOP_ASIDE1IS:     {0, 8, 3, 2, true},
	SPOP_VAAE1IS:      {0, 8, 3, 3, true},
	SPOP_VALE1IS:      {0, 8, 3, 5, true},
	SPOP_VAALE1IS:     {0, 8, 3, 7, true},
	SPOP_VMALLE1:      {0, 8, 7, 0, false},
	SPOP_VAE1:         {0, 8, 7, 1, true},
	SPOP_ASIDE1:       {0, 8, 7, 2, true},
	SPOP_VAAE1:        {0, 8, 7, 3, true},
	SPOP_VALE1:        {0, 8, 7, 5, true},
	SPOP_VAALE1:       {0, 8, 7, 7, true},
	SPOP_IPAS2E1IS:    {4, 8, 0, 1, true},
	SPOP_IPAS2LE1IS:   {4, 8, 0, 5, true},
	SPOP_ALLE2IS:      {4, 8, 3, 0, false},
	SPOP_VAE2IS:       {4, 8, 3, 1, true},
	SPOP_ALLE1IS:      {4, 8, 3, 4, false},
	SPOP_VALE2IS:      {4, 8, 3, 5, true},
	SPOP_VMALLS12E1IS: {4, 8, 3, 6, false},
	SPOP_IPAS2E1:      {4, 8, 4, 1, true},
	SPOP_IPAS2LE1:     {4, 8, 4, 5, true},
	SPOP_ALLE2:        {4, 8, 7, 0, false},
	SPOP_VAE2:         {4, 8, 7, 1, true},
	SPOP_ALLE1:        {4, 8, 7, 4, false},
	SPOP_VALE2:        {4, 8, 7, 5, true},
	SPOP_VMALLS12E1:   {4, 8, 7, 6, false},
	SPOP_ALLE3IS:      {6, 8, 3, 0, false},
	SPOP_VAE3IS:       {6, 8, 3, 1, true},
	SPOP_VALE3IS:      {6, 8, 3, 5, true},
	SPOP_ALLE3:        {6, 8, 7, 0, false},
	SPOP_VAE3:         {6, 8, 7, 1, true},
	SPOP_VALE3:        {6, 8, 7, 5, true},
	SPOP_VMALLE1OS:    {0, 8, 1, 0, false},
	SPOP_VAE1OS:       {0, 8, 1, 1, true},
	SPOP_ASIDE1OS:     {0, 8, 1, 2, true},
	SPOP_VAAE1OS:      {0, 8, 1, 3, true},
	SPOP_VALE1OS:      {0, 8, 1, 5, true},
	SPOP_VAALE1OS:     {0, 8, 1, 7, true},
	SPOP_RVAE1IS:      {0, 8, 2, 1, true},
	SPOP_RVAAE1IS:     {0, 8, 2, 3, true},
	SPOP_RVALE1IS:     {0, 8, 2, 5, true},
	SPOP_RVAALE1IS:    {0, 8, 2, 7, true},
	SPOP_RVAE1OS:      {0, 8, 5, 1, true},
	SPOP_RVAAE1OS:     {0, 8, 5, 3, true},
	SPOP_RVALE1OS:     {0, 8, 5, 5, true},
	SPOP_RVAALE1OS:    {0, 8, 5, 7, true},
	SPOP_RVAE1:        {0, 8, 6, 1, true},
	SPOP_RVAAE1:       {0, 8, 6, 3, true},
	SPOP_RVALE1:       {0, 8, 6, 5, true},
	SPOP_RVAALE1:      {0, 8, 6, 7, true},
	SPOP_RIPAS2E1IS:   {4, 8, 0, 2, true},
	SPOP_RIPAS2LE1IS:  {4, 8, 0, 6, true},
	SPOP_ALLE2OS:      {4, 8, 1, 0, false},
	SPOP_VAE2OS:       {4, 8, 1, 1, true},
	SPOP_ALLE1OS:      {4, 8, 1, 4, false},
	SPOP_VALE2OS:      {4, 8, 1, 5, true},
	SPOP_VMALLS12E1OS: {4, 8, 1, 6, false},
	SPOP_RVAE2IS:      {4, 8, 2, 1, true},
	SPOP_RVALE2IS:     {4, 8, 2, 5, true},
	SPOP_IPAS2E1OS:    {4, 8, 4, 0, true},
	SPOP_RIPAS2E1:     {4, 8, 4, 2, true},
	SPOP_RIPAS2E1OS:   {4, 8, 4, 3, true},
	SPOP_IPAS2LE1OS:   {4, 8, 4, 4, true},
	SPOP_RIPAS2LE1:    {4, 8, 4, 6, true},
	SPOP_RIPAS2LE1OS:  {4, 8, 4, 7, true},
	SPOP_RVAE2OS:      {4, 8, 5, 1, true},
	SPOP_RVALE2OS:     {4, 8, 5, 5, true},
	SPOP_RVAE2:        {4, 8, 6, 1, true},
	SPOP_RVALE2:       {4, 8, 6, 5, true},
	SPOP_ALLE3OS:      {6, 8, 1, 0, false},
	SPOP_VAE3OS:       {6, 8, 1, 1, true},
	SPOP_VALE3OS:      {6, 8, 1, 5, true},
	SPOP_RVAE3IS:      {6, 8, 2, 1, true},
	SPOP_RVALE3IS:     {6, 8, 2, 5, true},
	SPOP_RVAE3OS:      {6, 8, 5, 1, true},
	SPOP_RVALE3OS:     {6, 8, 5, 5, true},
	SPOP_RVAE3:        {6, 8, 6, 1, true},
	SPOP_RVALE3:       {6, 8, 6, 5, true},
	// DC
	SPOP_IVAC:    {0, 7, 6, 1, true},
	SPOP_ISW:     {0, 7, 6, 2, true},
	SPOP_CSW:     {0, 7, 10, 2, true},
	SPOP_CISW:    {0, 7, 14, 2, true},
	SPOP_ZVA:     {3, 7, 4, 1, true},
	SPOP_CVAC:    {3, 7, 10, 1, true},
	SPOP_CVAU:    {3, 7, 11, 1, true},
	SPOP_CIVAC:   {3, 7, 14, 1, true},
	SPOP_IGVAC:   {0, 7, 6, 3, true},
	SPOP_IGSW:    {0, 7, 6, 4, true},
	SPOP_IGDVAC:  {0, 7, 6, 5, true},
	SPOP_IGDSW:   {0, 7, 6, 6, true},
	SPOP_CGSW:    {0, 7, 10, 4, true},
	SPOP_CGDSW:   {0, 7, 10, 6, true},
	SPOP_CIGSW:   {0, 7, 14, 4, true},
	SPOP_CIGDSW:  {0, 7, 14, 6, true},
	SPOP_GVA:     {3, 7, 4, 3, true},
	SPOP_GZVA:    {3, 7, 4, 4, true},
	SPOP_CGVAC:   {3, 7, 10, 3, true},
	SPOP_CGDVAC:  {3, 7, 10, 5, true},
	SPOP_CGVAP:   {3, 7, 12, 3, true},
	SPOP_CGDVAP:  {3, 7, 12, 5, true},
	SPOP_CGVADP:  {3, 7, 13, 3, true},
	SPOP_CGDVADP: {3, 7, 13, 5, true},
	SPOP_CIGVAC:  {3, 7, 14, 3, true},
	SPOP_CIGDVAC: {3, 7, 14, 5, true},
	SPOP_CVAP:    {3, 7, 12, 1, true},
	SPOP_CVADP:   {3, 7, 13, 1, true},
}

// Used for padding NOOP instruction
const OP_NOOP = 0xd503201f

// pcAlignPadLength returns the number of bytes required to align pc to alignedValue,
// reporting an error if alignedValue is not a power of two or is out of range.
func pcAlignPadLength(ctxt *obj.Link, pc int64, alignedValue int64) int {
	if !((alignedValue&(alignedValue-1) == 0) && 8 <= alignedValue && alignedValue <= 2048) {
		ctxt.Diag("alignment value of an instruction must be a power of two and in the range [8, 2048], got %d\n", alignedValue)
	}
	return int(-pc & (alignedValue - 1))
}

// size returns the size of the sequence of machine instructions when p is encoded with o.
// Usually it just returns o.size directly, in some cases it checks whether the optimization
// conditions are met, and if so returns the size of the optimized instruction sequence.
// These optimizations need to be synchronized with the asmout function.
func (o *Optab) size(ctxt *obj.Link, p *obj.Prog) int {
	// Optimize adrp+add+ld/st to adrp+ld/st(offset).
	sz := movesize(p.As)
	if sz != -1 {
		// Relocations R_AARCH64_LDST{64,32,16,8}_ABS_LO12_NC can only generate 8-byte, 4-byte,
		// 2-byte and 1-byte aligned addresses, so the address of load/store must be aligned.
		// Also symbols with prefix of "go:string." are Go strings, which will go into
		// the symbol table, their addresses are not necessary aligned, rule this out.
		//
		// Note that the code generation routines for these addressing forms call o.size
		// to decide whether to use the unaligned/aligned forms, so o.size's result is always
		// in sync with the code generation decisions, because it *is* the code generation decision.
		align := int64(1 << sz)
		if o.a1 == C_ADDR && p.From.Offset%align == 0 && symAlign(p.From.Sym) >= align ||
			o.a4 == C_ADDR && p.To.Offset%align == 0 && symAlign(p.To.Sym) >= align {
			return 8
		}
	}
	return int(o.size_)
}

// symAlign returns the expected symbol alignment of the symbol s.
// This must match the linker's own default alignment decisions.
func symAlign(s *obj.LSym) int64 {
	name := s.Name
	switch {
	case strings.HasPrefix(name, "go:string."),
		strings.HasPrefix(name, "type:.namedata."),
		strings.HasPrefix(name, "type:.importpath."),
		strings.HasSuffix(name, ".opendefer"),
		strings.HasSuffix(name, ".arginfo0"),
		strings.HasSuffix(name, ".arginfo1"),
		strings.HasSuffix(name, ".argliveinfo"):
		// These are just bytes, or varints.
		return 1
	case strings.HasPrefix(name, "gclocals·"):
		// It has 32-bit fields.
		return 4
	default:
		switch {
		case s.Size%8 == 0:
			return 8
		case s.Size%4 == 0:
			return 4
		case s.Size%2 == 0:
			return 2
		}
	}
	return 1
}

func span7(ctxt *obj.Link, cursym *obj.LSym, newprog obj.ProgAlloc) {
	if ctxt.Retpoline {
		ctxt.Diag("-spectre=ret not supported on arm64")
		ctxt.Retpoline = false // don't keep printing
	}

	p := cursym.Func().Text
	if p == nil || p.Link == nil { // handle external functions and ELF section symbols
		return
	}

	if oprange[AAND&obj.AMask] == nil {
		ctxt.Diag("arm64 ops not initialized, call arm64.buildop first")
	}

	c := ctxt7{ctxt: ctxt, newprog: newprog, cursym: cursym, autosize: int32(p.To.Offset & 0xffffffff), extrasize: int32(p.To.Offset >> 32)}
	p.To.Offset &= 0xffffffff // extrasize is no longer needed

	// Process literal pool and allocate initial program counter for each Prog, before
	// generating branch veneers.
	pc := int64(0)
	p.Pc = pc
	for p = p.Link; p != nil; p = p.Link {
		p.Pc = pc
		c.addLiteralsToPool(p)
		pc += int64(c.asmsizeBytes(p))
	}

	/*
	 * if any procedure is large enough to
	 * generate a large SBRA branch, then
	 * generate extra passes putting branches
	 * around jmps to fix. this is rare.
	 */
	changed := true
	for changed {
		changed = false
		pc = 0
		for p = c.cursym.Func().Text.Link; p != nil; p = p.Link {
			p.Pc = pc
			changed = changed || c.fixUpLongBranch(p)
			pc += int64(c.asmsizeBytes(p))
		}
	}

	/*
	 * lay out the code, emitting code and data relocations.
	 */
	buf := codeBuffer{&c.cursym.P}

	for p := c.cursym.Func().Text.Link; p != nil; p = p.Link {
		c.pc = p.Pc
		switch p.As {
		case obj.APCALIGN, obj.APCALIGNMAX:
			v := obj.AlignmentPaddingLength(int32(p.Pc), p, c.ctxt)
			for i := 0; i < int(v/4); i++ {
				// emit ANOOP instruction by the padding size
				buf.emit(OP_NOOP)
			}
		case obj.ANOP, obj.AFUNCDATA, obj.APCDATA:
			continue
		default:
			var out [6]uint32
			o := c.oplook(p)
			var count int
			if o.flag&SVE == SVE {
				count = c.asmoutsve(p, out[:])
			} else {
				count = c.asmout(p, out[:])
			}
			buf.emit(out[:count]...)
		}
	}
	buf.finish()
	c.cursym.Size = int64(len(c.cursym.P))

	// Mark nonpreemptible instruction sequences.
	// We use REGTMP as a scratch register during call injection,
	// so instruction sequences that use REGTMP are unsafe to
	// preempt asynchronously.
	obj.MarkUnsafePoints(c.ctxt, c.cursym.Func().Text, c.newprog, c.isUnsafePoint, c.isRestartable)

	// Now that we know byte offsets, we can generate jump table entries.
	for _, jt := range cursym.Func().JumpTables {
		for i, p := range jt.Targets {
			// The ith jumptable entry points to the p.Pc'th
			// byte in the function symbol s.
			// TODO: try using relative PCs.
			jt.Sym.WriteAddr(ctxt, int64(i)*8, 8, cursym, p.Pc)
		}
	}
}

type codeBuffer struct {
	data *[]byte
}

func (cb *codeBuffer) pc() int64 {
	return int64(len(*cb.data))
}

// Write a sequence of opcodes into the code buffer.
func (cb *codeBuffer) emit(op ...uint32) {
	for _, o := range op {
		*cb.data = binary.LittleEndian.AppendUint32(*cb.data, o)
	}
}

// Completes the code buffer for the function by padding the buffer to function alignment
// with zero values.
func (cb *codeBuffer) finish() {
	for len(*cb.data)%funcAlign > 0 {
		*cb.data = append(*cb.data, 0)
	}
}

// Return the size of the assembled Prog, in bytes.
func (c *ctxt7) asmsizeBytes(p *obj.Prog) int {
	switch p.As {
	case obj.APCALIGN, obj.APCALIGNMAX:
		return obj.AlignmentPadding(int32(p.Pc), p, c.ctxt, c.cursym)
	case obj.ANOP, obj.AFUNCDATA, obj.APCDATA:
		return 0
	default:
		o := c.oplook(p)
		return o.size(c.ctxt, p)
	}
}

// Modify the Prog list if the Prog is a branch with a large offset that cannot be
// encoded in the instruction. Return true if a modification was made, false if not.
func (c *ctxt7) fixUpLongBranch(p *obj.Prog) bool {
	var toofar bool

	o := c.oplook(p)

	/* very large branches */
	if (o.flag&BRANCH14BITS != 0 || o.flag&BRANCH19BITS != 0) && p.To.Target() != nil {
		otxt := p.To.Target().Pc - p.Pc
		if o.flag&BRANCH14BITS != 0 { // branch instruction encodes 14 bits
			toofar = otxt <= -(1<<15)+10 || otxt >= (1<<15)-10
		} else if o.flag&BRANCH19BITS != 0 { // branch instruction encodes 19 bits
			toofar = otxt <= -(1<<20)+10 || otxt >= (1<<20)-10
		}
		if toofar {
			q := c.newprog()
			q.Link = p.Link
			p.Link = q
			q.As = AB
			q.To.Type = obj.TYPE_BRANCH
			q.To.SetTarget(p.To.Target())
			p.To.SetTarget(q)
			q = c.newprog()
			q.Link = p.Link
			p.Link = q
			q.As = AB
			q.To.Type = obj.TYPE_BRANCH
			q.To.SetTarget(q.Link.Link)
		}
	}

	return toofar
}

// Adds literal values from the Prog into the literal pool if necessary.
func (c *ctxt7) addLiteralsToPool(p *obj.Prog) {
	o := c.oplook(p)

	if o.flag&LFROM != 0 {
		c.addpool(p, &p.From)
	}
	if o.flag&LTO != 0 {
		c.addpool(p, &p.To)
	}
	if c.blitrl != nil {
		c.checkpool(p)
	}
}

// isUnsafePoint returns whether p is an unsafe point.
func (c *ctxt7) isUnsafePoint(p *obj.Prog) bool {
	// If p explicitly uses REGTMP, it's unsafe to preempt, because the
	// preemption sequence clobbers REGTMP.
	return p.From.Reg == REGTMP || p.To.Reg == REGTMP || p.Reg == REGTMP ||
		p.From.Type == obj.TYPE_REGREG && p.From.Offset == REGTMP ||
		p.To.Type == obj.TYPE_REGREG && p.To.Offset == REGTMP
}

// isRestartable returns whether p is a multi-instruction sequence that,
// if preempted, can be restarted.
func (c *ctxt7) isRestartable(p *obj.Prog) bool {
	if c.isUnsafePoint(p) {
		return false
	}
	// If p is a multi-instruction sequence with uses REGTMP inserted by
	// the assembler in order to materialize a large constant/offset, we
	// can restart p (at the start of the instruction sequence), recompute
	// the content of REGTMP, upon async preemption. Currently, all cases
	// of assembler-inserted REGTMP fall into this category.
	// If p doesn't use REGTMP, it can be simply preempted, so we don't
	// mark it.
	o := c.oplook(p)
	return o.size(c.ctxt, p) > 4 && o.flag&NOTUSETMP == 0
}

/*
 * when the first reference to the literal pool threatens
 * to go out of range of a 1Mb PC-relative offset
 * drop the pool now.
 */
func (c *ctxt7) checkpool(p *obj.Prog) {
	// If the pool is going to go out of range or p is the last instruction of the function,
	// flush the pool.
	if c.pool.size >= 0xffff0 || !ispcdisp(int32(p.Pc+4+int64(c.pool.size)-int64(c.pool.start)+8)) || p.Link == nil {
		c.flushpool(p)
	}
}

func (c *ctxt7) flushpool(p *obj.Prog) {
	// Needs to insert a branch before flushing the pool.
	// We don't need the jump if following an unconditional branch.
	// TODO: other unconditional operations.
	if !(p.As == AB || p.As == obj.ARET || p.As == AERET) {
		if c.ctxt.Debugvlog {
			fmt.Printf("note: flush literal pool at %#x: len=%d ref=%x\n", uint64(p.Pc+4), c.pool.size, c.pool.start)
		}
		q := c.newprog()
		if p.Link == nil {
			// If p is the last instruction of the function, insert an UNDEF instruction in case the
			// execution fall through to the pool.
			q.As = obj.AUNDEF
		} else {
			// Else insert a branch to the next instruction of p.
			q.As = AB
			q.To.Type = obj.TYPE_BRANCH
			q.To.SetTarget(p.Link)
		}
		q.Link = c.blitrl
		q.Pos = p.Pos
		c.blitrl = q
	}

	// The line number for constant pool entries doesn't really matter.
	// We set it to the line number of the preceding instruction so that
	// there are no deltas to encode in the pc-line tables.
	for q := c.blitrl; q != nil; q = q.Link {
		q.Pos = p.Pos
	}

	c.elitrl.Link = p.Link
	p.Link = c.blitrl

	c.blitrl = nil /* BUG: should refer back to values until out-of-range */
	c.elitrl = nil
	c.pool.size = 0
	c.pool.start = 0
}

/*
 * MOVD foo(SB), R is actually
 *   MOVD addr, REGTMP
 *   MOVD REGTMP, R
 * where addr is the address of the DWORD containing the address of foo.
 *
 * TODO: hash
 */
func (c *ctxt7) addpool(p *obj.Prog, a *obj.Addr) {
	cls := c.aclass(a)
	lit := c.instoffset
	t := c.newprog()
	t.As = AWORD
	sz := 4

	if a.Type == obj.TYPE_CONST {
		if lit != int64(int32(lit)) && uint64(lit) != uint64(uint32(lit)) {
			// out of range -0x80000000 ~ 0xffffffff, must store 64-bit.
			t.As = ADWORD
			sz = 8
		} // else store 32-bit
	} else if p.As == AMOVD && a.Type != obj.TYPE_MEM || cls == C_ADDR || cls == C_VCON || lit != int64(int32(lit)) || uint64(lit) != uint64(uint32(lit)) {
		// conservative: don't know if we want signed or unsigned extension.
		// in case of ambiguity, store 64-bit
		t.As = ADWORD
		sz = 8
	}

	t.To.Type = obj.TYPE_CONST
	t.To.Offset = lit

	for q := c.blitrl; q != nil; q = q.Link { /* could hash on t.t0.offset */
		if q.To == t.To {
			p.Pool = q
			return
		}
	}

	if c.blitrl == nil {
		c.blitrl = t
		c.pool.start = uint32(p.Pc)
	} else {
		c.elitrl.Link = t
	}
	c.elitrl = t
	if t.As == ADWORD {
		// make DWORD 8-byte aligned, this is not required by ISA,
		// just to avoid performance penalties when loading from
		// the constant pool across a cache line.
		c.pool.size = roundUp(c.pool.size, 8)
	}
	c.pool.size += uint32(sz)
	p.Pool = t
}

// roundUp rounds up x to "to".
func roundUp(x, to uint32) uint32 {
	if to == 0 || to&(to-1) != 0 {
		log.Fatalf("rounded up to a value that is not a power of 2: %d\n", to)
	}
	return (x + to - 1) &^ (to - 1)
}

// splitImm24uScaled splits an immediate into a scaled 12 bit unsigned lo value
// and an unscaled shifted 12 bit unsigned hi value. These are typically used
// by adding or subtracting the hi value and using the lo value as the offset
// for a load or store.
func splitImm24uScaled(v int32, shift int) (int32, int32, error) {
	if v < 0 {
		return 0, 0, fmt.Errorf("%d is not a 24 bit unsigned immediate", v)
	}
	if v > 0xfff000+0xfff<<shift {
		return 0, 0, fmt.Errorf("%d is too large for a scaled 24 bit unsigned immediate", v)
	}
	if v&((1<<shift)-1) != 0 {
		return 0, 0, fmt.Errorf("%d is not a multiple of %d", v, 1<<shift)
	}
	lo := (v >> shift) & 0xfff
	hi := v - (lo << shift)
	if hi > 0xfff000 {
		hi = 0xfff000
		lo = (v - hi) >> shift
	}
	if hi & ^0xfff000 != 0 {
		panic(fmt.Sprintf("bad split for %x with shift %v (%x, %x)", v, shift, hi, lo))
	}
	return hi, lo, nil
}

func (c *ctxt7) regoff(a *obj.Addr) int32 {
	c.instoffset = 0
	c.aclass(a)
	return int32(c.instoffset)
}

func isSTLXRop(op obj.As) bool {
	switch op {
	case ASTLXR, ASTLXRW, ASTLXRB, ASTLXRH,
		ASTXR, ASTXRW, ASTXRB, ASTXRH:
		return true
	}
	return false
}

func isSTXPop(op obj.As) bool {
	switch op {
	case ASTXP, ASTLXP, ASTXPW, ASTLXPW:
		return true
	}
	return false
}

func isANDop(op obj.As) bool {
	switch op {
	case AAND, AORR, AEOR, AANDS, ATST,
		ABIC, AEON, AORN, ABICS:
		return true
	}
	return false
}

func isANDWop(op obj.As) bool {
	switch op {
	case AANDW, AORRW, AEORW, AANDSW, ATSTW,
		ABICW, AEONW, AORNW, ABICSW:
		return true
	}
	return false
}

func isADDop(op obj.As) bool {
	switch op {
	case AADD, AADDS, ASUB, ASUBS, ACMN, ACMP:
		return true
	}
	return false
}

func isADDWop(op obj.As) bool {
	switch op {
	case AADDW, AADDSW, ASUBW, ASUBSW, ACMNW, ACMPW:
		return true
	}
	return false
}

func isADDSop(op obj.As) bool {
	switch op {
	case AADDS, AADDSW, ASUBS, ASUBSW:
		return true
	}
	return false
}

func isNEGop(op obj.As) bool {
	switch op {
	case ANEG, ANEGW, ANEGS, ANEGSW:
		return true
	}
	return false
}

func isLoadStorePairOp(op obj.As) bool {
	switch op {
	case AFLDPQ, AFSTPQ, ALDP, ASTP, ALDPW, ASTPW:
		return true
	}
	return false
}

func isMOVop(op obj.As) bool {
	switch op {
	case AMOVB, AMOVBU, AMOVH, AMOVHU, AMOVW, AMOVWU, AMOVD, AFMOVS, AFMOVD, AFMOVQ:
		return true
	}
	return false
}

func isRegShiftOrExt(a *obj.Addr) bool {
	return (a.Index-obj.RBaseARM64)&REG_EXT != 0 || (a.Index-obj.RBaseARM64)&REG_LSL != 0 ||
		(REG_SVE_VECTOR_EXT <= a.Index && a.Index < REG_SVE_VECTOR_EXT_END)
}

// Maximum PC-relative displacement.
// The actual limit is ±2²⁰, but we are conservative
// to avoid needing to recompute the literal pool flush points
// as span-dependent jumps are enlarged.
const maxPCDisp = 512 * 1024

// ispcdisp reports whether v is a valid PC-relative displacement.
func ispcdisp(v int32) bool {
	return -maxPCDisp < v && v < maxPCDisp && v&3 == 0
}

func isaddcon(v int64) bool {
	/* uimm12 or uimm24? */
	if v < 0 {
		return false
	}
	if (v & 0xFFF) == 0 {
		v >>= 12
	}
	return v <= 0xFFF
}

func isaddcon2(v int64) bool {
	return 0 <= v && v <= 0xFFFFFF
}

// isbitcon reports whether a constant can be encoded into a logical instruction.
// bitcon has a binary form of repetition of a bit sequence of length 2, 4, 8, 16, 32, or 64,
// which itself is a rotate (w.r.t. the length of the unit) of a sequence of ones.
// special cases: 0 and -1 are not bitcon.
// this function needs to run against virtually all the constants, so it needs to be fast.
// for this reason, bitcon testing and bitcon encoding are separate functions.
func isbitcon(x uint64) bool {
	if x == 1<<64-1 || x == 0 {
		return false
	}
	// determine the period and sign-extend a unit to 64 bits
	switch {
	case x != x>>32|x<<32:
		// period is 64
		// nothing to do
	case x != x>>16|x<<48:
		// period is 32
		x = uint64(int64(int32(x)))
	case x != x>>8|x<<56:
		// period is 16
		x = uint64(int64(int16(x)))
	case x != x>>4|x<<60:
		// period is 8
		x = uint64(int64(int8(x)))
	default:
		// period is 4 or 2, always true
		// 0001, 0010, 0100, 1000 -- 0001 rotate
		// 0011, 0110, 1100, 1001 -- 0011 rotate
		// 0111, 1011, 1101, 1110 -- 0111 rotate
		// 0101, 1010             -- 01   rotate, repeat
		return true
	}
	return sequenceOfOnes(x) || sequenceOfOnes(^x)
}

// sequenceOfOnes tests whether a constant is a sequence of ones in binary, with leading and trailing zeros.
func sequenceOfOnes(x uint64) bool {
	y := x & -x // lowest set bit of x. x is good iff x+y is a power of 2
	y += x
	return (y-1)&y == 0
}

// bitconEncode returns the encoding of a bitcon used in logical instructions
// x is known to be a bitcon
// a bitcon is a sequence of n ones at low bits (i.e. 1<<n-1), right rotated
// by R bits, and repeated with period of 64, 32, 16, 8, 4, or 2.
// it is encoded in logical instructions with 3 bitfields
// N (1 bit) : R (6 bits) : S (6 bits), where
// N=1           -- period=64
// N=0, S=0xxxxx -- period=32
// N=0, S=10xxxx -- period=16
// N=0, S=110xxx -- period=8
// N=0, S=1110xx -- period=4
// N=0, S=11110x -- period=2
// R is the shift amount, low bits of S = n-1
func bitconEncode(x uint64, mode int) uint32 {
	if mode == 32 {
		x &= 0xffffffff
		x = x<<32 | x
	}
	var period uint32
	// determine the period and sign-extend a unit to 64 bits
	switch {
	case x != x>>32|x<<32:
		period = 64
	case x != x>>16|x<<48:
		period = 32
		x = uint64(int64(int32(x)))
	case x != x>>8|x<<56:
		period = 16
		x = uint64(int64(int16(x)))
	case x != x>>4|x<<60:
		period = 8
		x = uint64(int64(int8(x)))
	case x != x>>2|x<<62:
		period = 4
		x = uint64(int64(x<<60) >> 60)
	default:
		period = 2
		x = uint64(int64(x<<62) >> 62)
	}
	neg := false
	if int64(x) < 0 {
		x = ^x
		neg = true
	}
	y := x & -x // lowest set bit of x.
	s := log2(y)
	n := log2(x+y) - s // x (or ^x) is a sequence of n ones left shifted by s bits
	if neg {
		// ^x is a sequence of n ones left shifted by s bits
		// adjust n, s for x
		s = n + s
		n = period - n
	}

	N := uint32(0)
	if mode == 64 && period == 64 {
		N = 1
	}
	R := (period - s) & (period - 1) & uint32(mode-1) // shift amount of right rotate
	S := (n - 1) | 63&^(period<<1-1)                  // low bits = #ones - 1, high bits encodes period
	return N<<22 | R<<16 | S<<10
}

func log2(x uint64) uint32 {
	if x == 0 {
		panic("log2 of 0")
	}
	n := uint32(0)
	if x >= 1<<32 {
		x >>= 32
		n += 32
	}
	if x >= 1<<16 {
		x >>= 16
		n += 16
	}
	if x >= 1<<8 {
		x >>= 8
		n += 8
	}
	if x >= 1<<4 {
		x >>= 4
		n += 4
	}
	if x >= 1<<2 {
		x >>= 2
		n += 2
	}
	if x >= 1<<1 {
		x >>= 1
		n += 1
	}
	return n
}

func autoclass(l int64) int {
	if l == 0 {
		return C_ZAUTO
	}

	if l < 0 {
		if l >= -256 && (l&15) == 0 {
			return C_NSAUTO_16
		}
		if l >= -256 && (l&7) == 0 {
			return C_NSAUTO_8
		}
		if l >= -256 && (l&3) == 0 {
			return C_NSAUTO_4
		}
		if l >= -256 {
			return C_NSAUTO
		}
		if l >= -512 && (l&15) == 0 {
			return C_NPAUTO_16
		}
		if l >= -512 && (l&7) == 0 {
			return C_NPAUTO
		}
		if l >= -1024 && (l&15) == 0 {
			return C_NQAUTO_16
		}
		if l >= -4095 {
			return C_NAUTO4K
		}
		return C_LAUTO
	}

	if l <= 255 {
		if (l & 15) == 0 {
			return C_PSAUTO_16
		}
		if (l & 7) == 0 {
			return C_PSAUTO_8
		}
		if (l & 3) == 0 {
			return C_PSAUTO_4
		}
		return C_PSAUTO
	}
	if l <= 504 {
		if l&15 == 0 {
			return C_PPAUTO_16
		}
		if l&7 == 0 {
			return C_PPAUTO
		}
	}
	if l <= 1008 {
		if l&15 == 0 {
			return C_PQAUTO_16
		}
	}
	if l <= 4095 {
		if l&15 == 0 {
			return C_UAUTO4K_16
		}
		if l&7 == 0 {
			return C_UAUTO4K_8
		}
		if l&3 == 0 {
			return C_UAUTO4K_4
		}
		if l&1 == 0 {
			return C_UAUTO4K_2
		}
		return C_UAUTO4K
	}
	if l <= 8190 {
		if l&15 == 0 {
			return C_UAUTO8K_16
		}
		if l&7 == 0 {
			return C_UAUTO8K_8
		}
		if l&3 == 0 {
			return C_UAUTO8K_4
		}
		if l&1 == 0 {
			return C_UAUTO8K
		}
	}
	if l <= 16380 {
		if l&15 == 0 {
			return C_UAUTO16K_16
		}
		if l&7 == 0 {
			return C_UAUTO16K_8
		}
		if l&3 == 0 {
			return C_UAUTO16K
		}
	}
	if l <= 32760 {
		if l&15 == 0 {
			return C_UAUTO32K_16
		}
		if l&7 == 0 {
			return C_UAUTO32K
		}
	}
	if l <= 65520 && (l&15) == 0 {
		return C_UAUTO64K
	}
	return C_LAUTO
}

func oregclass(l int64) int {
	return autoclass(l) - C_ZAUTO + C_ZOREG
}

/*
 * given an offset v and a class c (see above)
 * return the offset value to use in the instruction,
 * scaled if necessary
 */
func (c *ctxt7) offsetshift(p *obj.Prog, v int64, cls int) int64 {
	s := 0
	if cls >= C_SEXT1 && cls <= C_SEXT16 {
		s = cls - C_SEXT1
	} else {
		switch cls {
		case C_UAUTO4K, C_UOREG4K, C_ZOREG:
			s = 0
		case C_UAUTO8K, C_UOREG8K:
			s = 1
		case C_UAUTO16K, C_UOREG16K:
			s = 2
		case C_UAUTO32K, C_UOREG32K:
			s = 3
		case C_UAUTO64K, C_UOREG64K:
			s = 4
		default:
			c.ctxt.Diag("bad class: %v\n%v", DRconv(cls), p)
		}
	}
	vs := v >> uint(s)
	if vs<<uint(s) != v {
		c.ctxt.Diag("odd offset: %d\n%v", v, p)
	}
	return vs
}

/*
 * if v contains a single 16-bit value aligned
 * on a 16-bit field, and thus suitable for movk/movn,
 * return the field index 0 to 3; otherwise return -1.
 */
func movcon(v int64) int {
	for s := 0; s < 64; s += 16 {
		if (uint64(v) &^ (uint64(0xFFFF) << uint(s))) == 0 {
			return s / 16
		}
	}
	return -1
}

func rclass(r int16) int {
	switch {
	case REG_R0 <= r && r <= REG_R30: // not 31
		return C_REG
	case r == REGZERO:
		return C_REGZR
	case REG_F0 <= r && r <= REG_F31:
		return C_FREG
	case REG_V0 <= r && r <= REG_V31:
		return C_VREG
	case REG_Z0 <= r && r <= REG_Z31:
		return C_ZREG
	case REG_P0 <= r && r <= REG_P15:
		return C_PREG
	case r == REGSP:
		return C_RSP
	case r >= REG_ARNG && r < REG_ELEM:
		return C_ARNG
	case r >= REG_ELEM && r < REG_ELEM_END:
		return C_ELEM
	case r >= REG_LSL && r < REG_ARNG,
		r >= REG_EXT && r < REG_SVE_VECTOR_EXT:
		return C_EXTREG
	case r >= REG_SVE_VECTOR_EXT && r < REG_SVE_VECTOR_EXT_END:
		return C_EXTZARNG
	case r >= REG_SVE_VECTOR && r < REG_SVE_VECTOR_INDEX:
		return C_ZARNG
	case r >= REG_SVE_VECTOR_INDEX && r < REG_SVE_VECTOR_END:
		return C_ZELEM
	case r >= REG_SVE_PREDICATE && r < REG_SVE_PREDICATE_Z:
		return C_PARNG
	case r >= REG_SVE_PREDICATE_Z && r < REG_SVE_PREDICATE_M:
		return C_PREG_Z
	case r >= REG_SVE_PREDICATE_M && r < REG_SPECIAL:
		return C_PREG_M
	case r >= REG_SPECIAL:
		return C_SPR
	}
	return C_GOK
}

// con32class reclassifies the constant of 32-bit instruction. Because the constant type is 32-bit,
// but saved in Offset which type is int64, con32class treats it as uint32 type and reclassifies it.
func (c *ctxt7) con32class(a *obj.Addr) int {
	v := uint32(a.Offset)
	// For 32-bit instruction with constant, rewrite
	// the high 32-bit to be a repetition of the low
	// 32-bit, so that the BITCON test can be shared
	// for both 32-bit and 64-bit. 32-bit ops will
	// zero the high 32-bit of the destination register
	// anyway.
	vbitcon := uint64(v)<<32 | uint64(v)
	if v == 0 {
		return C_ZCON
	}
	if isaddcon(int64(v)) {
		if v <= 0xFFF {
			if isbitcon(vbitcon) {
				return C_ABCON0
			}
			return C_ADDCON0
		}
		if isbitcon(vbitcon) {
			return C_ABCON
		}
		if movcon(int64(v)) >= 0 {
			return C_AMCON
		}
		if movcon(int64(^v)) >= 0 {
			return C_AMCON
		}
		return C_ADDCON
	}

	t := movcon(int64(v))
	if t >= 0 {
		if isbitcon(vbitcon) {
			return C_MBCON
		}
		return C_MOVCON
	}

	t = movcon(int64(^v))
	if t >= 0 {
		if isbitcon(vbitcon) {
			return C_MBCON
		}
		return C_MOVCON
	}

	if isbitcon(vbitcon) {
		return C_BITCON
	}

	if 0 <= v && v <= 0xffffff {
		return C_ADDCON2
	}
	return C_LCON
}

// con64class reclassifies the constant of C_VCON and C_LCON class.
func (c *ctxt7) con64class(a *obj.Addr) int {
	zeroCount := 0
	negCount := 0
	for i := uint(0); i < 4; i++ {
		immh := uint32(a.Offset >> (i * 16) & 0xffff)
		if immh == 0 {
			zeroCount++
		} else if immh == 0xffff {
			negCount++
		}
	}
	if zeroCount >= 3 || negCount >= 3 {
		return C_MOVCON
	} else if zeroCount == 2 || negCount == 2 {
		return C_MOVCON2
	} else if zeroCount == 1 || negCount == 1 {
		return C_MOVCON3
	} else {
		return C_VCON
	}
}

// loadStoreClass reclassifies a load or store operation based on its offset.
func (c *ctxt7) loadStoreClass(p *obj.Prog, lsc int, v int64) int {
	// Avoid reclassification of pre/post-indexed loads and stores.
	if p.Scond == C_XPRE || p.Scond == C_XPOST {
		return lsc
	}
	if cmp(C_NSAUTO, lsc) || cmp(C_NSOREG, lsc) {
		return lsc
	}

	needsPool := true
	if v >= -4095 && v <= 4095 {
		needsPool = false
	}

	switch p.As {
	case AMOVB, AMOVBU:
		if cmp(C_UAUTO4K, lsc) || cmp(C_UOREG4K, lsc) {
			return lsc
		}
		if v >= 0 && v <= 0xffffff {
			needsPool = false
		}
	case AMOVH, AMOVHU:
		if cmp(C_UAUTO8K, lsc) || cmp(C_UOREG8K, lsc) {
			return lsc
		}
		if v >= 0 && v <= 0xfff000+0xfff<<1 && v&1 == 0 {
			needsPool = false
		}
	case AMOVW, AMOVWU, AFMOVS:
		if cmp(C_UAUTO16K, lsc) || cmp(C_UOREG16K, lsc) {
			return lsc
		}
		if v >= 0 && v <= 0xfff000+0xfff<<2 && v&3 == 0 {
			needsPool = false
		}
	case AMOVD, AFMOVD:
		if cmp(C_UAUTO32K, lsc) || cmp(C_UOREG32K, lsc) {
			return lsc
		}
		if v >= 0 && v <= 0xfff000+0xfff<<3 && v&7 == 0 {
			needsPool = false
		}
	case AFMOVQ:
		if cmp(C_UAUTO64K, lsc) || cmp(C_UOREG64K, lsc) {
			return lsc
		}
		if v >= 0 && v <= 0xfff000+0xfff<<4 && v&15 == 0 {
			needsPool = false
		}
	}
	if needsPool && cmp(C_LAUTO, lsc) {
		return C_LAUTOPOOL
	}
	if needsPool && cmp(C_LOREG, lsc) {
		return C_LOREGPOOL
	}
	return lsc
}

// loadStorePairClass reclassifies a load or store pair operation based on its offset.
func (c *ctxt7) loadStorePairClass(p *obj.Prog, lsc int, v int64) int {
	// Avoid reclassification of pre/post-indexed loads and stores.
	if p.Scond == C_XPRE || p.Scond == C_XPOST {
		return lsc
	}

	if cmp(C_NAUTO4K, lsc) || cmp(C_NOREG4K, lsc) {
		return lsc
	}
	if cmp(C_UAUTO4K, lsc) || cmp(C_UOREG4K, lsc) {
		return lsc
	}

	needsPool := true
	if v >= 0 && v <= 0xffffff {
		needsPool = false
	}
	if needsPool && cmp(C_LAUTO, lsc) {
		return C_LAUTOPOOL
	}
	if needsPool && cmp(C_LOREG, lsc) {
		return C_LOREGPOOL
	}
	return lsc
}

func (c *ctxt7) aclass(a *obj.Addr) int {
	switch a.Type {
	case obj.TYPE_NONE:
		return C_NONE

	case obj.TYPE_REG:
		return rclass(a.Reg)

	case obj.TYPE_REGREG:
		return C_PAIR

	case obj.TYPE_SHIFT:
		return C_SHIFT

	case obj.TYPE_REGLIST:
		return C_LIST

	case obj.TYPE_MEM:
		// The base register should be an integer register.
		if int16(REG_F0) <= a.Reg && a.Reg <= int16(REG_V31) {
			break
		}
		switch a.Name {
		case obj.NAME_EXTERN, obj.NAME_STATIC:
			if a.Sym == nil {
				break
			}
			c.instoffset = a.Offset
			if a.Sym != nil { // use relocation
				if a.Sym.Type == objabi.STLSBSS {
					if c.ctxt.Flag_shared {
						return C_TLS_IE
					} else {
						return C_TLS_LE
					}
				}
				return C_ADDR
			}
			return C_LEXT

		case obj.NAME_GOTREF:
			return C_GOTADDR

		case obj.NAME_AUTO:
			if a.Reg == REGSP {
				// unset base register for better printing, since
				// a.Offset is still relative to pseudo-SP.
				a.Reg = obj.REG_NONE
			}
			// The frame top 8 or 16 bytes are for FP
			c.instoffset = int64(c.autosize) + a.Offset - int64(c.extrasize)
			return autoclass(c.instoffset)

		case obj.NAME_PARAM:
			if a.Reg == REGSP {
				// unset base register for better printing, since
				// a.Offset is still relative to pseudo-FP.
				a.Reg = obj.REG_NONE
			}
			c.instoffset = int64(c.autosize) + a.Offset + 8
			return autoclass(c.instoffset)

		case obj.NAME_NONE:
			if a.Index != 0 {
				if a.Offset != 0 {
					if isRegShiftOrExt(a) {
						// extended or shifted register offset, (Rn)(Rm.UXTW<<2) or (Rn)(Rm<<2).
						return C_ROFF
					}
					return C_GOK
				}
				// register offset, (Rn)(Rm)
				return C_ROFF
			}
			c.instoffset = a.Offset
			return oregclass(c.instoffset)
		}
		return C_GOK

	case obj.TYPE_FCONST:
		return C_FCON

	case obj.TYPE_TEXTSIZE:
		return C_TEXTSIZE

	case obj.TYPE_CONST, obj.TYPE_ADDR:
		switch a.Name {
		case obj.NAME_NONE:
			c.instoffset = a.Offset
			if a.Reg != 0 && a.Reg != REGZERO {
				break
			}
			v := c.instoffset
			if v == 0 {
				return C_ZCON
			}
			if isaddcon(v) {
				if v <= 0xFFF {
					if isbitcon(uint64(v)) {
						return C_ABCON0
					}
					return C_ADDCON0
				}
				if isbitcon(uint64(v)) {
					return C_ABCON
				}
				if movcon(v) >= 0 {
					return C_AMCON
				}
				if movcon(^v) >= 0 {
					return C_AMCON
				}
				return C_ADDCON
			}

			t := movcon(v)
			if t >= 0 {
				if isbitcon(uint64(v)) {
					return C_MBCON
				}
				return C_MOVCON
			}

			t = movcon(^v)
			if t >= 0 {
				if isbitcon(uint64(v)) {
					return C_MBCON
				}
				return C_MOVCON
			}

			if isbitcon(uint64(v)) {
				return C_BITCON
			}

			if 0 <= v && v <= 0xffffff {
				return C_ADDCON2
			}

			if uint64(v) == uint64(uint32(v)) || v == int64(int32(v)) {
				return C_LCON
			}
			return C_VCON

		case obj.NAME_EXTERN, obj.NAME_STATIC:
			if a.Sym == nil {
				return C_GOK
			}
			if a.Sym.Type == objabi.STLSBSS {
				c.ctxt.Diag("taking address of TLS variable is not supported")
			}
			c.instoffset = a.Offset
			return C_VCONADDR

		case obj.NAME_AUTO:
			if a.Reg == REGSP {
				// unset base register for better printing, since
				// a.Offset is still relative to pseudo-SP.
				a.Reg = obj.REG_NONE
			}
			// The frame top 8 or 16 bytes are for FP
			c.instoffset = int64(c.autosize) + a.Offset - int64(c.extrasize)

		case obj.NAME_PARAM:
			if a.Reg == REGSP {
				// unset base register for better printing, since
				// a.Offset is still relative to pseudo-FP.
				a.Reg = obj.REG_NONE
			}
			c.instoffset = int64(c.autosize) + a.Offset + 8
		default:
			return C_GOK
		}
		cf := c.instoffset
		if isaddcon(cf) || isaddcon(-cf) {
			return C_AACON
		}
		if isaddcon2(cf) {
			return C_AACON2
		}

		return C_LACON

	case obj.TYPE_BRANCH:
		return C_SBRA

	case obj.TYPE_SPECIAL:
		opd := SpecialOperand(a.Offset)
		if SPOP_EQ <= opd && opd <= SPOP_NV {
			return C_COND
		}
		return C_SPOP
	}
	return C_GOK
}

func (c *ctxt7) oplook(p *obj.Prog) *Optab {
	a1 := int(p.Optab)
	if a1 != 0 {
		return &optab[a1-1]
	}
	a1 = int(p.From.Class)
	if a1 == 0 {
		a1 = c.aclass(&p.From)
		// do not break C_ADDCON2 when S bit is set
		if (p.As == AADDS || p.As == AADDSW || p.As == ASUBS || p.As == ASUBSW) && a1 == C_ADDCON2 {
			a1 = C_LCON
		}
		if p.From.Type == obj.TYPE_CONST && p.From.Name == obj.NAME_NONE {
			if p.As == AMOVW || isADDWop(p.As) || isANDWop(p.As) {
				// For 32-bit instruction with constant, we need to
				// treat its offset value as 32 bits to classify it.
				a1 = c.con32class(&p.From)
				// do not break C_ADDCON2 when S bit is set
				if (p.As == AADDSW || p.As == ASUBSW) && a1 == C_ADDCON2 {
					a1 = C_LCON
				}
			}
			if ((p.As == AMOVD) || isANDop(p.As) || isADDop(p.As)) && (a1 == C_LCON || a1 == C_VCON) {
				// more specific classification of 64-bit integers
				a1 = c.con64class(&p.From)
			}
		}
		if p.From.Type == obj.TYPE_MEM {
			if isMOVop(p.As) && (cmp(C_LAUTO, a1) || cmp(C_LOREG, a1)) {
				// More specific classification of large offset loads and stores.
				a1 = c.loadStoreClass(p, a1, c.instoffset)
			}
			if isLoadStorePairOp(p.As) && (cmp(C_LAUTO, a1) || cmp(C_LOREG, a1)) {
				// More specific classification of large offset loads and stores.
				a1 = c.loadStorePairClass(p, a1, c.instoffset)
			}
		}
		p.From.Class = int8(a1)
	}

	a2 := C_NONE
	if p.Reg != 0 {
		a2 = rclass(p.Reg)
	}

	a3 := C_NONE
	if p.GetFrom3() != nil {
		a3 = int(p.GetFrom3().Class)
		if a3 == 0 {
			a3 = c.aclass(p.GetFrom3())
			p.GetFrom3().Class = int8(a3)
		}
	}

	a4 := int(p.To.Class)
	if a4 == 0 {
		a4 = c.aclass(&p.To)
		if p.To.Type == obj.TYPE_MEM {
			if isMOVop(p.As) && (cmp(C_LAUTO, a4) || cmp(C_LOREG, a4)) {
				// More specific classification of large offset loads and stores.
				a4 = c.loadStoreClass(p, a4, c.instoffset)
			}
			if isLoadStorePairOp(p.As) && (cmp(C_LAUTO, a4) || cmp(C_LOREG, a4)) {
				// More specific classification of large offset loads and stores.
				a4 = c.loadStorePairClass(p, a4, c.instoffset)
			}
		}
		p.To.Class = int8(a4)
	}

	a5 := C_NONE
	if p.RegTo2 != 0 {
		a5 = rclass(p.RegTo2)
	} else if p.GetTo2() != nil {
		a5 = int(p.GetTo2().Class)
		if a5 == 0 {
			a5 = c.aclass(p.GetTo2())
			p.GetTo2().Class = int8(a5)
		}
	}

	if false {
		fmt.Printf("oplook %v %d %d %d %d %d\n", p.As, a1, a2, a3, a4, a5)
		fmt.Printf("\t\t%d %d\n", p.From.Type, p.To.Type)
	}

	ops := oprange[p.As&obj.AMask]
	c1 := &xcmp[a1]
	c2 := &xcmp[a2]
	c3 := &xcmp[a3]
	c4 := &xcmp[a4]
	c5 := &xcmp[a5]
	for i := range ops {
		op := &ops[i]
		if c1[op.a1] && c2[op.a2] && c3[op.a3] && c4[op.a4] && c5[op.a5] && p.Scond == op.scond {
			p.Optab = uint16(cap(optab) - cap(ops) + i + 1)
			return op
		}
	}

	c.ctxt.Diag("illegal combination: %v %v %v %v %v %v, %s %s", p, DRconv(a1), DRconv(a2), DRconv(a3), DRconv(a4), DRconv(a5), p.From.Type.String(), p.To.Type.String())
	// Turn illegal instruction into an UNDEF, avoid crashing in asmout
	return &Optab{obj.AUNDEF, C_NONE, C_NONE, C_NONE, C_NONE, C_NONE, 90, 4, 0, 0, 0}
}

func cmp(a int, b int) bool {
	if a == b {
		return true
	}
	switch a {
	case C_RSP:
		if b == C_REG {
			return true
		}

	case C_REGZR:
		if b == C_REG {
			return true
		}

	case C_ADDCON0:
		if b == C_ZCON || b == C_ABCON0 {
			return true
		}

	case C_ADDCON:
		if b == C_ZCON || b == C_ABCON0 || b == C_ADDCON0 || b == C_ABCON || b == C_AMCON {
			return true
		}

	case C_MBCON:
		if b == C_ABCON0 {
			return true
		}

	case C_BITCON:
		if b == C_ABCON0 || b == C_ABCON || b == C_MBCON {
			return true
		}

	case C_MOVCON:
		if b == C_MBCON || b == C_ZCON || b == C_ADDCON0 || b == C_ABCON0 || b == C_AMCON {
			return true
		}

	case C_ADDCON2:
		if b == C_ZCON || b == C_ADDCON || b == C_ADDCON0 {
			return true
		}

	case C_LCON:
		if b == C_ZCON || b == C_BITCON || b == C_ADDCON || b == C_ADDCON0 || b == C_ABCON || b == C_ABCON0 || b == C_MBCON || b == C_MOVCON || b == C_ADDCON2 || b == C_AMCON {
			return true
		}

	case C_MOVCON2:
		return cmp(C_LCON, b)

	case C_VCON:
		return cmp(C_LCON, b)

	case C_LACON:
		if b == C_AACON || b == C_AACON2 {
			return true
		}

	case C_SEXT2:
		if b == C_SEXT1 {
			return true
		}

	case C_SEXT4:
		if b == C_SEXT1 || b == C_SEXT2 {
			return true
		}

	case C_SEXT8:
		if b >= C_SEXT1 && b <= C_SEXT4 {
			return true
		}

	case C_SEXT16:
		if b >= C_SEXT1 && b <= C_SEXT8 {
			return true
		}

	case C_LEXT:
		if b >= C_SEXT1 && b <= C_SEXT16 {
			return true
		}

	case C_NSAUTO_8:
		if b == C_NSAUTO_16 {
			return true
		}

	case C_NSAUTO_4:
		if b == C_NSAUTO_16 || b == C_NSAUTO_8 {
			return true
		}

	case C_NSAUTO:
		switch b {
		case C_NSAUTO_4, C_NSAUTO_8, C_NSAUTO_16:
			return true
		}

	case C_NPAUTO_16:
		switch b {
		case C_NSAUTO_16:
			return true
		}

	case C_NPAUTO:
		switch b {
		case C_NSAUTO_16, C_NSAUTO_8, C_NPAUTO_16:
			return true
		}

	case C_NQAUTO_16:
		switch b {
		case C_NSAUTO_16, C_NPAUTO_16:
			return true
		}

	case C_NAUTO4K:
		switch b {
		case C_NSAUTO_16, C_NSAUTO_8, C_NSAUTO_4, C_NSAUTO, C_NPAUTO_16,
			C_NPAUTO, C_NQAUTO_16:
			return true
		}

	case C_PSAUTO_16:
		if b == C_ZAUTO {
			return true
		}

	case C_PSAUTO_8:
		if b == C_ZAUTO || b == C_PSAUTO_16 {
			return true
		}

	case C_PSAUTO_4:
		switch b {
		case C_ZAUTO, C_PSAUTO_16, C_PSAUTO_8:
			return true
		}

	case C_PSAUTO:
		switch b {
		case C_ZAUTO, C_PSAUTO_16, C_PSAUTO_8, C_PSAUTO_4:
			return true
		}

	case C_PPAUTO_16:
		switch b {
		case C_ZAUTO, C_PSAUTO_16:
			return true
		}

	case C_PPAUTO:
		switch b {
		case C_ZAUTO, C_PSAUTO_16, C_PSAUTO_8, C_PPAUTO_16:
			return true
		}

	case C_PQAUTO_16:
		switch b {
		case C_ZAUTO, C_PSAUTO_16, C_PPAUTO_16:
			return true
		}

	case C_UAUTO4K:
		switch b {
		case C_ZAUTO, C_PSAUTO, C_PSAUTO_4, C_PSAUTO_8, C_PSAUTO_16,
			C_PPAUTO, C_PPAUTO_16, C_PQAUTO_16,
			C_UAUTO4K_2, C_UAUTO4K_4, C_UAUTO4K_8, C_UAUTO4K_16:
			return true
		}

	case C_UAUTO8K:
		switch b {
		case C_ZAUTO, C_PSAUTO, C_PSAUTO_4, C_PSAUTO_8, C_PSAUTO_16,
			C_PPAUTO, C_PPAUTO_16, C_PQAUTO_16,
			C_UAUTO4K_2, C_UAUTO4K_4, C_UAUTO4K_8, C_UAUTO4K_16,
			C_UAUTO8K_4, C_UAUTO8K_8, C_UAUTO8K_16:
			return true
		}

	case C_UAUTO16K:
		switch b {
		case C_ZAUTO, C_PSAUTO, C_PSAUTO_4, C_PSAUTO_8, C_PSAUTO_16,
			C_PPAUTO, C_PPAUTO_16, C_PQAUTO_16,
			C_UAUTO4K_4, C_UAUTO4K_8, C_UAUTO4K_16,
			C_UAUTO8K_4, C_UAUTO8K_8, C_UAUTO8K_16,
			C_UAUTO16K_8, C_UAUTO16K_16:
			return true
		}

	case C_UAUTO32K:
		switch b {
		case C_ZAUTO, C_PSAUTO, C_PSAUTO_4, C_PSAUTO_8, C_PSAUTO_16,
			C_PPAUTO, C_PPAUTO_16, C_PQAUTO_16,
			C_UAUTO4K_8, C_UAUTO4K_16,
			C_UAUTO8K_8, C_UAUTO8K_16,
			C_UAUTO16K_8, C_UAUTO16K_16,
			C_UAUTO32K_16:
			return true
		}

	case C_UAUTO64K:
		switch b {
		case C_ZAUTO, C_PSAUTO, C_PSAUTO_4, C_PSAUTO_8, C_PSAUTO_16,
			C_PPAUTO_16, C_PQAUTO_16, C_UAUTO4K_16, C_UAUTO8K_16, C_UAUTO16K_16,
			C_UAUTO32K_16:
			return true
		}

	case C_LAUTO:
		switch b {
		case C_ZAUTO, C_NSAUTO, C_NSAUTO_4, C_NSAUTO_8, C_NSAUTO_16, C_NPAUTO_16, C_NPAUTO, C_NQAUTO_16, C_NAUTO4K,
			C_PSAUTO, C_PSAUTO_4, C_PSAUTO_8, C_PSAUTO_16,
			C_PPAUTO, C_PPAUTO_16, C_PQAUTO_16,
			C_UAUTO4K, C_UAUTO4K_2, C_UAUTO4K_4, C_UAUTO4K_8, C_UAUTO4K_16,
			C_UAUTO8K, C_UAUTO8K_4, C_UAUTO8K_8, C_UAUTO8K_16,
			C_UAUTO16K, C_UAUTO16K_8, C_UAUTO16K_16,
			C_UAUTO32K, C_UAUTO32K_16,
			C_UAUTO64K:
			return true
		}

	case C_NSOREG_8:
		if b == C_NSOREG_16 {
			return true
		}

	case C_NSOREG_4:
		if b == C_NSOREG_8 || b == C_NSOREG_16 {
			return true
		}

	case C_NSOREG:
		switch b {
		case C_NSOREG_4, C_NSOREG_8, C_NSOREG_16:
			return true
		}

	case C_NPOREG_16:
		switch b {
		case C_NSOREG_16:
			return true
		}

	case C_NPOREG:
		switch b {
		case C_NSOREG_16, C_NSOREG_8, C_NPOREG_16:
			return true
		}

	case C_NQOREG_16:
		switch b {
		case C_NSOREG_16, C_NPOREG_16:
			return true
		}

	case C_NOREG4K:
		switch b {
		case C_NSOREG_16, C_NSOREG_8, C_NSOREG_4, C_NSOREG, C_NPOREG_16, C_NPOREG, C_NQOREG_16:
			return true
		}

	case C_PSOREG_16:
		if b == C_ZOREG {
			return true
		}

	case C_PSOREG_8:
		if b == C_ZOREG || b == C_PSOREG_16 {
			return true
		}

	case C_PSOREG_4:
		switch b {
		case C_ZOREG, C_PSOREG_16, C_PSOREG_8:
			return true
		}

	case C_PSOREG:
		switch b {
		case C_ZOREG, C_PSOREG_16, C_PSOREG_8, C_PSOREG_4:
			return true
		}

	case C_PPOREG_16:
		switch b {
		case C_ZOREG, C_PSOREG_16:
			return true
		}

	case C_PPOREG:
		switch b {
		case C_ZOREG, C_PSOREG_16, C_PSOREG_8, C_PPOREG_16:
			return true
		}

	case C_PQOREG_16:
		switch b {
		case C_ZOREG, C_PSOREG_16, C_PPOREG_16:
			return true
		}

	case C_UOREG4K:
		switch b {
		case C_ZOREG, C_PSOREG, C_PSOREG_4, C_PSOREG_8, C_PSOREG_16,
			C_PPOREG, C_PPOREG_16, C_PQOREG_16,
			C_UOREG4K_2, C_UOREG4K_4, C_UOREG4K_8, C_UOREG4K_16:
			return true
		}

	case C_UOREG8K:
		switch b {
		case C_ZOREG, C_PSOREG, C_PSOREG_4, C_PSOREG_8, C_PSOREG_16,
			C_PPOREG, C_PPOREG_16, C_PQOREG_16,
			C_UOREG4K_2, C_UOREG4K_4, C_UOREG4K_8, C_UOREG4K_16,
			C_UOREG8K_4, C_UOREG8K_8, C_UOREG8K_16:
			return true
		}

	case C_UOREG16K:
		switch b {
		case C_ZOREG, C_PSOREG, C_PSOREG_4, C_PSOREG_8, C_PSOREG_16,
			C_PPOREG, C_PPOREG_16, C_PQOREG_16,
			C_UOREG4K_4, C_UOREG4K_8, C_UOREG4K_16,
			C_UOREG8K_4, C_UOREG8K_8, C_UOREG8K_16,
			C_UOREG16K_8, C_UOREG16K_16:
			return true
		}

	case C_UOREG32K:
		switch b {
		case C_ZOREG, C_PSOREG, C_PSOREG_4, C_PSOREG_8, C_PSOREG_16,
			C_PPOREG, C_PPOREG_16, C_PQOREG_16,
			C_UOREG4K_8, C_UOREG4K_16,
			C_UOREG8K_8, C_UOREG8K_16,
			C_UOREG16K_8, C_UOREG16K_16,
			C_UOREG32K_16:
			return true
		}

	case C_UOREG64K:
		switch b {
		case C_ZOREG, C_PSOREG, C_PSOREG_4, C_PSOREG_8, C_PSOREG_16,
			C_PPOREG_16, C_PQOREG_16, C_UOREG4K_16, C_UOREG8K_16, C_UOREG16K_16,
			C_UOREG32K_16:
			return true
		}

	case C_LOREG:
		switch b {
		case C_ZOREG, C_NSOREG, C_NSOREG_4, C_NSOREG_8, C_NSOREG_16, C_NPOREG, C_NPOREG_16, C_NQOREG_16, C_NOREG4K,
			C_PSOREG, C_PSOREG_4, C_PSOREG_8, C_PSOREG_16,
			C_PPOREG, C_PPOREG_16, C_PQOREG_16,
			C_UOREG4K, C_UOREG4K_2, C_UOREG4K_4, C_UOREG4K_8, C_UOREG4K_16,
			C_UOREG8K, C_UOREG8K_4, C_UOREG8K_8, C_UOREG8K_16,
			C_UOREG16K, C_UOREG16K_8, C_UOREG16K_16,
			C_UOREG32K, C_UOREG32K_16,
			C_UOREG64K:
			return true
		}

	case C_LBRA:
		if b == C_SBRA {
			return true
		}
	}

	return false
}

func ocmp(p1, p2 Optab) int {
	if p1.as != p2.as {
		return int(p1.as) - int(p2.as)
	}
	if p1.a1 != p2.a1 {
		return int(p1.a1) - int(p2.a1)
	}
	if p1.a2 != p2.a2 {
		return int(p1.a2) - int(p2.a2)
	}
	if p1.a3 != p2.a3 {
		return int(p1.a3) - int(p2.a3)
	}
	if p1.a4 != p2.a4 {
		return int(p1.a4) - int(p2.a4)
	}
	if p1.scond != p2.scond {
		return int(p1.scond) - int(p2.scond)
	}
	return 0
}

func oprangeset(a obj.As, t []Optab) {
	oprange[a&obj.AMask] = t
}

func buildop(ctxt *obj.Link) {
	if oprange[AAND&obj.AMask] != nil {
		// Already initialized; stop now.
		// This happens in the cmd/asm tests,
		// each of which re-initializes the arch.
		return
	}

	for i := 0; i < C_GOK; i++ {
		for j := 0; j < C_GOK; j++ {
			if cmp(j, i) {
				xcmp[i][j] = true
			}
		}
	}

	slices.SortFunc(optab, ocmp)
	for i := 0; i < len(optab); i++ {
		as, start := optab[i].as, i
		for ; i < len(optab)-1; i++ {
			if optab[i+1].as != as {
				break
			}
		}
		t := optab[start : i+1]
		oprangeset(as, t)
		switch as {
		default:
			ctxt.Diag("unknown op in build: %v", as)
			ctxt.DiagFlush()
			log.Fatalf("bad code")

		case AADD:
			oprangeset(AADDS, t)
			oprangeset(ASUB, t)
			oprangeset(ASUBS, t)
			oprangeset(AADDW, t)
			oprangeset(AADDSW, t)
			oprangeset(ASUBW, t)
			oprangeset(ASUBSW, t)

		case AAND: /* logical immediate, logical shifted register */
			oprangeset(AANDW, t)
			oprangeset(AEOR, t)
			oprangeset(AEORW, t)
			oprangeset(AORR, t)
			oprangeset(AORRW, t)
			oprangeset(ABIC, t)
			oprangeset(ABICW, t)
			oprangeset(AEON, t)
			oprangeset(AEONW, t)
			oprangeset(AORN, t)
			oprangeset(AORNW, t)

		case AANDS: /* logical immediate, logical shifted register, set flags, cannot target RSP */
			oprangeset(AANDSW, t)
			oprangeset(ABICS, t)
			oprangeset(ABICSW, t)

		case ANEG:
			oprangeset(ANEGS, t)
			oprangeset(ANEGSW, t)
			oprangeset(ANEGW, t)

		case AADC: /* rn=Rd */
			oprangeset(AADCW, t)

			oprangeset(AADCS, t)
			oprangeset(AADCSW, t)
			oprangeset(ASBC, t)
			oprangeset(ASBCW, t)
			oprangeset(ASBCS, t)
			oprangeset(ASBCSW, t)

		case ANGC: /* rn=REGZERO */
			oprangeset(ANGCW, t)

			oprangeset(ANGCS, t)
			oprangeset(ANGCSW, t)

		case ACMP:
			oprangeset(ACMPW, t)
			oprangeset(ACMN, t)
			oprangeset(ACMNW, t)

		case ATST:
			oprangeset(ATSTW, t)

			/* register/register, and shifted */
		case AMVN:
			oprangeset(AMVNW, t)

		case AMOVK:
			oprangeset(AMOVKW, t)
			oprangeset(AMOVN, t)
			oprangeset(AMOVNW, t)
			oprangeset(AMOVZ, t)
			oprangeset(AMOVZW, t)

		case ASWPD:
			for i := range atomicLDADD {
				oprangeset(i, t)
			}
			for i := range atomicSWP {
				if i == ASWPD {
					continue
				}
				oprangeset(i, t)
			}

		case ACASPD:
			oprangeset(ACASPW, t)
		case ABEQ:
			oprangeset(ABNE, t)
			oprangeset(ABCS, t)
			oprangeset(ABHS, t)
			oprangeset(ABCC, t)
			oprangeset(ABLO, t)
			oprangeset(ABMI, t)
			oprangeset(ABPL, t)
			oprangeset(ABVS, t)
			oprangeset(ABVC, t)
			oprangeset(ABHI, t)
			oprangeset(ABLS, t)
			oprangeset(ABGE, t)
			oprangeset(ABLT, t)
			oprangeset(ABGT, t)
			oprangeset(ABLE, t)

		case ALSL:
			oprangeset(ALSLW, t)
			oprangeset(ALSR, t)
			oprangeset(ALSRW, t)
			oprangeset(AASR, t)
			oprangeset(AASRW, t)
			oprangeset(AROR, t)
			oprangeset(ARORW, t)

		case ACLS:
			oprangeset(ACLSW, t)
			oprangeset(ACLZ, t)
			oprangeset(ACLZW, t)
			oprangeset(ARBIT, t)
			oprangeset(ARBITW, t)
			oprangeset(AREV, t)
			oprangeset(AREVW, t)
			oprangeset(AREV16, t)
			oprangeset(AREV16W, t)
			oprangeset(AREV32, t)

		case ASDIV:
			oprangeset(ASDIVW, t)
			oprangeset(AUDIV, t)
			oprangeset(AUDIVW, t)
			oprangeset(ACRC32B, t)
			oprangeset(ACRC32CB, t)
			oprangeset(ACRC32CH, t)
			oprangeset(ACRC32CW, t)
			oprangeset(ACRC32CX, t)
			oprangeset(ACRC32H, t)
			oprangeset(ACRC32W, t)
			oprangeset(ACRC32X, t)

		case AMADD:
			oprangeset(AMADDW, t)
			oprangeset(AMSUB, t)
			oprangeset(AMSUBW, t)
			oprangeset(ASMADDL, t)
			oprangeset(ASMSUBL, t)
			oprangeset(AUMADDL, t)
			oprangeset(AUMSUBL, t)

		case AREM:
			oprangeset(AREMW, t)
			oprangeset(AUREM, t)
			oprangeset(AUREMW, t)

		case AMUL:
			oprangeset(AMULW, t)
			oprangeset(AMNEG, t)
			oprangeset(AMNEGW, t)
			oprangeset(ASMNEGL, t)
			oprangeset(ASMULL, t)
			oprangeset(ASMULH, t)
			oprangeset(AUMNEGL, t)
			oprangeset(AUMULH, t)
			oprangeset(AUMULL, t)

		case AMOVB:
			oprangeset(AMOVBU, t)

		case AMOVH:
			oprangeset(AMOVHU, t)

		case AMOVW:
			oprangeset(AMOVWU, t)

		case ABFM:
			oprangeset(ABFMW, t)
			oprangeset(ASBFM, t)
			oprangeset(ASBFMW, t)
			oprangeset(AUBFM, t)
			oprangeset(AUBFMW, t)

		case ABFI:
			oprangeset(ABFIW, t)
			oprangeset(ABFXIL, t)
			oprangeset(ABFXILW, t)
			oprangeset(ASBFIZ, t)
			oprangeset(ASBFIZW, t)
			oprangeset(ASBFX, t)
			oprangeset(ASBFXW, t)
			oprangeset(AUBFIZ, t)
			oprangeset(AUBFIZW, t)
			oprangeset(AUBFX, t)
			oprangeset(AUBFXW, t)

		case AEXTR:
			oprangeset(AEXTRW, t)

		case ASXTB:
			oprangeset(ASXTBW, t)
			oprangeset(ASXTH, t)
			oprangeset(ASXTHW, t)
			oprangeset(ASXTW, t)
			oprangeset(AUXTB, t)
			oprangeset(AUXTH, t)
			oprangeset(AUXTW, t)
			oprangeset(AUXTBW, t)
			oprangeset(AUXTHW, t)

		case ACCMN:
			oprangeset(ACCMNW, t)
			oprangeset(ACCMP, t)
			oprangeset(ACCMPW, t)

		case ACSEL:
			oprangeset(ACSELW, t)
			oprangeset(ACSINC, t)
			oprangeset(ACSINCW, t)
			oprangeset(ACSINV, t)
			oprangeset(ACSINVW, t)
			oprangeset(ACSNEG, t)
			oprangeset(ACSNEGW, t)

		case ACINC:
			// aliases Rm=Rn, !cond
			oprangeset(ACINCW, t)
			oprangeset(ACINV, t)
			oprangeset(ACINVW, t)
			oprangeset(ACNEG, t)
			oprangeset(ACNEGW, t)

			// aliases, Rm=Rn=REGZERO, !cond
		case ACSET:
			oprangeset(ACSETW, t)

			oprangeset(ACSETM, t)
			oprangeset(ACSETMW, t)

		case AMOVD,
			AB,
			ABL,
			AWORD,
			ADWORD,
			obj.ARET,
			obj.ATEXT:
			break

		case AFLDPQ:
			break
		case AFSTPQ:
			break
		case ALDP:
			oprangeset(AFLDPD, t)

		case ASTP:
			oprangeset(AFSTPD, t)

		case ASTPW:
			oprangeset(AFSTPS, t)

		case ALDPW:
			oprangeset(ALDPSW, t)
			oprangeset(AFLDPS, t)

		case AERET:
			oprangeset(AWFE, t)
			oprangeset(AWFI, t)
			oprangeset(AYIELD, t)
			oprangeset(ASEV, t)
			oprangeset(ASEVL, t)
			oprangeset(ANOOP, t)
			oprangeset(ADRPS, t)

		case ACBZ:
			oprangeset(ACBZW, t)
			oprangeset(ACBNZ, t)
			oprangeset(ACBNZW, t)

		case ATBZ:
			oprangeset(ATBNZ, t)

		case AADR, AADRP:
			break

		case ACLREX:
			break

		case ASVC:
			oprangeset(AHVC, t)
			oprangeset(AHLT, t)
			oprangeset(ASMC, t)
			oprangeset(ABRK, t)
			oprangeset(ADCPS1, t)
			oprangeset(ADCPS2, t)
			oprangeset(ADCPS3, t)

		case AFADDS:
			oprangeset(AFADDD, t)
			oprangeset(AFSUBS, t)
			oprangeset(AFSUBD, t)
			oprangeset(AFMULS, t)
			oprangeset(AFMULD, t)
			oprangeset(AFNMULS, t)
			oprangeset(AFNMULD, t)
			oprangeset(AFDIVS, t)
			oprangeset(AFMAXD, t)
			oprangeset(AFMAXS, t)
			oprangeset(AFMIND, t)
			oprangeset(AFMINS, t)
			oprangeset(AFMAXNMD, t)
			oprangeset(AFMAXNMS, t)
			oprangeset(AFMINNMD, t)
			oprangeset(AFMINNMS, t)
			oprangeset(AFDIVD, t)

		case AFMSUBD:
			oprangeset(AFMSUBS, t)
			oprangeset(AFMADDS, t)
			oprangeset(AFMADDD, t)
			oprangeset(AFNMSUBS, t)
			oprangeset(AFNMSUBD, t)
			oprangeset(AFNMADDS, t)
			oprangeset(AFNMADDD, t)

		case AFCVTSD:
			oprangeset(AFCVTDS, t)
			oprangeset(AFABSD, t)
			oprangeset(AFABSS, t)
			oprangeset(AFNEGD, t)
			oprangeset(AFNEGS, t)
			oprangeset(AFSQRTD, t)
			oprangeset(AFSQRTS, t)
			oprangeset(AFRINTNS, t)
			oprangeset(AFRINTND, t)
			oprangeset(AFRINTPS, t)
			oprangeset(AFRINTPD, t)
			oprangeset(AFRINTMS, t)
			oprangeset(AFRINTMD, t)
			oprangeset(AFRINTZS, t)
			oprangeset(AFRINTZD, t)
			oprangeset(AFRINTAS, t)
			oprangeset(AFRINTAD, t)
			oprangeset(AFRINTXS, t)
			oprangeset(AFRINTXD, t)
			oprangeset(AFRINTIS, t)
			oprangeset(AFRINTID, t)
			oprangeset(AFCVTDH, t)
			oprangeset(AFCVTHS, t)
			oprangeset(AFCVTHD, t)
			oprangeset(AFCVTSH, t)

		case AFCMPS:
			oprangeset(AFCMPD, t)
			oprangeset(AFCMPES, t)
			oprangeset(AFCMPED, t)

		case AFCCMPS:
			oprangeset(AFCCMPD, t)
			oprangeset(AFCCMPES, t)
			oprangeset(AFCCMPED, t)

		case AFCSELD:
			oprangeset(AFCSELS, t)

		case AFMOVQ, AFMOVD, AFMOVS,
			AVMOVQ, AVMOVD, AVMOVS:
			break

		case AFCVTZSD:
			oprangeset(AFCVTZSDW, t)
			oprangeset(AFCVTZSS, t)
			oprangeset(AFCVTZSSW, t)
			oprangeset(AFCVTZUD, t)
			oprangeset(AFCVTZUDW, t)
			oprangeset(AFCVTZUS, t)
			oprangeset(AFCVTZUSW, t)

		case ASCVTFD:
			oprangeset(ASCVTFS, t)
			oprangeset(ASCVTFWD, t)
			oprangeset(ASCVTFWS, t)
			oprangeset(AUCVTFD, t)
			oprangeset(AUCVTFS, t)
			oprangeset(AUCVTFWD, t)
			oprangeset(AUCVTFWS, t)

		case ASYS:
			oprangeset(AAT, t)
			oprangeset(AIC, t)

		case ATLBI:
			oprangeset(ADC, t)

		case ASYSL, AHINT:
			break

		case ADMB:
			oprangeset(ADSB, t)
			oprangeset(AISB, t)

		case AMRS, AMSR:
			break

		case ALDAPR:
			oprangeset(ALDAPRW, t)
			oprangeset(ALDAPRB, t)
			oprangeset(ALDAPRH, t)

		case ALDAR:
			oprangeset(ALDARW, t)
			oprangeset(ALDARB, t)
			oprangeset(ALDARH, t)
			fallthrough

		case ALDXR:
			oprangeset(ALDXRB, t)
			oprangeset(ALDXRH, t)
			oprangeset(ALDXRW, t)

		case ALDAXR:
			oprangeset(ALDAXRB, t)
			oprangeset(ALDAXRH, t)
			oprangeset(ALDAXRW, t)

		case ALDXP:
			oprangeset(ALDXPW, t)
			oprangeset(ALDAXP, t)
			oprangeset(ALDAXPW, t)

		case ASTLR:
			oprangeset(ASTLRB, t)
			oprangeset(ASTLRH, t)
			oprangeset(ASTLRW, t)

		case ASTXR:
			oprangeset(ASTXRB, t)
			oprangeset(ASTXRH, t)
			oprangeset(ASTXRW, t)

		case ASTLXR:
			oprangeset(ASTLXRB, t)
			oprangeset(ASTLXRH, t)
			oprangeset(ASTLXRW, t)

		case ASTXP:
			oprangeset(ASTLXP, t)
			oprangeset(ASTLXPW, t)
			oprangeset(ASTXPW, t)

		case AVADDP:
			oprangeset(AVAND, t)
			oprangeset(AVCMEQ, t)
			oprangeset(AVORR, t)
			oprangeset(AVEOR, t)
			oprangeset(AVBSL, t)
			oprangeset(AVBIT, t)
			oprangeset(AVCMTST, t)
			oprangeset(AVUMAX, t)
			oprangeset(AVUMIN, t)
			oprangeset(AVUZP1, t)
			oprangeset(AVUZP2, t)
			oprangeset(AVBIF, t)

		case AVADD:
			oprangeset(AVSUB, t)
			oprangeset(AVRAX1, t)

		case AAESD:
			oprangeset(AAESE, t)
			oprangeset(AAESMC, t)
			oprangeset(AAESIMC, t)
			oprangeset(ASHA1SU1, t)
			oprangeset(ASHA256SU0, t)
			oprangeset(ASHA512SU0, t)
			oprangeset(ASHA1H, t)

		case ASHA1C:
			oprangeset(ASHA1P, t)
			oprangeset(ASHA1M, t)
			oprangeset(ASHA256H, t)
			oprangeset(ASHA256H2, t)
			oprangeset(ASHA512H, t)
			oprangeset(ASHA512H2, t)

		case ASHA1SU0:
			oprangeset(ASHA256SU1, t)
			oprangeset(ASHA512SU1, t)

		case AVADDV:
			oprangeset(AVUADDLV, t)

		case AVFMLA:
			oprangeset(AVFMLS, t)

		case AVPMULL:
			oprangeset(AVPMULL2, t)

		case AVUSHR:
			oprangeset(AVSHL, t)
			oprangeset(AVSRI, t)
			oprangeset(AVSLI, t)
			oprangeset(AVUSRA, t)

		case AVREV32:
			oprangeset(AVCNT, t)
			oprangeset(AVRBIT, t)
			oprangeset(AVREV64, t)
			oprangeset(AVREV16, t)

		case AVZIP1:
			oprangeset(AVZIP2, t)
			oprangeset(AVTRN1, t)
			oprangeset(AVTRN2, t)

		case AVUXTL:
			oprangeset(AVUXTL2, t)

		case AVUSHLL:
			oprangeset(AVUSHLL2, t)

		case AVLD1R:
			oprangeset(AVLD2, t)
			oprangeset(AVLD2R, t)
			oprangeset(AVLD3, t)
			oprangeset(AVLD3R, t)
			oprangeset(AVLD4, t)
			oprangeset(AVLD4R, t)

		case AVEOR3:
			oprangeset(AVBCAX, t)

		case AVUADDW:
			oprangeset(AVUADDW2, t)

		case AVTBL:
			oprangeset(AVTBX, t)

		case AVCNT,
			AVMOV,
			AVLD1,
			AVST1,
			AVST2,
			AVST3,
			AVST4,
			AVDUP,
			AVMOVI,
			APRFM,
			ARPRFM,
			AVEXT,
			AVXAR:
			break

		case AZADD:
			oprangeset(AZSUB, t)
			oprangeset(AZSQADD, t)
			oprangeset(AZUQADD, t)
			oprangeset(AZSQSUB, t)
			oprangeset(AZUQSUB, t)

		case AZSUBR:
			oprangeset(AZSMAX, t)
			oprangeset(AZUMAX, t)
			oprangeset(AZSMIN, t)
			oprangeset(AZUMIN, t)

		case AZAND:
			oprangeset(AZBIC, t)
			oprangeset(AZORR, t)
			oprangeset(AZEOR, t)

		case APAND:
			oprangeset(APBIC, t)
			oprangeset(APEOR, t)
			oprangeset(APANDS, t)
			oprangeset(APBICS, t)
			oprangeset(APEORS, t)
			oprangeset(APORR, t)
			oprangeset(APORN, t)
			oprangeset(APNOR, t)
			oprangeset(APNAND, t)
			oprangeset(APORRS, t)
			oprangeset(APORNS, t)
			oprangeset(APNORS, t)
			oprangeset(APNANDS, t)
			oprangeset(APBRKPA, t)
			oprangeset(APBRKPAS, t)
			oprangeset(APBRKPB, t)
			oprangeset(APBRKPBS, t)

		case APSEL:
			break

		case AZORN:
			oprangeset(AZEON, t)

		case AZDUP:
			break

		case AWHILELT:
			oprangeset(AWHILELE, t)
			oprangeset(AWHILEGT, t)
			oprangeset(AWHILEGE, t)
			oprangeset(AWHILELO, t)
			oprangeset(AWHILELS, t)
			oprangeset(AWHILEHI, t)
			oprangeset(AWHILEHS, t)

		case AWHILELTW:
			oprangeset(AWHILELEW, t)
			oprangeset(AWHILEGTW, t)
			oprangeset(AWHILEGEW, t)
			oprangeset(AWHILELOW, t)
			oprangeset(AWHILELSW, t)
			oprangeset(AWHILEHIW, t)
			oprangeset(AWHILEHSW, t)

		case AZCMPEQ:
			oprangeset(AZCMPGE, t)
			oprangeset(AZCMPGT, t)
			oprangeset(AZCMPHI, t)
			oprangeset(AZCMPHS, t)
			oprangeset(AZCMPLE, t)
			oprangeset(AZCMPLO, t)
			oprangeset(AZCMPLS, t)
			oprangeset(AZCMPLT, t)
			oprangeset(AZCMPNE, t)

		case AZSUQADD:
			oprangeset(AZUSQADD, t)
			oprangeset(AZSQSUBR, t)
			oprangeset(AZUQSUBR, t)
			oprangeset(AZSABD, t)
			oprangeset(AZUABD, t)
			oprangeset(AZSDIV, t)
			oprangeset(AZUDIV, t)
			oprangeset(AZSDIVR, t)
			oprangeset(AZUDIVR, t)

		case AZASR:
			oprangeset(AZLSR, t)
			oprangeset(AZLSL, t)

		case AZSQSHL:
			oprangeset(AZUQSHL, t)

		case AZASRD:
			oprangeset(AZSRSHR, t)
			oprangeset(AZURSHR, t)
			oprangeset(AZSQSHLU, t)

		case AZASRR:
			oprangeset(AZLSLR, t)
			oprangeset(AZLSRR, t)

		case AZSEL:
			break

		case AZSXTB:
			oprangeset(AZUXTB, t)
			oprangeset(AZSXTH, t)
			oprangeset(AZUXTH, t)
			oprangeset(AZSXTW, t)
			oprangeset(AZUXTW, t)
			oprangeset(AZABS, t)
			oprangeset(AZNEG, t)
			oprangeset(AZCLS, t)
			oprangeset(AZCLZ, t)
			oprangeset(AZCNT, t)
			oprangeset(AZCNOT, t)
			oprangeset(AZFABS, t)
			oprangeset(AZFNEG, t)
			oprangeset(AZNOT, t)

		case APTEST,
			APFIRST,
			APFALSE,
			APRDFFR,
			APRDFFRS,
			APNEXT,
			APTRUE,
			APTRUES:
			break

		case AZINDEX:
			break

		case AZMUL:
			break

		case AZPMUL:
			break

		case AZSQDMULH:
			oprangeset(AZSQRDMULH, t)

		case AZSMULH:
			oprangeset(AZUMULH, t)

		case APUNPKHI:
			oprangeset(APUNPKLO, t)

		case APZIP1:
			oprangeset(APZIP2, t)
			oprangeset(APUZP1, t)
			oprangeset(APUZP2, t)
			oprangeset(APTRN1, t)
			oprangeset(APTRN2, t)

		case AZZIP1:
			oprangeset(AZZIP2, t)
			oprangeset(AZUZP1, t)
			oprangeset(AZUZP2, t)
			oprangeset(AZTRN1, t)
			oprangeset(AZTRN2, t)

		case APBRKN:
			oprangeset(APBRKNS, t)

		case APBRKA:
			oprangeset(APBRKB, t)

		case APBRKAS:
			oprangeset(APBRKBS, t)

		case AZLD1B:
			oprangeset(AZLD1H, t)
			oprangeset(AZLD1W, t)
			oprangeset(AZLD1D, t)

		case AZLD1SB:
			oprangeset(AZLD1SH, t)
			oprangeset(AZLD1SW, t)

		case AZLDNF1B:
			oprangeset(AZLDNF1D, t)
			oprangeset(AZLDNF1H, t)
			oprangeset(AZLDNF1W, t)
			oprangeset(AZLDNF1SB, t)
			oprangeset(AZLDNF1SH, t)
			oprangeset(AZLDNF1SW, t)

		case AZLDFF1B:
			oprangeset(AZLDFF1D, t)
			oprangeset(AZLDFF1H, t)
			oprangeset(AZLDFF1W, t)
			oprangeset(AZLDFF1SB, t)
			oprangeset(AZLDFF1SH, t)
			oprangeset(AZLDFF1SW, t)

		case AZLD2B:
			oprangeset(AZLD2D, t)
			oprangeset(AZLD2H, t)
			oprangeset(AZLD2Q, t)
			oprangeset(AZLD2W, t)
			oprangeset(AZLD3B, t)
			oprangeset(AZLD3D, t)
			oprangeset(AZLD3H, t)
			oprangeset(AZLD3Q, t)
			oprangeset(AZLD3W, t)
			oprangeset(AZLD4B, t)
			oprangeset(AZLD4D, t)
			oprangeset(AZLD4H, t)
			oprangeset(AZLD4Q, t)
			oprangeset(AZLD4W, t)

		case AZAESE:
			oprangeset(AZAESD, t)

		case AZAESMC:
			oprangeset(AZAESIMC, t)

		case AZST1B:
			oprangeset(AZST1H, t)
			oprangeset(AZST1W, t)
			oprangeset(AZST1D, t)

		case AZST2B:
			oprangeset(AZST2D, t)
			oprangeset(AZST2H, t)
			oprangeset(AZST2Q, t)
			oprangeset(AZST2W, t)
			oprangeset(AZST3B, t)
			oprangeset(AZST3D, t)
			oprangeset(AZST3H, t)
			oprangeset(AZST3Q, t)
			oprangeset(AZST3W, t)
			oprangeset(AZST4B, t)
			oprangeset(AZST4D, t)
			oprangeset(AZST4H, t)
			oprangeset(AZST4Q, t)
			oprangeset(AZST4W, t)

		case AZINCB:
			oprangeset(AZDECB, t)
			oprangeset(AZCNTB, t)
			oprangeset(AZCNTH, t)
			oprangeset(AZCNTW, t)
			oprangeset(AZCNTD, t)

		case AZINCH:
			oprangeset(AZINCW, t)
			oprangeset(AZINCD, t)
			oprangeset(AZDECH, t)
			oprangeset(AZDECW, t)
			oprangeset(AZDECD, t)

		case AZLASTA:
			oprangeset(AZLASTB, t)

		case AZCNTP:
			break

		case AZSADDV:
			oprangeset(AZUADDV, t)

		case obj.ANOP,
			obj.AUNDEF,
			obj.AFUNCDATA,
			obj.APCALIGN,
			obj.APCALIGNMAX,
			obj.APCDATA,
			obj.ADUFFZERO,
			obj.ADUFFCOPY:
			break
		}
	}
}

// chipfloat7() checks if the immediate constants available in  FMOVS/FMOVD instructions.
// For details of the range of constants available, see
// http://infocenter.arm.com/help/topic/com.arm.doc.dui0473m/dom1359731199385.html.
func (c *ctxt7) chipfloat7(e float64) int {
	ei := math.Float64bits(e)
	l := uint32(int32(ei))
	h := uint32(int32(ei >> 32))

	if l != 0 || h&0xffff != 0 {
		return -1
	}
	h1 := h & 0x7fc00000
	if h1 != 0x40000000 && h1 != 0x3fc00000 {
		return -1
	}
	n := 0

	// sign bit (a)
	if h&0x80000000 != 0 {
		n |= 1 << 7
	}

	// exp sign bit (b)
	if h1 == 0x3fc00000 {
		n |= 1 << 6
	}

	// rest of exp and mantissa (cd-efgh)
	n |= int((h >> 16) & 0x3f)

	//print("match %.8lux %.8lux %d\n", l, h, n);
	return n
}

/* form offset parameter to SYS; special register number */
func SYSARG5(op0 int, op1 int, Cn int, Cm int, op2 int) int {
	return op0<<19 | op1<<16 | Cn<<12 | Cm<<8 | op2<<5
}

func SYSARG4(op1 int, Cn int, Cm int, op2 int) int {
	return SYSARG5(0, op1, Cn, Cm, op2)
}

// checkUnpredictable checks if the source and transfer registers are the same register.
// ARM64 manual says it is "constrained unpredictable" if the src and dst registers of STP/LDP are same.
func (c *ctxt7) checkUnpredictable(p *obj.Prog, isload bool, wback bool, rn int16, rt1 int16, rt2 int16) {
	if wback && rn != REGSP && (rn == rt1 || rn == rt2) {
		c.ctxt.Diag("constrained unpredictable behavior: %v", p)
	}
	if isload && rt1 == rt2 {
		c.ctxt.Diag("constrained unpredictable behavior: %v", p)
	}
}

/* checkindex checks if index >= 0 && index <= maxindex */
func (c *ctxt7) checkindex(p *obj.Prog, index, maxindex int) {
	if index < 0 || index > maxindex {
		c.ctxt.Diag("register element index out of range 0 to %d: %v", maxindex, p)
	}
}

/* checkoffset checks whether the immediate offset is valid for VLD[1-4].P and VST[1-4].P */
func (c *ctxt7) checkoffset(p *obj.Prog, as obj.As) {
	var offset, list, n, expect int64
	switch as {
	case AVLD1, AVLD2, AVLD3, AVLD4, AVLD1R, AVLD2R, AVLD3R, AVLD4R:
		offset = p.From.Offset
		list = p.To.Offset
	case AVST1, AVST2, AVST3, AVST4:
		offset = p.To.Offset
		list = p.From.Offset
	default:
		c.ctxt.Diag("invalid operation on op %v", p.As)
	}
	opcode := (list >> 12) & 15
	q := (list >> 30) & 1
	size := (list >> 10) & 3
	if offset == 0 {
		return
	}
	switch opcode {
	case 0x7:
		n = 1 // one register
	case 0xa:
		n = 2 // two registers
	case 0x6:
		n = 3 // three registers
	case 0x2:
		n = 4 // four registers
	default:
		c.ctxt.Diag("invalid register numbers in ARM64 register list: %v", p)
	}

	switch as {
	case AVLD1R, AVLD2R, AVLD3R, AVLD4R:
		if offset != n*(1<<uint(size)) {
			c.ctxt.Diag("invalid post-increment offset: %v", p)
		}
	default:
		if !(q == 0 && offset == n*8) && !(q == 1 && offset == n*16) {
			c.ctxt.Diag("invalid post-increment offset: %v", p)
		}
	}

	switch as {
	case AVLD1, AVST1:
		return
	case AVLD1R:
		expect = 1
	case AVLD2, AVST2, AVLD2R:
		expect = 2
	case AVLD3, AVST3, AVLD3R:
		expect = 3
	case AVLD4, AVST4, AVLD4R:
		expect = 4
	}

	if expect != n {
		c.ctxt.Diag("expected %d registers, got %d: %v.", expect, n, p)
	}
}

/* checkShiftAmount checks whether the index shift amount is valid */
/* for load with register offset instructions */
func (c *ctxt7) checkShiftAmount(p *obj.Prog, a *obj.Addr) {
	var amount int16
	amount = (a.Index >> 5) & 7
	switch p.As {
	case AMOVB, AMOVBU:
		if amount != 0 {
			c.ctxt.Diag("invalid index shift amount: %v", p)
		}
	case AMOVH, AMOVHU:
		if amount != 1 && amount != 0 {
			c.ctxt.Diag("invalid index shift amount: %v", p)
		}
	case AMOVW, AMOVWU, AFMOVS:
		if amount != 2 && amount != 0 {
			c.ctxt.Diag("invalid index shift amount: %v", p)
		}
	case AMOVD, AFMOVD:
		if amount != 3 && amount != 0 {
			c.ctxt.Diag("invalid index shift amount: %v", p)
		}
	default:
		panic("invalid operation")
	}
}

// checkGPreg checks whether the governing scalable predicate register Pg is valid when
// the destination operand is the scalable vector register.
// According to ARM64 SVE reference manual, if destination operand is the scalable vector
// register(Zn), Pg is encoded in 3 bits (P0-P7); if destination operand is the scalable
// predicate register(Pn), Pg is encoded in 4 bits (P0-P15).
func (c *ctxt7) checkGPreg(p *obj.Prog, preg int16) {
	if preg&15 > 7 {
		c.ctxt.Diag("invalid governing scalable predicate register P0-P7: %v", p)
	}
}

func (c *ctxt7) parseArng(reg int16) uint32 {
	return uint32(reg >> 5 & 15)
}

func (c *ctxt7) sveDataSize(arng uint32) uint32 {
	size := 0
	switch arng {
	case ARNG_B:
		size = 0
	case ARNG_H:
		size = 1
	case ARNG_S:
		size = 2
	case ARNG_D:
		size = 3
	default:
		c.ctxt.Diag("invalid data type")
	}
	return uint32(size)
}

func (c *ctxt7) whileccFlags(p *obj.Prog) (u, lt, eq, sf uint32) {
	as := p.As
	switch as {
	case AWHILELT, AWHILELE, AWHILEGT, AWHILEGE, AWHILELTW, AWHILELEW, AWHILEGTW, AWHILEGEW:
		u = 0
	case AWHILELO, AWHILELS, AWHILEHI, AWHILEHS, AWHILELOW, AWHILELSW, AWHILEHIW, AWHILEHSW:
		u = 1
	default:
		c.ctxt.Diag("unexpected asm: %v", p)
	}
	switch as {
	case AWHILEGT, AWHILEGE, AWHILEHI, AWHILEHS, AWHILEGTW, AWHILEGEW, AWHILEHIW, AWHILEHSW:
		lt = 0
	default:
		lt = 1
	}
	switch as {
	case AWHILELT, AWHILEGE, AWHILELO, AWHILEHS, AWHILELTW, AWHILEGEW, AWHILELOW, AWHILEHSW:
		eq = 0
	default:
		eq = 1
	}
	switch as {
	case AWHILELTW, AWHILELEW, AWHILEGTW, AWHILEGEW, AWHILELOW, AWHILELSW, AWHILEHIW, AWHILEHSW:
		sf = 0
	default:
		sf = 1
	}
	return
}

func (c *ctxt7) cmpccFlags(p *obj.Prog) (u, eq, lt, ne uint32) {
	as := p.As
	switch as {
	case AZCMPEQ, AZCMPNE, AZCMPLT, AZCMPLE, AZCMPGT, AZCMPGE:
		u = 0
	case AZCMPLO, AZCMPLS, AZCMPHI, AZCMPHS:
		u = 1
	default:
		c.ctxt.Diag("unexpected asm: %v", p)
	}
	switch as {
	case AZCMPEQ, AZCMPNE:
		eq = 1
	default:
		eq = 0
	}
	switch as {
	case AZCMPLT, AZCMPLE, AZCMPLO, AZCMPLS:
		lt = 1
	default:
		lt = 0
	}
	switch as {
	case AZCMPNE, AZCMPGT, AZCMPLE, AZCMPHI, AZCMPLS:
		ne = 1
	default:
		ne = 0
	}
	return
}

func (c *ctxt7) asmoutsve(p *obj.Prog, out []uint32) (count int) {
	o := c.oplook(p)

	o1 := uint32(0)
	switch o.type_ {
	default:
		c.ctxt.Diag("%v: unknown asm %d", p, o.type_)

	case 0:
		break

	case 1:
		/* zadd/zsub/zsqadd/zsqsub/zuqadd/zuqsub/zmul/zsmulh/zumulh/zpmul/zsqdmulh/zsqrdmulh
		 *     Zm.<T>, Zn.<T>, Zd.<T> */
		switch p.As {
		case AZADD:
			o1 |= 1<<26 | 1<<21 | 0<<13 | 0<<10
		case AZSUB:
			o1 |= 1<<26 | 1<<21 | 0<<13 | 1<<10
		case AZSQADD:
			o1 |= 1<<26 | 1<<21 | 0<<13 | 4<<10
		case AZUQADD:
			o1 |= 1<<26 | 1<<21 | 0<<13 | 5<<10
		case AZSQSUB:
			o1 |= 1<<26 | 1<<21 | 0<<13 | 6<<10
		case AZUQSUB:
			o1 |= 1<<26 | 1<<21 | 0<<13 | 7<<10
		case AZMUL:
			o1 |= 1<<26 | 1<<21 | 3<<13 | 0<<10
		case AZPMUL:
			o1 |= 1<<26 | 1<<21 | 3<<13 | 1<<10
		case AZSMULH:
			o1 |= 1<<26 | 1<<21 | 3<<13 | 2<<10
		case AZUMULH:
			o1 |= 1<<26 | 1<<21 | 3<<13 | 3<<10
		case AZSQDMULH:
			o1 |= 1<<26 | 1<<21 | 3<<13 | 4<<10
		case AZSQRDMULH:
			o1 |= 1<<26 | 1<<21 | 3<<13 | 5<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zm := p.From.Reg
		zn := p.Reg
		zd := p.To.Reg

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if am != an || am != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		if p.As == AZPMUL && am != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		sz := c.sveDataSize(am)

		o1 |= uint32(sz&3)<<22 | uint32(zm&31)<<16 | uint32(zn&31)<<5 | uint32(zd&31)

	case 2:
		/* zadd/zsub/zsubr/zsmax/zumax/zsmin/zumin/zsabd/zuabd/zmul/zsmulh/zumulh/zsdiv/zudiv/zsdivr/zudivr/
		 * zand/zorr/zbic/zeor/zsqadd/zuqadd/zsqsub/zuqsub/zsuqadd/zusqadd/zsqsubr/zuqsubr
		 *     Zm.<T>, Zdn.<T>, Pg/M, Zdn.<T> */
		switch p.As {
		case AZADD:
			o1 |= 1<<26 | 0<<19 | 0<<16
		case AZSUB:
			o1 |= 1<<26 | 0<<19 | 1<<16
		case AZSUBR:
			o1 |= 1<<26 | 0<<19 | 3<<16
		case AZSMAX:
			o1 |= 1<<26 | 1<<19 | 0<<16
		case AZUMAX:
			o1 |= 1<<26 | 1<<19 | 1<<16
		case AZSMIN:
			o1 |= 1<<26 | 1<<19 | 2<<16
		case AZUMIN:
			o1 |= 1<<26 | 1<<19 | 3<<16
		case AZSABD:
			o1 |= 1<<26 | 1<<19 | 4<<16
		case AZUABD:
			o1 |= 1<<26 | 1<<19 | 5<<16
		case AZMUL:
			o1 |= 1<<26 | 2<<19 | 0<<16
		case AZSMULH:
			o1 |= 1<<26 | 2<<19 | 2<<16
		case AZUMULH:
			o1 |= 1<<26 | 2<<19 | 3<<16
		case AZSDIV:
			o1 |= 1<<26 | 2<<19 | 4<<16
		case AZUDIV:
			o1 |= 1<<26 | 2<<19 | 5<<16
		case AZSDIVR:
			o1 |= 1<<26 | 2<<19 | 6<<16
		case AZUDIVR:
			o1 |= 1<<26 | 2<<19 | 7<<16
		case AZORR:
			o1 |= 1<<26 | 3<<19 | 0<<16
		case AZEOR:
			o1 |= 1<<26 | 3<<19 | 1<<16
		case AZAND:
			o1 |= 1<<26 | 3<<19 | 2<<16
		case AZBIC:
			o1 |= 1<<26 | 3<<19 | 3<<16
		case AZSQADD:
			o1 |= 0x11<<26 | 3<<19 | 0<<16 | 1<<15
		case AZUQADD:
			o1 |= 0x11<<26 | 3<<19 | 1<<16 | 1<<15
		case AZSQSUB:
			o1 |= 0x11<<26 | 3<<19 | 2<<16 | 1<<15
		case AZUQSUB:
			o1 |= 0x11<<26 | 3<<19 | 3<<16 | 1<<15
		case AZSUQADD:
			o1 |= 0x11<<26 | 3<<19 | 4<<16 | 1<<15
		case AZUSQADD:
			o1 |= 0x11<<26 | 3<<19 | 5<<16 | 1<<15
		case AZSQSUBR:
			o1 |= 0x11<<26 | 3<<19 | 6<<16 | 1<<15
		case AZUQSUBR:
			o1 |= 0x11<<26 | 3<<19 | 7<<16 | 1<<15
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zm := p.From.Reg
		zn := p.Reg
		pg := p.GetFrom3().Reg
		zd := p.To.Reg
		if zn != zd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		zdn := zd

		am := c.parseArng(zm)
		ad := c.parseArng(zd)
		if am != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		c.checkGPreg(p, pg)

		sz := c.sveDataSize(am)

		if o1>>30&1 == 0 && o1>>18&7 == 5 /* divide */ && sz < 2 {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(sz&3)<<22 | uint32(pg&7)<<10 | uint32(zm&31)<<5 | uint32(zdn&31)

	case 3:
		/* zadd/zsub/zsubr/zsqadd/zuqadd/zsqsub/zuqsub $(imm8|imm16), Zdn.<T>, Zdn<T>
		 * zsmax/zumax/zsmin/zumin/zmul $imm8, Zdn.<T>, Zdn.<T>
		 * zdup $(imm8|imm16), Zd.<T> */
		switch p.As {
		case AZADD:
			o1 |= 0x25<<24 | 1<<21 | 0<<19 | 0<<16 | 3<<14
		case AZSUB:
			o1 |= 0x25<<24 | 1<<21 | 0<<19 | 1<<16 | 3<<14
		case AZSUBR:
			o1 |= 0x25<<24 | 1<<21 | 0<<19 | 3<<16 | 3<<14
		case AZSQADD:
			o1 |= 0x25<<24 | 1<<21 | 0<<19 | 4<<16 | 3<<14
		case AZUQADD:
			o1 |= 0x25<<24 | 1<<21 | 0<<19 | 5<<16 | 3<<14
		case AZSQSUB:
			o1 |= 0x25<<24 | 1<<21 | 0<<19 | 6<<16 | 3<<14
		case AZUQSUB:
			o1 |= 0x25<<24 | 1<<21 | 0<<19 | 7<<16 | 3<<14
		case AZSMAX:
			o1 |= 0x25<<24 | 1<<21 | 1<<19 | 0<<16 | 3<<14
		case AZUMAX:
			o1 |= 0x25<<24 | 1<<21 | 1<<19 | 1<<16 | 3<<14
		case AZSMIN:
			o1 |= 0x25<<24 | 1<<21 | 1<<19 | 2<<16 | 3<<14
		case AZUMIN:
			o1 |= 0x25<<24 | 1<<21 | 1<<19 | 3<<16 | 3<<14
		case AZMUL:
			o1 |= 0x25<<24 | 1<<21 | 2<<19 | 0<<16 | 3<<14
		case AZDUP:
			o1 |= 0x25<<24 | 1<<21 | 3<<19 | 0<<16 | 3<<14
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		imm8 := p.From.Offset
		zn := p.Reg
		zd := p.To.Reg

		ad := c.parseArng(zd)
		sz := c.sveDataSize(ad)

		sh := 0
		opg := o1 >> 19 & 3
		switch opg {
		case 0: /* add/substract */
			if imm8&0xff == 0 {
				imm8 >>= 8
				sh = 1
			}
			if imm8&^0xff != 0 {
				c.ctxt.Diag("immediate out of range [0, 255]{ << 8 }: %v", p)
			}
			if sz == 0 && sh == 1 {
				c.ctxt.Diag("undefined instruction cause [size:sh == '001']: %v", p)
			}
		case 1: /* min/max */
			u := o1 >> 16 & 1
			if u == 0 { /* signed */
				if imm8 < -128 || imm8 > 127 {
					c.ctxt.Diag("immediate out of range [-128, 127]: %v", p)
				}
			} else { /* unsigned */
				if imm8&^0xff != 0 {
					c.ctxt.Diag("immediate out of range [0, 255]: %v", p)
				}
			}
		case 2: /* multiply */
			if imm8 < -128 || imm8 > 127 {
				c.ctxt.Diag("immediate out of range [-128, 127]: %v", p)
			}
		case 3: /* duplicate */
			if imm8&0xff == 0 {
				imm8 >>= 8
				sh = 1
			}
			if imm8&^0xff != 0 {
				c.ctxt.Diag("immediate out of range [0, 255]{ << 8 }: %v", p)
			}
			if sz == 0 && sh == 1 {
				c.ctxt.Diag("undefined instruction cause [size:sh == '001']: %v", p)
			}
			zn = zd
		default:
			break
		}
		if zn != zd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		zdn := zd

		o1 |= uint32(sz&3)<<22 | uint32(sh&1)<<13 | uint32(imm8&0xff)<<5 | uint32(zdn&31)

	case 4: /* zand/zorr/zbic/zeor Zm.D, Zn.D, Zd.D */
		switch p.As {
		case AZAND:
			o1 |= 1<<26 | 0<<22 | 1<<21 | 3<<12
		case AZORR:
			o1 |= 1<<26 | 1<<22 | 1<<21 | 3<<12
		case AZEOR:
			o1 |= 1<<26 | 2<<22 | 1<<21 | 3<<12
		case AZBIC:
			o1 |= 1<<26 | 3<<22 | 1<<21 | 3<<12
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zm := p.From.Reg
		zn := p.Reg
		zd := p.To.Reg

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if am != ARNG_D || an != ARNG_D || ad != ARNG_D {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(zm&31)<<16 | uint32(zn&31)<<5 | uint32(zd&31)

	case 5: /* zand/zorr/zbic/zorn/zeor/zeon $bitcon, Zdn.<T>, Zdn.<T> */
		switch p.As {
		case AZORR, AZORN:
			o1 |= 5<<24 | 0<<22
		case AZEOR, AZEON:
			o1 |= 5<<24 | 1<<22
		case AZAND, AZBIC:
			o1 |= 5<<24 | 2<<22
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		con := p.From.Offset
		zn := p.Reg
		zd := p.To.Reg
		if zn != zd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		zdn := zd
		switch p.As {
		case AZBIC, AZORN, AZEON:
			con = ^con /* con = -con - 1 */
		default:
			break
		}

		imm13 := bitconEncode(uint64(con), 64) >> 10
		n := imm13 >> 12 & 1
		s := imm13 & 0x3f
		var a uint32
		switch {
		case n == 1:
			a = ARNG_D
		case s&32 != 32:
			a = ARNG_S
		case s&16 != 16:
			a = ARNG_H
		case s&14 != 14:
			a = ARNG_B
		default:
			c.ctxt.Diag("invalid immediate: %v", p)
		}
		an := c.parseArng(zn)
		if a != an {
			c.ctxt.Diag("data size mismatch with immediate: %v", p)
		}

		o1 |= uint32(imm13&0x1fff)<<5 | uint32(zdn&31)

	case 6:
		/* pand/pbic/peor/pands/pbics/peors/porr/porn/pnor/pnand/porrs/porns/pnors/pnands/
		 * pbrkpa/pbrkpas/pbrkpb/pbrkpbs Pm.B, Pn.B, Pg/Z, Pd.B
		 * psel Pm.B, Pn.B, Pg, Pd.B */
		switch p.As {
		case APAND:
			o1 |= 0x25<<24 | 0<<23 | 0<<22 | 1<<14 | 0<<9 | 0<<4
		case APBIC:
			o1 |= 0x25<<24 | 0<<23 | 0<<22 | 1<<14 | 0<<9 | 1<<4
		case APEOR:
			o1 |= 0x25<<24 | 0<<23 | 0<<22 | 1<<14 | 1<<9 | 0<<4
		case APSEL:
			o1 |= 0x25<<24 | 0<<23 | 0<<22 | 1<<14 | 1<<9 | 1<<4
		case APANDS:
			o1 |= 0x25<<24 | 0<<23 | 1<<22 | 1<<14 | 0<<9 | 0<<4
		case APBICS:
			o1 |= 0x25<<24 | 0<<23 | 1<<22 | 1<<14 | 0<<9 | 1<<4
		case APEORS:
			o1 |= 0x25<<24 | 0<<23 | 1<<22 | 1<<14 | 1<<9 | 0<<4
		case APORR:
			o1 |= 0x25<<24 | 1<<23 | 0<<22 | 1<<14 | 0<<9 | 0<<4
		case APORN:
			o1 |= 0x25<<24 | 1<<23 | 0<<22 | 1<<14 | 0<<9 | 1<<4
		case APNOR:
			o1 |= 0x25<<24 | 1<<23 | 0<<22 | 1<<14 | 1<<9 | 0<<4
		case APNAND:
			o1 |= 0x25<<24 | 1<<23 | 0<<22 | 1<<14 | 1<<9 | 1<<4
		case APORRS:
			o1 |= 0x25<<24 | 1<<23 | 1<<22 | 1<<14 | 0<<9 | 0<<4
		case APORNS:
			o1 |= 0x25<<24 | 1<<23 | 1<<22 | 1<<14 | 0<<9 | 1<<4
		case APNORS:
			o1 |= 0x25<<24 | 1<<23 | 1<<22 | 1<<14 | 1<<9 | 0<<4
		case APNANDS:
			o1 |= 0x25<<24 | 1<<23 | 1<<22 | 1<<14 | 1<<9 | 1<<4
		case APBRKPA:
			o1 |= 0x25<<24 | 0<<22 | 3<<14 | 0<<4
		case APBRKPB:
			o1 |= 0x25<<24 | 0<<22 | 3<<14 | 1<<4
		case APBRKPAS:
			o1 |= 0x25<<24 | 1<<22 | 3<<14 | 0<<4
		case APBRKPBS:
			o1 |= 0x25<<24 | 1<<22 | 3<<14 | 1<<4
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		pm := p.From.Reg
		pn := p.Reg
		pg := p.GetFrom3().Reg
		pd := p.To.Reg

		am := c.parseArng(pm)
		an := c.parseArng(pn)
		ad := c.parseArng(pd)
		if am != ARNG_B || an != ARNG_B || ad != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(pm&15)<<16 | uint32(pg&15)<<10 | uint32(pn&15)<<5 | uint32(pd&15)

	case 7: /* zdup R(n|SP), Zd.<T> */
		o1 |= 5<<24 | 1<<21 | 7<<11

		rn := p.From.Reg
		zd := p.To.Reg

		ad := c.parseArng(zd)
		sz := c.sveDataSize(ad)

		o1 |= uint32(sz&3)<<22 | uint32(rn&31)<<5 | uint32(zd&31)

	case 8: /* zdup Zn.<T>[idx], Zd.<T> */
		o1 |= 5<<24 | 1<<21 | 1<<13

		zn := p.From.Reg
		idx := p.From.Index
		zd := p.To.Reg

		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if an != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}

		var imm7 uint32
		switch an {
		case ARNG_B:
			if idx&^63 != 0 {
				c.ctxt.Diag("index out of range [0, 63]: %v", p)
			}
			imm7 = uint32(idx)<<1 | 1
		case ARNG_H:
			if idx&^31 != 0 {
				c.ctxt.Diag("index out of range [0, 31]: %v", p)
			}
			imm7 = uint32(idx)<<2 | 2
		case ARNG_S:
			if idx&^15 != 0 {
				c.ctxt.Diag("index out of range [0, 15]: %v", p)
			}
			imm7 = uint32(idx)<<3 | 4
		case ARNG_D:
			if idx&^7 != 0 {
				c.ctxt.Diag("index out of range [0, 7]: %v", p)
			}
			imm7 = uint32(idx)<<4 | 8
		case ARNG_Q:
			if idx&^3 != 0 {
				c.ctxt.Diag("index out of range [0, 3]: %v", p)
			}
			imm7 = uint32(idx)<<5 | 16
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		imm2 := imm7 >> 5 & 3
		tsz := imm7 & 31

		o1 |= uint32(imm2&3)<<22 | uint32(tsz&31)<<16 | uint32(zn&31)<<5 | uint32(zd&31)

	case 9: /* zmul Zm.<T>[idx], Zn.<T>, Zd.<T> */
		o1 |= 0x11<<26 | 1<<21 | 0x1f<<11

		zm := p.From.Reg
		idx := p.From.Index
		zn := p.Reg
		zd := p.To.Reg

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if am != an || am != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		switch am {
		case ARNG_B:
			c.ctxt.Diag("invalid arrangement: %v", p)
		case ARNG_H:
			if idx&^7 != 0 {
				c.ctxt.Diag("index out of range [0, 7]: %v", p)
			}
			if zm&31&^7 != 0 {
				c.ctxt.Diag("register Zm out of range [0, 7]: %v", p)
			}
			i3h := idx >> 2 & 1
			i3l := idx & 3
			o1 |= uint32(i3h&1)<<22 | uint32(i3l&3)<<19 | uint32(zm&7)<<16
		case ARNG_S:
			if idx&^3 != 0 {
				c.ctxt.Diag("index out of range [0, 3]: %v", p)
			}
			if zm&31&^7 != 0 {
				c.ctxt.Diag("register Zm out of range [0, 7]: %v", p)
			}
			i2 := idx & 3
			o1 |= 2<<22 | uint32(i2&3)<<19 | uint32(zm&7)<<16
		case ARNG_D:
			if idx&^1 != 0 {
				c.ctxt.Diag("index out of range [0, 1]: %v", p)
			}
			if zm&31&^15 != 0 {
				c.ctxt.Diag("register Zm out of range [0, 15]: %v", p)
			}
			i1 := idx & 1
			o1 |= 3<<22 | uint32(i1&1)<<20 | uint32(zm&15)<<16
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(zn&31)<<5 | uint32(zd&31)

	case 10: /* while<cc> Rm, Rn, Pd.<T> => whilexxw is the 32-bit version of whilexx */
		u, lt, eq, sf := c.whileccFlags(p)
		o1 |= 0x25<<24 | 1<<21 | sf<<12 | u<<11 | lt<<10 | eq<<4

		rm := p.From.Reg
		rn := p.Reg
		pd := p.To.Reg

		ad := c.parseArng(pd)
		sz := c.sveDataSize(ad)

		o1 |= uint32(sz&3)<<22 | uint32(rm&31)<<16 | uint32(rn&31)<<5 | uint32(pd&15)

	case 11: /* while<cc> Rm, Rn, {Pd1.<T>, Pd2.<T>} => whilexxw is unavailable */
		u, lt, eq, sf := c.whileccFlags(p)
		if sf == 0 {
			c.ctxt.Diag("unexpected asm: %v", p)
		}
		o1 |= 0x25<<24 | 1<<21 | 5<<12 | u<<11 | lt<<10 | 1<<4 | eq

		rm := p.From.Reg
		rn := p.Reg
		pd := p.To.Offset

		if pd&1 != 0 {
			c.ctxt.Diag("first register number must be even: %v", p)
		}
		regcnt := pd >> 9 & 7
		if regcnt != 2 {
			c.ctxt.Diag("register number should be 2: %v", p)
		}

		ad := c.parseArng(int16(pd))
		sz := c.sveDataSize(ad)

		o1 |= uint32(sz&3)<<22 | uint32(rm&31)<<16 | uint32(rn&31)<<5 | uint32(pd>>1&7)<<1

	case 12:
		/* zcmp<cc> Zm.<T>, Zn.<T>, Pg/Z, Pd.<T> (vectors)
		 * zcmp<cc> Zm.D, Zn.<T>, Pg/Z, Pd.<T> (wide elements) */
		u, eq, lt, ne := c.cmpccFlags(p)

		zm := p.From.Reg
		zn := p.Reg
		pg := p.GetFrom3().Reg
		pd := p.To.Reg

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		ad := c.parseArng(pd)
		c.checkGPreg(p, pg)
		if an != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		if am != an { /* wide elements */
			if am != ARNG_D || an == ARNG_D {
				c.ctxt.Diag("data size mismatch: %v", p)
			}
			o1 |= 0x24<<24 | u<<15 | (eq^1)<<14 | (lt|eq)<<13 | ne<<4
		} else { /* vectors */
			if lt == 1 {
				zm, zn = zn, zm
				ne ^= 1
			}
			o1 |= 0x24<<24 | (u^1)<<15 | eq<<13 | ne<<4
		}

		sz := c.sveDataSize(an)

		o1 |= uint32(sz&3)<<22 | uint32(zm&31)<<16 | uint32(pg&7)<<10 | uint32(zn&31)<<5 | uint32(pd&15)

	case 13: /* zcmp<cc> $imm, Zn.<T>, Pg/Z, Pd.<T> */
		u, eq, lt, ne := c.cmpccFlags(p)
		o1 |= 0x24<<24 | (u^1)<<24 | u<<21 | eq<<15 | lt<<13 | ne<<4

		imm := p.From.Offset
		zn := p.Reg
		pg := p.GetFrom3().Reg
		pd := p.To.Reg

		an := c.parseArng(zn)
		ad := c.parseArng(pd)
		if an != ad {
			c.ctxt.Diag("data type mismatch: %v", p)
		}
		c.checkGPreg(p, pg)

		sz := c.sveDataSize(an)

		if u == 1 {
			if imm < 0 || imm > 127 {
				c.ctxt.Diag("immediate out of range [0, 127]: %v", p)
			}
			o1 |= uint32(sz&3)<<22 | uint32(imm&0x7f)<<14 | uint32(pg&7)<<10 | uint32(zn&31)<<5 | uint32(pd&15)
		} else {
			if imm < -16 || imm > 15 {
				c.ctxt.Diag("immediate out of range [-16, 15]: %v", p)
			}
			o1 |= uint32(sz&3)<<22 | uint32(imm&0x1f)<<16 | uint32(pg&7)<<10 | uint32(zn&31)<<5 | uint32(pd&15)
		}

	case 14: /* zsel Zm.<T>, Zn.<T>, Pv, Zd.<T> */
		o1 |= 5<<24 | 1<<21 | 3<<14

		zm := p.From.Reg
		zn := p.Reg
		pv := p.GetFrom3().Reg
		zd := p.To.Reg

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if am != an || am != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}

		sz := c.sveDataSize(am)

		o1 |= uint32(sz&3)<<22 | uint32(zm&31)<<16 | uint32(pv&15)<<10 | uint32(zn&31)<<5 | uint32(zd&31)

	case 15: /* zasr/zlsr/zlsl/zsqshl/zuqshl/zasrd/zsrshr/zurshr/zsqshlu $const, Zdn.<T>, Pg/M, Zdn.<T> */
		switch p.As {
		case AZASR:
			o1 |= 1<<26 | 0<<16 | 1<<15
		case AZLSR:
			o1 |= 1<<26 | 1<<16 | 1<<15
		case AZLSL:
			o1 |= 1<<26 | 3<<16 | 1<<15
		case AZASRD:
			o1 |= 1<<26 | 4<<16 | 1<<15
		case AZSQSHL:
			o1 |= 1<<26 | 6<<16 | 1<<15
		case AZUQSHL:
			o1 |= 1<<26 | 7<<16 | 1<<15
		case AZSRSHR:
			o1 |= 1<<26 | 12<<16 | 1<<15
		case AZURSHR:
			o1 |= 1<<26 | 13<<16 | 1<<15
		case AZSQSHLU:
			o1 |= 1<<26 | 15<<16 | 1<<15
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		con := p.From.Offset
		zn := p.Reg
		pg := p.GetFrom3().Reg
		zd := p.To.Reg
		if zn != zd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		zdn := zd
		c.checkGPreg(p, pg)

		var r bool
		switch p.As {
		case AZASR, AZLSR, AZASRD, AZSRSHR, AZURSHR:
			r = true
		default:
			r = false
		}

		an := c.parseArng(zn)
		var imm7 uint32
		switch an {
		case ARNG_B:
			if r {
				if con <= 0 || con > 8 {
					c.ctxt.Diag("immediate shift amount out of range [1, 8]: %v", p)
				}
				imm7 = uint32(16 - con)
			} else {
				if con&^7 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 7]: %v", p)
				}
				imm7 = uint32(con) | 1<<3
			}
		case ARNG_H:
			if r {
				if con <= 0 || con > 16 {
					c.ctxt.Diag("immediate shift amount out of range [1, 16]: %v", p)
				}
				imm7 = uint32(32 - con)
			} else {
				if con&^15 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 15]: %v", p)
				}
				imm7 = uint32(con) | 1<<4
			}
		case ARNG_S:
			if r {
				if con <= 0 || con > 32 {
					c.ctxt.Diag("immediate shift amount out of range [1, 32]: %v", p)
				}
				imm7 = uint32(64 - con)
			} else {
				if con&^31 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 31]: %v", p)
				}
				imm7 = uint32(con) | 1<<5
			}
		case ARNG_D:
			if r {
				if con <= 0 || con > 64 {
					c.ctxt.Diag("immediate shift amount out of range [1, 64]: %v", p)
				}
				imm7 = uint32(128 - con)
			} else {
				if con&^63 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 63]: %v", p)
				}
				imm7 = uint32(con) | 1<<6
			}
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		tszh := imm7 >> 5 & 3
		tszl := imm7 >> 3 & 3
		imm3 := imm7 & 7

		o1 |= uint32(tszh&3)<<22 | uint32(pg&7)<<10 | uint32(tszl&3)<<8 | uint32(imm3&7)<<5 | uint32(zdn&31)

	case 16:
		/* zasr/zlsr/zlsl/zasrr/zlsrr/zlslr Zm.<T>, Zdn.<T>, Pg/M, Zdn.<T> (vectors)
		 * zasr/zlsr/zlsl Zm.D, Zdn.<T>, Pg/M, Zdn.<T> (wide elements) */
		switch p.As {
		case AZASR:
			o1 |= 1<<26 | 1<<20 | 0<<16 | 1<<15
		case AZLSR:
			o1 |= 1<<26 | 1<<20 | 1<<16 | 1<<15
		case AZLSL:
			o1 |= 1<<26 | 1<<20 | 3<<16 | 1<<15
		case AZASRR:
			o1 |= 1<<26 | 1<<20 | 4<<16 | 1<<15
		case AZLSRR:
			o1 |= 1<<26 | 1<<20 | 5<<16 | 1<<15
		case AZLSLR:
			o1 |= 1<<26 | 1<<20 | 7<<16 | 1<<15
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zm := p.From.Reg
		zn := p.Reg
		pg := p.GetFrom3().Reg
		zd := p.To.Reg
		if zn != zd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		zdn := zd
		c.checkGPreg(p, pg)

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		if am != an {
			switch p.As {
			case AZASR, AZLSR, AZLSL:
				if am != ARNG_D || an == ARNG_D {
					c.ctxt.Diag("data size mismatch: %v", p)
				}
				/* wide elements */
				o1 |= 1 << 19
			default:
				c.ctxt.Diag("data size mismatch: %v", p)
			}
		}

		sz := c.sveDataSize(an)

		o1 |= uint32(sz&3)<<22 | uint32(pg&7)<<10 | uint32(zm&31)<<5 | uint32(zdn&31)

	case 17:
		/* zsxtb/zuxtb/zsxth/zuxth/zsxtw/zustw/zabs/zneg/zcls/zclz/zcnt/zcnot/zfabs/zfneg/znot
		 *     Zn.<T>, Pg/M, Zd.<T> */
		switch p.As {
		case AZSXTB:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 0<<16 | 5<<13
		case AZUXTB:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 1<<16 | 5<<13
		case AZSXTH:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 2<<16 | 5<<13
		case AZUXTH:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 3<<16 | 5<<13
		case AZSXTW:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 4<<16 | 5<<13
		case AZUXTW:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 5<<16 | 5<<13
		case AZABS:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 6<<16 | 5<<13
		case AZNEG:
			o1 |= 1<<26 | 1<<20 | 0<<19 | 7<<16 | 5<<13
		case AZCLS:
			o1 |= 1<<26 | 1<<20 | 1<<19 | 0<<16 | 5<<13
		case AZCLZ:
			o1 |= 1<<26 | 1<<20 | 1<<19 | 1<<16 | 5<<13
		case AZCNT:
			o1 |= 1<<26 | 1<<20 | 1<<19 | 2<<16 | 5<<13
		case AZCNOT:
			o1 |= 1<<26 | 1<<20 | 1<<19 | 3<<16 | 5<<13
		case AZFABS:
			o1 |= 1<<26 | 1<<20 | 1<<19 | 4<<16 | 5<<13
		case AZFNEG:
			o1 |= 1<<26 | 1<<20 | 1<<19 | 5<<16 | 5<<13
		case AZNOT:
			o1 |= 1<<26 | 1<<20 | 1<<19 | 6<<16 | 5<<13
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zn := p.From.Reg
		pg := p.Reg
		zd := p.To.Reg

		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if an != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		c.checkGPreg(p, pg)

		sz := c.sveDataSize(an)

		switch p.As {
		case AZSXTB, AZUXTB, AZFABS, AZFNEG:
			if sz < 1 {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		case AZSXTH, AZUXTH:
			if sz < 2 {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		case AZSXTW, AZUXTW:
			if sz < 3 {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		default:
			break
		}

		o1 |= uint32(sz&3)<<22 | uint32(pg&7)<<10 | uint32(zn&31)<<5 | uint32(zd&31)

	case 18: /* zasr/zlsr/zlsl $const, Zn,<T>, Zd.<T> */
		var r bool
		switch p.As {
		case AZASR:
			o1 |= 1<<26 | 1<<21 | 1<<15 | 1<<12 | 0<<10
			r = true
		case AZLSR:
			o1 |= 1<<26 | 1<<21 | 1<<15 | 1<<12 | 1<<10
			r = true
		case AZLSL:
			o1 |= 1<<26 | 1<<21 | 1<<15 | 1<<12 | 3<<10
			r = false
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		con := p.From.Offset
		zn := p.Reg
		zd := p.To.Reg

		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if an != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		var imm7 uint32
		switch an {
		case ARNG_B:
			if r {
				if con <= 0 || con > 8 {
					c.ctxt.Diag("immediate shift amount out of range [1, 8]: %v", p)
				}
				imm7 = uint32(16 - con)
			} else {
				if con&^7 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 7]: %v", p)
				}
				imm7 = uint32(con) | 1<<3
			}
		case ARNG_H:
			if r {
				if con <= 0 || con > 16 {
					c.ctxt.Diag("immediate shift amount out of range [1, 16]: %v", p)
				}
				imm7 = uint32(32 - con)
			} else {
				if con&^15 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 15]: %v", p)
				}
				imm7 = uint32(con) | 1<<4
			}
		case ARNG_S:
			if r {
				if con <= 0 || con > 32 {
					c.ctxt.Diag("immediate shift amount out of range [1, 32]: %v", p)
				}
				imm7 = uint32(64 - con)
			} else {
				if con&^31 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 31]: %v", p)
				}
				imm7 = uint32(con) | 1<<5
			}
		case ARNG_D:
			if r {
				if con <= 0 || con > 64 {
					c.ctxt.Diag("immediate shift amount out of range [1, 64]: %v", p)
				}
				imm7 = uint32(128 - con)
			} else {
				if con&^63 != 0 {
					c.ctxt.Diag("immediate shift amount out of range [0, 63]: %v", p)
				}
				imm7 = uint32(con) | 1<<6
			}
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		tszh := imm7 >> 5 & 3
		tszl := imm7 >> 3 & 3
		imm3 := imm7 & 7

		o1 |= uint32(tszh&3)<<22 | uint32(tszl&3)<<19 | uint32(imm3&7)<<16 | uint32(zn&31)<<5 | uint32(zd&31)

	case 19: /* zasr/zlsr/zlsl Zm.D, Zn,<T>, Zd.<T> */
		switch p.As {
		case AZASR:
			o1 |= 1<<26 | 1<<21 | 1<<15 | 0<<10
		case AZLSR:
			o1 |= 1<<26 | 1<<21 | 1<<15 | 1<<10
		case AZLSL:
			o1 |= 1<<26 | 1<<21 | 1<<15 | 3<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zm := p.From.Reg
		zn := p.Reg
		zd := p.To.Reg

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if an != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		if am != ARNG_D || an == ARNG_D {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		sz := c.sveDataSize(an)

		o1 |= uint32(sz&3)<<22 | uint32(zm&31)<<16 | uint32(zn&31)<<5 | uint32(zd&31)

	case 20: /* ptest Pn.B, Pg */
		o1 |= 0x25<<24 | 1<<22 | 1<<20 | 3<<14

		pn := p.From.Reg
		pg := p.Reg

		an := c.parseArng(pn)
		if an != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(pg&15)<<10 | uint32(pn&15)<<5

	case 21: /* pfirst Pdn.B, Pg, Pdn.B */
		o1 |= 0x25<<24 | 1<<22 | 3<<19 | 3<<14

		pn := p.From.Reg
		pg := p.Reg
		pd := p.To.Reg
		if pn != pd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		pdn := pd

		an := c.parseArng(pn)
		if an != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(pg&15)<<5 | uint32(pdn&15)

	case 22: /* pfalse Pd.B */
		o1 |= 0x25<<24 | 3<<19 | 7<<13 | 1<<10

		pd := p.To.Reg

		ad := c.parseArng(pd)
		if ad != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(pd & 15)

	case 23:
		/* prdffr/prdffrs Pg/Z, Pd.B (predicated)
		 * prdffr Pd.B (unpredicated) */
		switch p.As {
		case APRDFFR:
			o1 |= 0x25<<24 | 0<<22 | 3<<19 | 15<<12
		case APRDFFRS:
			o1 |= 0x25<<24 | 1<<22 | 3<<19 | 15<<12
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		pd := p.To.Reg

		ad := c.parseArng(pd)
		if ad != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		if p.From.Type == obj.TYPE_NONE { /* unpredicated */
			o1 |= 1 << 16
			o1 |= uint32(pd & 15)
		} else { /* predicated */
			pg := p.From.Reg
			o1 |= uint32(pg&15)<<5 | uint32(pd&15)
		}

	case 24: /* pnext Pdn.B, Pv, Pdn.B */
		o1 |= 0x25<<24 | 1<<22 | 3<<19 | 7<<14 | 1<<10

		pn := p.From.Reg
		pv := p.Reg
		pd := p.To.Reg
		if pn != pd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		pdn := pd

		an := c.parseArng(pn)
		sz := c.sveDataSize(an)

		o1 |= uint32(sz&3)<<22 | uint32(pv&15)<<5 | uint32(pdn&15)

	case 25: /* ptrue/ptrues {pattern, }Pd.<T> */
		switch p.As {
		case APTRUE:
			o1 |= 0x25<<24 | 3<<19 | 0<<16 | 7<<13
		case APTRUES:
			o1 |= 0x25<<24 | 3<<19 | 1<<16 | 7<<13
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		pat := Pattern(p.From.Offset)
		if p.From.Type == obj.TYPE_NONE {
			pat = PAT_ALL
		}
		pd := p.To.Reg
		if pat&^31 != 0 {
			c.ctxt.Diag("illegal pattern code: %v", p)
		}

		ad := c.parseArng(pd)
		sz := c.sveDataSize(ad)

		o1 |= uint32(sz&3)<<22 | uint32(pat&31)<<5 | uint32(pd&31)

	case 26: /* zindex (Rn|$imm5), (Rn|$imm5), Zd.<T> */
		o1 |= 1<<26 | 1<<21 | 1<<14

		var op1, op2 int64
		if p.From.Type == obj.TYPE_CONST {
			op1 = int64(p.From.Offset)
			if op1 < -16 || op1 > 15 {
				c.ctxt.Diag("immediate out of range [-16, 15]: %v", p)
			}
		} else {
			o1 |= 1 << 11
			op1 = int64(p.From.Reg)
		}
		if p.GetFrom3().Type == obj.TYPE_CONST {
			op2 = int64(p.GetFrom3().Offset)
			if op2 < -16 || op2 > 15 {
				c.ctxt.Diag("immediate out of range [-16, 15]: %v", p)
			}
		} else {
			o1 |= 1 << 10
			op2 = int64(p.GetFrom3().Reg)
		}
		zd := p.To.Reg

		ad := c.parseArng(zd)
		sz := c.sveDataSize(ad)

		o1 |= uint32(sz&3)<<22 | uint32(op1&31)<<16 | uint32(op2&31)<<5 | uint32(zd&31)

	case 27: /* punpkhi/punpklo Pn.B, Pd.H */
		switch p.As {
		case APUNPKLO:
			o1 |= 5<<24 | 3<<20 | 0<<16 | 1<<14
		case APUNPKHI:
			o1 |= 5<<24 | 3<<20 | 1<<16 | 1<<14
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		pn := p.From.Reg
		pd := p.To.Reg

		an := c.parseArng(pn)
		ad := c.parseArng(pd)
		if an != ARNG_B || ad != ARNG_H {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(pn&15)<<5 | uint32(pd&15)

	case 28: /* pzip1/pzip2/puzp1/puzp2/ptrn1/ptrn2 Pm.<T>, Pn.<T>, Pd.<T> */
		switch p.As {
		case APZIP1:
			o1 |= 5<<24 | 1<<21 | 1<<14 | 0<<10
		case APZIP2:
			o1 |= 5<<24 | 1<<21 | 1<<14 | 1<<10
		case APUZP1:
			o1 |= 5<<24 | 1<<21 | 1<<14 | 2<<10
		case APUZP2:
			o1 |= 5<<24 | 1<<21 | 1<<14 | 3<<10
		case APTRN1:
			o1 |= 5<<24 | 1<<21 | 1<<14 | 4<<10
		case APTRN2:
			o1 |= 5<<24 | 1<<21 | 1<<14 | 5<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		pm := p.From.Reg
		pn := p.Reg
		pd := p.To.Reg

		am := c.parseArng(pm)
		an := c.parseArng(pn)
		ad := c.parseArng(pd)
		if am != an || am != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		sz := c.sveDataSize(am)

		o1 |= uint32(sz&3)<<22 | uint32(pm&15)<<16 | uint32(pn&15)<<5 | uint32(pd&15)

	case 29: /* zzip1/zzip2/zuzp1/zuzp2/ztrn1/ztrn2 Zm.<T>, Zn.<T>, Zd.<T> */
		switch p.As {
		case AZZIP1:
			o1 |= 5<<24 | 1<<21 | 3<<13 | 0<<10
		case AZZIP2:
			o1 |= 5<<24 | 1<<21 | 3<<13 | 1<<10
		case AZUZP1:
			o1 |= 5<<24 | 1<<21 | 3<<13 | 2<<10
		case AZUZP2:
			o1 |= 5<<24 | 1<<21 | 3<<13 | 3<<10
		case AZTRN1:
			o1 |= 5<<24 | 1<<21 | 3<<13 | 4<<10
		case AZTRN2:
			o1 |= 5<<24 | 1<<21 | 3<<13 | 5<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zm := p.From.Reg
		zn := p.Reg
		zd := p.To.Reg

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		ad := c.parseArng(zd)
		if am != an || am != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		var sz uint32
		if am == ARNG_Q {
			sz = 2
			o1 &^= 3 << 13
			if p.As == AZTRN1 || p.As == AZTRN2 {
				o1 |= 1 << 11
			}
		} else {
			sz = c.sveDataSize(am)
		}

		o1 |= uint32(sz&3)<<22 | uint32(zm&31)<<16 | uint32(zn&31)<<5 | uint32(zd&31)

	case 30: /* pbrkn/pbrkns Pdm.B, Pn.B, Pg/Z, Pdm.B */
		switch p.As {
		case APBRKN:
			o1 |= 0x25<<24 | 0<<22 | 3<<19 | 1<<14
		case APBRKNS:
			o1 |= 0x25<<24 | 1<<22 | 3<<19 | 1<<14
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		pm := p.From.Reg
		pn := p.Reg
		pg := p.GetFrom3().Reg
		pd := p.To.Reg
		if pm != pd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		pdm := pd

		am := c.parseArng(pm)
		an := c.parseArng(pn)
		if am != ARNG_B || an != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(pg&15)<<10 | uint32(pn&15)<<5 | uint32(pdm&15)

	case 31: /* pbrka/pbrkb/pbrkas/pbrkbs Pn.B, Pg/ZM, Pd.B */
		switch p.As {
		case APBRKA:
			o1 |= 0x25<<24 | 0<<22 | 1<<20 | 1<<14
		case APBRKAS:
			o1 |= 0x25<<24 | 1<<22 | 1<<20 | 1<<14
		case APBRKB:
			o1 |= 0x25<<24 | 2<<22 | 1<<20 | 1<<14
		case APBRKBS:
			o1 |= 0x25<<24 | 3<<22 | 1<<20 | 1<<14
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		pn := p.From.Reg
		pg := p.Reg
		pd := p.To.Reg
		if pg>>5&1 == 1 { /* Pg/M */
			o1 |= 1 << 4
		}

		an := c.parseArng(pn)
		ad := c.parseArng(pd)
		if an != ARNG_B || ad != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(pg&15)<<10 | uint32(pn&15)<<5 | uint32(pd&15)
	case 32:
		/* zld1b/zld1h/zld1w/zld1d/zld1sb/zld1sh/zld1sw/zldnf1b/zldnf1h/zldnf1w/zldnf1d/zldnf1sb/zldnf1sh/zldnf1sw
		 *     {(VL*imm)}(Rn|SP), Pg/Z, [Zt.<T>] */
		o1 |= 0x29<<26 | 5<<13
		var ls, es, sn, nf uint32
		switch p.As {
		case AZLD1B:
			ls, sn, nf = 0, 0, 0
		case AZLD1H:
			ls, sn, nf = 1, 0, 0
		case AZLD1W:
			ls, sn, nf = 2, 0, 0
		case AZLD1D:
			ls, sn, nf = 3, 0, 0
		case AZLD1SB:
			ls, sn, nf = 0, 1, 0
		case AZLD1SH:
			ls, sn, nf = 1, 1, 0
		case AZLD1SW:
			ls, sn, nf = 2, 1, 0
		case AZLDNF1B:
			ls, sn, nf = 0, 0, 1
		case AZLDNF1H:
			ls, sn, nf = 1, 0, 1
		case AZLDNF1W:
			ls, sn, nf = 2, 0, 1
		case AZLDNF1D:
			ls, sn, nf = 3, 0, 1
		case AZLDNF1SB:
			ls, sn, nf = 0, 1, 1
		case AZLDNF1SH:
			ls, sn, nf = 1, 1, 1
		case AZLDNF1SW:
			ls, sn, nf = 2, 1, 1
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		rn := p.From.Reg
		imm4 := p.From.Offset
		pg := p.Reg
		zt := p.To.Offset
		if imm4 < -8 || imm4 > 7 {
			c.ctxt.Diag("immediate out of range [-8, 7]: %v", p)
		}
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if ll != 1 {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if at == ARNG_Q {
			if sn == 1 || ls < 2 || nf != 0 {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 ^= 1<<20 | 1<<15
			o1 |= uint32(ls&3) << 23
		} else {
			es = c.sveDataSize(at)
			if es < ls+sn {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			dtype := ls<<2 | es
			if sn == 1 {
				dtype ^= 15
			}
			o1 |= uint32(dtype&15)<<21 | uint32(nf)<<20
		}

		o1 |= uint32(imm4&15)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 33:
		/* zld1b/zld1h/zld1w/zld1d/zld1sb/zld1sh/zld1sw (Rn|SP)(Rm{<<shift}), Pg/Z, [Zt.<T>]
		 * zldff1b/zldff1h/zldff1w/zldff1d/zldff1sb/zldff1sh/zldff1sw (Rn|SP){(Rm{<<shift})}, Pg/Z, [Zt.<T>] */
		o1 |= 0x29<<26 | 2<<13
		var ls, es, sn, ff uint32
		switch p.As {
		case AZLD1B:
			ls, sn, ff = 0, 0, 0
		case AZLD1H:
			ls, sn, ff = 1, 0, 0
		case AZLD1W:
			ls, sn, ff = 2, 0, 0
		case AZLD1D:
			ls, sn, ff = 3, 0, 0
		case AZLD1SB:
			ls, sn, ff = 0, 1, 0
		case AZLD1SH:
			ls, sn, ff = 1, 1, 0
		case AZLD1SW:
			ls, sn, ff = 2, 1, 0
		case AZLDFF1B:
			ls, sn, ff = 0, 0, 1
		case AZLDFF1H:
			ls, sn, ff = 1, 0, 1
		case AZLDFF1W:
			ls, sn, ff = 2, 0, 1
		case AZLDFF1D:
			ls, sn, ff = 3, 0, 1
		case AZLDFF1SB:
			ls, sn, ff = 0, 1, 1
		case AZLDFF1SH:
			ls, sn, ff = 1, 1, 1
		case AZLDFF1SW:
			ls, sn, ff = 2, 1, 1
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		rn := p.From.Reg
		rm := p.From.Index
		if ff == 0 {
			if rm == REGSP {
				c.ctxt.Diag("SP is not allowed for index register: %v", p)
			}
			if (rm < REG_R0 || rm > REG_R30) && (rm < REG_LSL || rm >= REG_ARNG || rm&31 == 31) {
				c.ctxt.Diag("unexpected index register: %v", p)
			}
		} else {
			if rm == 0 && p.From.Offset == 0 { /* default case: zldff1x (Rn|SP), Pg/Z, [Zt.<T>] */
				rm = REGZERO
			}
			if (rm < REG_R0 || rm > REG_R31) && (rm < REG_LSL || rm >= REG_ARNG) {
				c.ctxt.Diag("unexpected index register: %v", p)
			}
		}
		shift := rm >> 5 & 7
		pg := p.Reg
		zt := p.To.Offset
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if ll != 1 {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if ls != uint32(shift) && rm != REGZERO {
			c.ctxt.Diag("incorrect shift amount: %v", p)
		}
		if at == ARNG_Q {
			if sn == 1 || ls < 2 || ff != 0 {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 ^= 3 << 14
			o1 |= uint32(ls&3) << 23
		} else {
			es = c.sveDataSize(at)
			if es < ls+sn {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			dtype := ls<<2 | es
			if sn == 1 {
				dtype ^= 15
			}
			o1 |= uint32(dtype&15)<<21 | uint32(ff)<<13
		}

		o1 |= uint32(rm&31)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 34:
		/* zld2b/zld2h/zld2w/zld2d/zld2q {(VL*imm)}(Rn|SP), Pg/Z, [Zt1.<T>, Zt2.<T>]
		 * zld3b/zld3h/zld3w/zld3d/zld3q {(VL*imm)}(Rn|SP), Pg/Z, [Zt1.<T>, Zt2.<T>, Zt3.<T>]
		 * zld4b/zld4h/zld4w/zld4d/zld4q {(VL*imm)}(Rn|SP), Pg/Z, [Zt1.<T>, Zt2.<T>, Zt3.<T>, Zt4.<T>] */
		o1 |= 0x29<<26 | 7<<13
		var ls, cnt uint32
		switch p.As {
		case AZLD2B:
			ls, cnt = 0, 2
		case AZLD2H:
			ls, cnt = 1, 2
		case AZLD2W:
			ls, cnt = 2, 2
		case AZLD2D:
			ls, cnt = 3, 2
		case AZLD2Q:
			ls, cnt = 4, 2
		case AZLD3B:
			ls, cnt = 0, 3
		case AZLD3H:
			ls, cnt = 1, 3
		case AZLD3W:
			ls, cnt = 2, 3
		case AZLD3D:
			ls, cnt = 3, 3
		case AZLD3Q:
			ls, cnt = 4, 3
		case AZLD4B:
			ls, cnt = 0, 4
		case AZLD4H:
			ls, cnt = 1, 4
		case AZLD4W:
			ls, cnt = 2, 4
		case AZLD4D:
			ls, cnt = 3, 4
		case AZLD4Q:
			ls, cnt = 4, 4
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		rn := p.From.Reg
		imm4 := p.From.Offset
		pg := p.Reg
		zt := p.To.Offset
		if imm4%int64(cnt) != 0 {
			c.ctxt.Diag("invalid immediate, expected multiple of %v: %v", cnt, p)
		}
		imm4 /= int64(cnt)
		if imm4 < -8 || imm4 > 7 {
			c.ctxt.Diag("immediate out of range [%v, %v]: %v", -8*int64(cnt), 7*int64(cnt), p)
		}
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if uint32(ll) != cnt {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if ls == 4 {
			if at != ARNG_Q {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 |= 1 << 20
			o1 |= uint32((cnt-1)&3) << 23
		} else {
			es := c.sveDataSize(at)
			if es != ls {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 |= uint32(ls&3)<<23 | uint32((cnt-1)&3)<<21
		}

		o1 |= uint32(imm4&15)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 35:
		/* zld2b/zld2h/zld2w/zld2d/zld2q (Rn|SP)(Rm{<<shift}), Pg/Z, [Zt1.<T>, Zt2.<T>]
		 * zld3b/zld3h/zld3w/zld3d/zld3q (Rn|SP)(Rm{<<shift}), Pg/Z, [Zt1.<T>, Zt2.<T>, Zt3.<T>]
		 * zld4b/zld4h/zld4w/zld4d/zld4q (Rn|SP)(Rm{<<shift}), Pg/Z, [Zt1.<T>, Zt2.<T>, Zt3.<T>, Zt4.<T>] */
		o1 |= 0x29<<26 | 6<<13
		var ls, cnt uint32
		switch p.As {
		case AZLD2B:
			ls, cnt = 0, 2
		case AZLD2H:
			ls, cnt = 1, 2
		case AZLD2W:
			ls, cnt = 2, 2
		case AZLD2D:
			ls, cnt = 3, 2
		case AZLD2Q:
			ls, cnt = 4, 2
		case AZLD3B:
			ls, cnt = 0, 3
		case AZLD3H:
			ls, cnt = 1, 3
		case AZLD3W:
			ls, cnt = 2, 3
		case AZLD3D:
			ls, cnt = 3, 3
		case AZLD3Q:
			ls, cnt = 4, 3
		case AZLD4B:
			ls, cnt = 0, 4
		case AZLD4H:
			ls, cnt = 1, 4
		case AZLD4W:
			ls, cnt = 2, 4
		case AZLD4D:
			ls, cnt = 3, 4
		case AZLD4Q:
			ls, cnt = 4, 4
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		rn := p.From.Reg
		rm := p.From.Index
		if rm == REGSP {
			c.ctxt.Diag("SP is not allowed for index register: %v", p)
		}
		if (rm < REG_R0 || rm > REG_R30) && (rm < REG_LSL || rm >= REG_ARNG || rm&31 == 31) {
			c.ctxt.Diag("unexpected index register: %v", p)
		}
		shift := rm >> 5 & 7
		pg := p.Reg
		zt := p.To.Offset
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if uint32(ll) != cnt {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if ls != uint32(shift) {
			c.ctxt.Diag("incorrect shift amount: %v", p)
		}
		if ls == 4 {
			if at != ARNG_Q {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 ^= 1<<21 | 1<<14
			o1 |= uint32((cnt-1)&3) << 23
		} else {
			es := c.sveDataSize(at)
			if es != ls {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 |= uint32(ls&3)<<23 | uint32((cnt-1)&3)<<21
		}

		o1 |= uint32(rm&31)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 36: /* zaese/zaesd Zm.B, Zdn.B, Zdn.B */
		switch p.As {
		case AZAESE:
			o1 |= 0x45<<24 | 1<<21 | 1<<17 | 7<<13 | 0<<10
		case AZAESD:
			o1 |= 0x45<<24 | 1<<21 | 1<<17 | 7<<13 | 1<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zm := p.From.Reg
		zn := p.Reg
		zd := p.To.Reg
		if zn != zd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		zdn := zd

		am := c.parseArng(zm)
		an := c.parseArng(zn)
		if am != ARNG_B || an != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(zm&31)<<5 | uint32(zdn&31)

	case 37: /* zaesmc/zaesimc Zdn.B, Zdn.B */
		switch p.As {
		case AZAESMC:
			o1 |= 0x45<<24 | 1<<21 | 7<<13 | 0<<10
		case AZAESIMC:
			o1 |= 0x45<<24 | 1<<21 | 7<<13 | 1<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zn := p.From.Reg
		zd := p.To.Reg
		if zn != zd {
			c.ctxt.Diag("register mismatch: %v", p)
		}
		zdn := zd

		an := c.parseArng(zn)
		if an != ARNG_B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(zdn & 31)

	case 38: /* zst1b/zst1h/zst1w/zst1d [Zt.<T>], Pg, {(VL*imm)}(Rn|SP) */
		o1 |= 0x39<<26 | 7<<13
		var ls, es uint32
		switch p.As {
		case AZST1B:
			ls = 0
		case AZST1H:
			ls = 1
		case AZST1W:
			ls = 2
		case AZST1D:
			ls = 3
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zt := p.From.Offset
		rn := p.To.Reg
		pg := p.Reg
		imm4 := p.To.Offset
		if imm4 < -8 || imm4 > 7 {
			c.ctxt.Diag("immediate out of range [-8, 7]: %v", p)
		}
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if ll != 1 {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if at == ARNG_Q {
			switch p.As {
			case AZST1W:
				es = 0
			case AZST1D:
				es = 2
			default:
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		} else {
			es = c.sveDataSize(at)
			if es < ls {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		}

		o1 |= uint32(ls&3)<<23 | uint32(es&3)<<21 | uint32(imm4&15)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 39: /* zst1b/zst1h/zst1w/zst1d [Zt.<T>], Pg, (Rn|SP)(Rm{<<shift}) */
		o1 |= 0x39<<26 | 2<<13
		var ls, es uint32
		switch p.As {
		case AZST1B:
			ls = 0
		case AZST1H:
			ls = 1
		case AZST1W:
			ls = 2
		case AZST1D:
			ls = 3
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zt := p.From.Offset
		pg := p.Reg
		rn := p.To.Reg
		rm := p.To.Index
		if rm == REGSP {
			c.ctxt.Diag("SP is not allowed for index register: %v", p)
		}
		if (rm < REG_R0 || rm > REG_R30) && (rm < REG_LSL || rm >= REG_ARNG || rm&31 == 31) {
			c.ctxt.Diag("unexpected index register: %v", p)
		}
		shift := rm >> 5 & 7
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if ll != 1 {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if ls != uint32(shift) {
			c.ctxt.Diag("incorrect shift amount: %v", p)
		}
		if at == ARNG_Q {
			switch p.As {
			case AZST1W:
				es = 0
			case AZST1D:
				es = 2
			default:
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		} else {
			es = c.sveDataSize(at)
			if es < ls {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		}

		o1 |= uint32(ls&3)<<23 | uint32(es&3)<<21 | uint32(rm&31)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 40:
		/* zst2b/zst2h/zst2w/zst2d/zst2q [Zt1.<T>, Zt2.<T>], Pg/Z, {(VL*imm)}(Rn|SP)
		 * zst3b/zst3h/zst3w/zst3d/zst3q [Zt1.<T>, Zt2.<T>, Zt3.<T>], Pg/Z, {(VL*imm)}(Rn|SP)
		 * zst4b/zst4h/zst4w/zst4d/zst4q [Zt1.<T>, Zt2.<T>, Zt3.<T>, Zt4.<T>], Pg/Z, {(VL*imm)}(Rn|SP) */
		o1 |= 0x39<<26 | 1<<20 | 7<<13
		var ls, cnt uint32
		switch p.As {
		case AZST2B:
			ls, cnt = 0, 2
		case AZST2H:
			ls, cnt = 1, 2
		case AZST2W:
			ls, cnt = 2, 2
		case AZST2D:
			ls, cnt = 3, 2
		case AZST2Q:
			ls, cnt = 4, 2
		case AZST3B:
			ls, cnt = 0, 3
		case AZST3H:
			ls, cnt = 1, 3
		case AZST3W:
			ls, cnt = 2, 3
		case AZST3D:
			ls, cnt = 3, 3
		case AZST3Q:
			ls, cnt = 4, 3
		case AZST4B:
			ls, cnt = 0, 4
		case AZST4H:
			ls, cnt = 1, 4
		case AZST4W:
			ls, cnt = 2, 4
		case AZST4D:
			ls, cnt = 3, 4
		case AZST4Q:
			ls, cnt = 4, 4
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zt := p.From.Offset
		pg := p.Reg
		rn := p.To.Reg
		imm4 := p.To.Offset
		if imm4%int64(cnt) != 0 {
			c.ctxt.Diag("invalid immediate, expected multiple of %v: %v", cnt, p)
		}
		imm4 /= int64(cnt)
		if imm4 < -8 || imm4 > 7 {
			c.ctxt.Diag("immediate out of range [%v, %v]: %v", -8*int64(cnt), 7*int64(cnt), p)
		}
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if uint32(ll) != cnt {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if ls == 4 {
			if at != ARNG_Q {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 ^= 1<<20 | 7<<13
			o1 |= uint32((cnt-1)&3) << 22
		} else {
			es := c.sveDataSize(at)
			if es != ls {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 |= uint32(ls&3)<<23 | uint32((cnt-1)&3)<<21
		}

		o1 |= uint32(imm4&15)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 41:
		/* zst2b/zst2h/zst2w/zst2d/zst2q [Zt1.<T>, Zt2.<T>], Pg/Z, (Rn|SP)(Rm{<<shift})
		 * zst3b/zst3h/zst3w/zst3d/zst3q [Zt1.<T>, Zt2.<T>, Zt3.<T>], Pg/Z, (Rn|SP)(Rm{<<shift})
		 * zst4b/zst4h/zst4w/zst4d/zst4q [Zt1.<T>, Zt2.<T>, Zt3.<T>, Zt4.<T>], Pg/Z, (Rn|SP)(Rm{<<shift}) */
		o1 |= 0x39<<26 | 3<<13
		var ls, cnt uint32
		switch p.As {
		case AZST2B:
			ls, cnt = 0, 2
		case AZST2H:
			ls, cnt = 1, 2
		case AZST2W:
			ls, cnt = 2, 2
		case AZST2D:
			ls, cnt = 3, 2
		case AZST2Q:
			ls, cnt = 4, 2
		case AZST3B:
			ls, cnt = 0, 3
		case AZST3H:
			ls, cnt = 1, 3
		case AZST3W:
			ls, cnt = 2, 3
		case AZST3D:
			ls, cnt = 3, 3
		case AZST3Q:
			ls, cnt = 4, 3
		case AZST4B:
			ls, cnt = 0, 4
		case AZST4H:
			ls, cnt = 1, 4
		case AZST4W:
			ls, cnt = 2, 4
		case AZST4D:
			ls, cnt = 3, 4
		case AZST4Q:
			ls, cnt = 4, 4
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zt := p.From.Offset
		pg := p.Reg
		rn := p.To.Reg
		rm := p.To.Index
		if rm == REGSP {
			c.ctxt.Diag("SP is not allowed for index register: %v", p)
		}
		if (rm < REG_R0 || rm > REG_R30) && (rm < REG_LSL || rm >= REG_ARNG || rm&31 == 31) {
			c.ctxt.Diag("unexpected index register: %v", p)
		}
		shift := rm >> 5 & 7
		c.checkGPreg(p, pg)
		ll := zt >> 9 & 15
		if uint32(ll) != cnt {
			c.ctxt.Diag("invalid length of reg list: %v", p)
		}

		at := c.parseArng(int16(zt))
		if ls != uint32(shift) {
			c.ctxt.Diag("incorrect shift amount: %v", p)
		}
		if ls == 4 {
			if at != ARNG_Q {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 ^= 1<<21 | 3<<13
			o1 |= uint32((cnt-1)&3) << 22
		} else {
			es := c.sveDataSize(at)
			if es != ls {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			o1 |= uint32(ls&3)<<23 | uint32((cnt-1)&3)<<21
		}

		o1 |= uint32(rm&31)<<16 | uint32(pg&7)<<10 | uint32(rn&31)<<5 | uint32(zt&31)

	case 42: /* zincb/zinch/zincw/zincd/zdecb/zdech/zdecw/zdecd {{$imm, }pattern, }Rdn */
		switch p.As {
		case AZCNTB:
			o1 |= 1<<26 | 0<<22 | 2<<20 | 7<<13
		case AZCNTH:
			o1 |= 1<<26 | 1<<22 | 2<<20 | 7<<13
		case AZCNTW:
			o1 |= 1<<26 | 2<<22 | 2<<20 | 7<<13
		case AZCNTD:
			o1 |= 1<<26 | 3<<22 | 2<<20 | 7<<13
		case AZINCB:
			o1 |= 1<<26 | 0<<22 | 3<<20 | 7<<13 | 0<<10
		case AZDECB:
			o1 |= 1<<26 | 0<<22 | 3<<20 | 7<<13 | 1<<10
		case AZINCH:
			o1 |= 1<<26 | 1<<22 | 3<<20 | 7<<13 | 0<<10
		case AZDECH:
			o1 |= 1<<26 | 1<<22 | 3<<20 | 7<<13 | 1<<10
		case AZINCW:
			o1 |= 1<<26 | 2<<22 | 3<<20 | 7<<13 | 0<<10
		case AZDECW:
			o1 |= 1<<26 | 2<<22 | 3<<20 | 7<<13 | 1<<10
		case AZINCD:
			o1 |= 1<<26 | 3<<22 | 3<<20 | 7<<13 | 0<<10
		case AZDECD:
			o1 |= 1<<26 | 3<<22 | 3<<20 | 7<<13 | 1<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		var imm4 uint32
		if p.From.Type == obj.TYPE_NONE {
			imm4 = 0
		} else {
			imm := p.From.Offset
			if imm < 1 || imm > 16 {
				c.ctxt.Diag("immediate out of range [1, 16]: %v", p)
			}
			imm4 = uint32(imm - 1)
		}
		var pat Pattern
		if p.From3Type() == obj.TYPE_NONE {
			pat = PAT_ALL
		} else {
			pat = Pattern(p.GetFrom3().Offset)
		}
		if pat&^31 != 0 {
			c.ctxt.Diag("illegal pattern code: %v", p)
		}
		rdn := p.To.Reg

		o1 |= uint32(imm4&15)<<16 | uint32(pat&31)<<5 | uint32(rdn&31)

	case 43: /* zinch/zincw/zincd/zdech/zdecw/zdecd {{$imm, }pattern, }Zdn.<T> */
		switch p.As {
		case AZINCH:
			o1 |= 1<<26 | 1<<22 | 3<<20 | 3<<14 | 0<<10
		case AZDECH:
			o1 |= 1<<26 | 1<<22 | 3<<20 | 3<<14 | 1<<10
		case AZINCW:
			o1 |= 1<<26 | 2<<22 | 3<<20 | 3<<14 | 0<<10
		case AZDECW:
			o1 |= 1<<26 | 2<<22 | 3<<20 | 3<<14 | 1<<10
		case AZINCD:
			o1 |= 1<<26 | 3<<22 | 3<<20 | 3<<14 | 0<<10
		case AZDECD:
			o1 |= 1<<26 | 3<<22 | 3<<20 | 3<<14 | 1<<10
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		var imm4 uint32
		if p.From.Type == obj.TYPE_NONE {
			imm4 = 0
		} else {
			imm := p.From.Offset
			if imm < 1 || imm > 16 {
				c.ctxt.Diag("immediate out of range [1, 16]: %v", p)
			}
			imm4 = uint32(imm - 1)
		}
		var pat Pattern
		if p.From3Type() == obj.TYPE_NONE {
			pat = PAT_ALL
		} else {
			pat = Pattern(p.GetFrom3().Offset)
		}
		if pat&^31 != 0 {
			c.ctxt.Diag("illegal pattern code: %v", p)
		}
		zdn := p.To.Reg

		adn := c.parseArng(zdn)
		sz := o1 >> 22 & 3
		if sz != c.sveDataSize(adn) {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		o1 |= uint32(imm4&15)<<16 | uint32(pat&31)<<5 | uint32(zdn&31)

	case 44: /* zlasta/zlastb Zn.<T>, Pg, Rd */
		switch p.As {
		case AZLASTA:
			o1 |= 5<<24 | 1<<21 | 0<<16 | 5<<13
		case AZLASTB:
			o1 |= 5<<24 | 1<<21 | 1<<16 | 5<<13
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zn := p.From.Reg
		pg := p.Reg
		rd := p.To.Reg

		an := c.parseArng(zn)
		c.checkGPreg(p, pg)

		sz := c.sveDataSize(an)

		o1 |= uint32(sz&3)<<22 | uint32(pg&7)<<10 | uint32(zn&31)<<5 | uint32(rd&31)

	case 45: /* zlasta/zlastb Zn.<T>, Pg, Vd.<T> */
		switch p.As {
		case AZLASTA:
			o1 |= 5<<24 | 1<<21 | 1<<17 | 0<<16 | 5<<13
		case AZLASTB:
			o1 |= 5<<24 | 1<<21 | 1<<17 | 1<<16 | 5<<13
		default:
			c.ctxt.Diag("unexpected asm: %v", p)
		}

		zn := p.From.Reg
		pg := p.Reg
		vd := p.To.Reg

		an := c.parseArng(zn)
		ad := c.parseArng(vd)
		if an != ad {
			c.ctxt.Diag("data size mismatch: %v", p)
		}
		c.checkGPreg(p, pg)

		sz := c.sveDataSize(an)

		o1 |= uint32(sz&3)<<22 | uint32(pg&7)<<10 | uint32(zn&31)<<5 | uint32(vd&31)

	case 46: /* zcntp Pn.<T>, Pg, Rd */
		o1 |= 0x25<<24 | 1<<21 | 1<<15

		pn := p.From.Reg
		pg := p.Reg
		rd := p.To.Reg

		an := c.parseArng(pn)
		sz := c.sveDataSize(an)

		o1 |= uint32(sz&3)<<22 | uint32(pg&15)<<10 | uint32(pn&15)<<5 | uint32(rd&31)

	case 47: /* zsaddv/zuaddv Zn.<T>, Pg, Vd.D2 */
		switch p.As {
		case AZSADDV:
			o1 |= 1<<26 | 0<<19 | 0<<16 | 1<<13
		case AZUADDV:
			o1 |= 1<<26 | 0<<19 | 1<<16 | 1<<13
		}

		zn := p.From.Reg
		pg := p.Reg
		vd := p.To.Reg

		an := c.parseArng(zn)
		sz := c.sveDataSize(an)
		if p.As == AZSADDV && sz == 3 {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		c.checkGPreg(p, pg)

		o1 |= uint32(sz&3)<<22 | uint32(pg&7)<<10 | uint32(zn&31)<<5 | uint32(vd&31)
	}

	out[0] = o1
	return int(o.size(c.ctxt, p) / 4)
}

func (c *ctxt7) asmout(p *obj.Prog, out []uint32) (count int) {
	o := c.oplook(p)

	var os [5]uint32
	o1 := uint32(0)
	o2 := uint32(0)
	o3 := uint32(0)
	o4 := uint32(0)
	o5 := uint32(0)
	if false { /*debug['P']*/
		fmt.Printf("%x: %v\ttype %d\n", uint32(p.Pc), p, o.type_)
	}
	switch o.type_ {
	default:
		c.ctxt.Diag("%v: unknown asm %d", p, o.type_)

	case 0: /* pseudo ops */
		break

	case 1: /* op Rm,[Rn],Rd; default Rn=Rd -> op Rm<<0,[Rn,]Rd (shifted register) */
		o1 = c.oprrr(p, p.As)

		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		r := int(p.Reg)
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		if r == obj.REG_NONE {
			r = rt
		}
		o1 |= (uint32(rf&31) << 16) | (uint32(r&31) << 5) | uint32(rt&31)

	case 2: /* add/sub $(uimm12|uimm24)[,R],R; cmp $(uimm12|uimm24),R */
		if p.To.Reg == REG_RSP && isADDSop(p.As) {
			c.ctxt.Diag("illegal destination register: %v\n", p)
		}
		o1 = c.opirr(p, p.As)

		rt, r := p.To.Reg, p.Reg
		if p.To.Type == obj.TYPE_NONE {
			if (o1 & Sbit) == 0 {
				c.ctxt.Diag("ineffective ZR destination\n%v", p)
			}
			rt = REGZERO
		}
		if r == obj.REG_NONE {
			r = rt
		}
		v := c.regoff(&p.From)
		o1 = c.oaddi(p, p.As, v, rt, r)

	case 3: /* op R<<n[,R],R (shifted register) */
		o1 = c.oprrr(p, p.As)

		amount := (p.From.Offset >> 10) & 63
		is64bit := o1 & (1 << 31)
		if is64bit == 0 && amount >= 32 {
			c.ctxt.Diag("shift amount out of range 0 to 31: %v", p)
		}
		shift := (p.From.Offset >> 22) & 3
		if (shift > 2 || shift < 0) && (isADDop(p.As) || isADDWop(p.As) || isNEGop(p.As)) {
			c.ctxt.Diag("unsupported shift operator: %v", p)
		}
		o1 |= uint32(p.From.Offset) /* includes reg, op, etc */
		rt := int(p.To.Reg)
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		r := int(p.Reg)
		if p.As == AMVN || p.As == AMVNW || isNEGop(p.As) {
			r = REGZERO
		} else if r == obj.REG_NONE {
			r = rt
		}
		o1 |= (uint32(r&31) << 5) | uint32(rt&31)

	case 4: /* mov $addcon, R; mov $recon, R; mov $racon, R; mov $addcon2, R */
		rt, r := p.To.Reg, o.param
		if r == obj.REG_NONE {
			r = REGZERO
		} else if r == REGFROM {
			r = p.From.Reg
		}
		if r == obj.REG_NONE {
			r = REGSP
		}

		v := c.regoff(&p.From)
		a := AADD
		if v < 0 {
			a = ASUB
			v = -v
		}

		if o.size(c.ctxt, p) == 8 {
			// NOTE: this case does not use REGTMP. If it ever does,
			// remove the NOTUSETMP flag in optab.
			o1 = c.oaddi(p, a, v&0xfff000, rt, r)
			o2 = c.oaddi(p, a, v&0x000fff, rt, rt)
			break
		}

		o1 = c.oaddi(p, a, v, rt, r)

	case 5: /* b s; bl s */
		o1 = c.opbra(p, p.As)

		if p.To.Sym == nil {
			o1 |= uint32(c.brdist(p, 0, 26, 2))
			break
		}

		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: objabi.R_CALLARM64,
			Off:  int32(c.pc),
			Siz:  4,
			Sym:  p.To.Sym,
			Add:  p.To.Offset,
		})

	case 6: /* b ,O(R); bl ,O(R) */
		o1 = c.opbrr(p, p.As)
		o1 |= uint32(p.To.Reg&31) << 5
		if p.As == obj.ACALL {
			c.cursym.AddRel(c.ctxt, obj.Reloc{
				Type: objabi.R_CALLIND,
				Off:  int32(c.pc),
			})
		}

	case 7: /* beq s */
		o1 = c.opbra(p, p.As)

		o1 |= uint32(c.brdist(p, 0, 19, 2) << 5)

	case 8: /* lsl $c,[R],R -> ubfm $(W-1)-c,$(-c MOD (W-1)),Rn,Rd */
		rt, rf := p.To.Reg, p.Reg
		if rf == obj.REG_NONE {
			rf = rt
		}
		v := p.From.Offset
		switch p.As {
		case AASR:
			o1 = c.opbfm(p, ASBFM, v, 63, rf, rt)

		case AASRW:
			o1 = c.opbfm(p, ASBFMW, v, 31, rf, rt)

		case ALSL:
			o1 = c.opbfm(p, AUBFM, (64-v)&63, 63-v, rf, rt)

		case ALSLW:
			o1 = c.opbfm(p, AUBFMW, (32-v)&31, 31-v, rf, rt)

		case ALSR:
			o1 = c.opbfm(p, AUBFM, v, 63, rf, rt)

		case ALSRW:
			o1 = c.opbfm(p, AUBFMW, v, 31, rf, rt)

		case AROR:
			o1 = c.opextr(p, AEXTR, v, rf, rf, rt)

		case ARORW:
			o1 = c.opextr(p, AEXTRW, v, rf, rf, rt)

		default:
			c.ctxt.Diag("bad shift $con\n%v", p)
			break
		}

	case 9: /* lsl Rm,[Rn],Rd -> lslv Rm, Rn, Rd */
		o1 = c.oprrr(p, p.As)

		r := int(p.Reg)
		if r == obj.REG_NONE {
			r = int(p.To.Reg)
		}
		o1 |= (uint32(p.From.Reg&31) << 16) | (uint32(r&31) << 5) | uint32(p.To.Reg&31)

	case 10: /* brk/hvc/.../svc [$con] */
		o1 = c.opimm(p, p.As)

		if p.From.Type != obj.TYPE_NONE {
			o1 |= uint32((p.From.Offset & 0xffff) << 5)
		}

	case 11: /* dword */
		c.aclass(&p.To)

		o1 = uint32(c.instoffset)
		o2 = uint32(c.instoffset >> 32)
		if p.To.Sym != nil {
			c.cursym.AddRel(c.ctxt, obj.Reloc{
				Type: objabi.R_ADDR,
				Off:  int32(c.pc),
				Siz:  8,
				Sym:  p.To.Sym,
				Add:  p.To.Offset,
			})
			o2 = 0
			o1 = o2
		}

	case 12: /* movT $vcon, reg */
		// NOTE: this case does not use REGTMP. If it ever does,
		// remove the NOTUSETMP flag in optab.
		num := c.omovlconst(p.As, p, &p.From, int(p.To.Reg), os[:])
		if num == 0 {
			c.ctxt.Diag("invalid constant: %v", p)
		}
		o1 = os[0]
		o2 = os[1]
		o3 = os[2]
		o4 = os[3]

	case 13: /* addop $vcon, [R], R (64 bit literal); cmp $lcon,R -> addop $lcon,R, ZR */
		if p.Reg == REGTMP {
			c.ctxt.Diag("cannot use REGTMP as source: %v\n", p)
		}
		if p.To.Reg == REG_RSP && isADDSop(p.As) {
			c.ctxt.Diag("illegal destination register: %v\n", p)
		}
		o := uint32(0)
		num := uint8(0)
		cls := int(p.From.Class)
		if isADDWop(p.As) {
			if !cmp(C_LCON, cls) {
				c.ctxt.Diag("illegal combination: %v", p)
			}
			num = c.omovlconst(AMOVW, p, &p.From, REGTMP, os[:])
		} else {
			num = c.omovlconst(AMOVD, p, &p.From, REGTMP, os[:])
		}
		if num == 0 {
			c.ctxt.Diag("invalid constant: %v", p)
		}

		rt, r, rf := p.To.Reg, p.Reg, int16(REGTMP)
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		if r == obj.REG_NONE {
			r = rt
		}
		if p.To.Type != obj.TYPE_NONE && (rt == REGSP || r == REGSP) {
			o = c.opxrrr(p, p.As, rt, r, rf, false)
			o |= LSL0_64
		} else {
			o = c.oprrr(p, p.As)
			o |= uint32(rf&31) << 16 /* shift is 0 */
			o |= uint32(r&31) << 5
			o |= uint32(rt & 31)
		}

		os[num] = o
		o1 = os[0]
		o2 = os[1]
		o3 = os[2]
		o4 = os[3]
		o5 = os[4]

	case 14: /* word */
		if c.aclass(&p.To) == C_ADDR {
			c.ctxt.Diag("address constant needs DWORD\n%v", p)
		}
		o1 = uint32(c.instoffset)
		if p.To.Sym != nil {
			// This case happens with words generated
			// in the PC stream as part of the literal pool.
			c.cursym.AddRel(c.ctxt, obj.Reloc{
				Type: objabi.R_ADDR,
				Off:  int32(c.pc),
				Siz:  4,
				Sym:  p.To.Sym,
				Add:  p.To.Offset,
			})
			o1 = 0
		}

	case 15: /* mul/mneg/umulh/umull r,[r,]r; madd/msub/fmadd/fmsub/fnmadd/fnmsub Rm,Ra,Rn,Rd */
		o1 = c.oprrr(p, p.As)

		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		var r int
		var ra int
		if p.From3Type() == obj.TYPE_REG {
			r = int(p.GetFrom3().Reg)
			ra = int(p.Reg)
			if ra == obj.REG_NONE {
				ra = REGZERO
			}
		} else {
			r = int(p.Reg)
			if r == obj.REG_NONE {
				r = rt
			}
			ra = REGZERO
		}

		o1 |= (uint32(rf&31) << 16) | (uint32(ra&31) << 10) | (uint32(r&31) << 5) | uint32(rt&31)

	case 16: /* XremY R[,R],R -> XdivY; XmsubY */
		o1 = c.oprrr(p, p.As)

		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		r := int(p.Reg)
		if r == obj.REG_NONE {
			r = rt
		}
		o1 |= (uint32(rf&31) << 16) | (uint32(r&31) << 5) | REGTMP&31
		o2 = c.oprrr(p, AMSUBW)
		o2 |= o1 & (1 << 31) /* same size */
		o2 |= (uint32(rf&31) << 16) | (uint32(r&31) << 10) | (REGTMP & 31 << 5) | uint32(rt&31)

	case 17: /* op Rm,[Rn],Rd; default Rn=ZR */
		o1 = c.oprrr(p, p.As)

		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		r := int(p.Reg)
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		if r == obj.REG_NONE {
			r = REGZERO
		}
		o1 |= (uint32(rf&31) << 16) | (uint32(r&31) << 5) | uint32(rt&31)

	case 18: /* csel cond,Rn,Rm,Rd; cinc/cinv/cneg cond,Rn,Rd; cset cond,Rd */
		o1 = c.oprrr(p, p.As)

		cond := SpecialOperand(p.From.Offset)
		if cond < SPOP_EQ || cond > SPOP_NV || (cond == SPOP_AL || cond == SPOP_NV) && p.From3Type() == obj.TYPE_NONE {
			c.ctxt.Diag("invalid condition: %v", p)
		} else {
			cond -= SPOP_EQ
		}

		r := int(p.Reg)
		var rf int = r
		if p.From3Type() == obj.TYPE_NONE {
			/* CINC/CINV/CNEG or CSET/CSETM*/
			if r == obj.REG_NONE {
				/* CSET/CSETM */
				rf = REGZERO
				r = rf
			}
			cond ^= 1
		} else {
			rf = int(p.GetFrom3().Reg) /* CSEL */
		}

		rt := int(p.To.Reg)
		o1 |= (uint32(rf&31) << 16) | (uint32(cond&15) << 12) | (uint32(r&31) << 5) | uint32(rt&31)

	case 19: /* CCMN cond, (Rm|uimm5),Rn, uimm4 -> ccmn Rn,Rm,uimm4,cond */
		nzcv := int(p.To.Offset)

		cond := SpecialOperand(p.From.Offset)
		if cond < SPOP_EQ || cond > SPOP_NV {
			c.ctxt.Diag("invalid condition\n%v", p)
		} else {
			cond -= SPOP_EQ
		}
		var rf int
		if p.GetFrom3().Type == obj.TYPE_REG {
			o1 = c.oprrr(p, p.As)
			rf = int(p.GetFrom3().Reg) /* Rm */
		} else {
			o1 = c.opirr(p, p.As)
			rf = int(p.GetFrom3().Offset & 0x1F)
		}

		o1 |= (uint32(rf&31) << 16) | (uint32(cond&15) << 12) | (uint32(p.Reg&31) << 5) | uint32(nzcv)

	case 20: /* movT R,O(R) -> strT */
		v := c.regoff(&p.To)
		sz := int32(1 << uint(movesize(p.As)))

		rt, rf := p.To.Reg, p.From.Reg
		if rt == obj.REG_NONE {
			rt = o.param
		}
		if v < 0 || v%sz != 0 { /* unscaled 9-bit signed */
			o1 = c.olsr9s(p, c.opstr(p, p.As), v, rt, rf)
		} else {
			v = int32(c.offsetshift(p, int64(v), int(o.a4)))
			o1 = c.olsr12u(p, c.opstr(p, p.As), v, rt, rf)
		}

	case 21: /* movT O(R),R -> ldrT */
		v := c.regoff(&p.From)
		sz := int32(1 << uint(movesize(p.As)))

		rt, rf := p.To.Reg, p.From.Reg
		if rf == obj.REG_NONE {
			rf = o.param
		}
		if v < 0 || v%sz != 0 { /* unscaled 9-bit signed */
			o1 = c.olsr9s(p, c.opldr(p, p.As), v, rf, rt)
		} else {
			v = int32(c.offsetshift(p, int64(v), int(o.a1)))
			o1 = c.olsr12u(p, c.opldr(p, p.As), v, rf, rt)
		}

	case 22: /* movT (R)O!,R; movT O(R)!, R -> ldrT */
		if p.From.Reg != REGSP && p.From.Reg == p.To.Reg {
			c.ctxt.Diag("constrained unpredictable behavior: %v", p)
		}

		v := int32(p.From.Offset)

		if v < -256 || v > 255 {
			c.ctxt.Diag("offset out of range [-256,255]: %v", p)
		}
		o1 = c.opldr(p, p.As)
		if o.scond == C_XPOST {
			o1 |= 1 << 10
		} else {
			o1 |= 3 << 10
		}
		o1 |= ((uint32(v) & 0x1FF) << 12) | (uint32(p.From.Reg&31) << 5) | uint32(p.To.Reg&31)

	case 23: /* movT R,(R)O!; movT O(R)!, R -> strT */
		if p.To.Reg != REGSP && p.From.Reg == p.To.Reg {
			c.ctxt.Diag("constrained unpredictable behavior: %v", p)
		}

		v := int32(p.To.Offset)

		if v < -256 || v > 255 {
			c.ctxt.Diag("offset out of range [-256,255]: %v", p)
		}
		o1 = c.opstr(p, p.As)
		if o.scond == C_XPOST {
			o1 |= 1 << 10
		} else {
			o1 |= 3 << 10
		}
		o1 |= ((uint32(v) & 0x1FF) << 12) | (uint32(p.To.Reg&31) << 5) | uint32(p.From.Reg&31)

	case 24: /* mov/mvn Rs,Rd -> add $0,Rs,Rd or orr Rs,ZR,Rd */
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		if rf == REGSP || rt == REGSP {
			if p.As == AMVN || p.As == AMVNW {
				c.ctxt.Diag("illegal SP reference\n%v", p)
			}
			o1 = c.opirr(p, p.As)
			o1 |= (uint32(rf&31) << 5) | uint32(rt&31)
		} else {
			o1 = c.oprrr(p, p.As)
			o1 |= (uint32(rf&31) << 16) | (REGZERO & 31 << 5) | uint32(rt&31)
		}

	case 25: /* negX Rs, Rd -> subX Rs<<0, ZR, Rd */
		o1 = c.oprrr(p, p.As)

		rf := int(p.From.Reg)
		if rf == C_NONE {
			rf = int(p.To.Reg)
		}
		rt := int(p.To.Reg)
		o1 |= (uint32(rf&31) << 16) | (REGZERO & 31 << 5) | uint32(rt&31)

	case 26: /* op Vn, Vd; op Vn.<T>, Vd.<T> */
		o1 = c.oprrr(p, p.As)
		cf := c.aclass(&p.From)
		af := (p.From.Reg >> 5) & 15
		at := (p.To.Reg >> 5) & 15
		var sz int16
		switch p.As {
		case AAESD, AAESE, AAESIMC, AAESMC:
			sz = ARNG_16B
		case ASHA1SU1, ASHA256SU0:
			sz = ARNG_4S
		case ASHA512SU0:
			sz = ARNG_2D
		}

		if cf == C_ARNG {
			if p.As == ASHA1H {
				c.ctxt.Diag("invalid operands: %v", p)
			} else {
				if af != sz || af != at {
					c.ctxt.Diag("invalid arrangement: %v", p)
				}
			}
		}
		o1 |= uint32(p.From.Reg&31)<<5 | uint32(p.To.Reg&31)

	case 27: /* op Rm<<n[,Rn],Rd (extended register) */
		if p.To.Reg == REG_RSP && isADDSop(p.As) {
			c.ctxt.Diag("illegal destination register: %v\n", p)
		}
		rt, r, rf := p.To.Reg, p.Reg, p.From.Reg
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		if r == obj.REG_NONE {
			r = rt
		}
		if (p.From.Reg-obj.RBaseARM64)&REG_EXT != 0 ||
			(p.From.Reg >= REG_LSL && p.From.Reg < REG_ARNG) {
			amount := (p.From.Reg >> 5) & 7
			if amount > 4 {
				c.ctxt.Diag("shift amount out of range 0 to 4: %v", p)
			}
			o1 = c.opxrrr(p, p.As, rt, r, obj.REG_NONE, true)
			o1 |= c.encRegShiftOrExt(p, &p.From, p.From.Reg) /* includes reg, op, etc */
		} else {
			o1 = c.opxrrr(p, p.As, rt, r, rf, false)
		}

	case 28: /* logop $vcon, [R], R (64 bit literal) */
		if p.Reg == REGTMP {
			c.ctxt.Diag("cannot use REGTMP as source: %v\n", p)
		}
		o := uint32(0)
		num := uint8(0)
		cls := int(p.From.Class)
		if isANDWop(p.As) {
			if !cmp(C_LCON, cls) {
				c.ctxt.Diag("illegal combination: %v", p)
			}
			num = c.omovlconst(AMOVW, p, &p.From, REGTMP, os[:])
		} else {
			num = c.omovlconst(AMOVD, p, &p.From, REGTMP, os[:])
		}

		if num == 0 {
			c.ctxt.Diag("invalid constant: %v", p)
		}
		rt := int(p.To.Reg)
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		r := int(p.Reg)
		if r == obj.REG_NONE {
			r = rt
		}
		o = c.oprrr(p, p.As)
		o |= REGTMP & 31 << 16 /* shift is 0 */
		o |= uint32(r&31) << 5
		o |= uint32(rt & 31)

		os[num] = o
		o1 = os[0]
		o2 = os[1]
		o3 = os[2]
		o4 = os[3]
		o5 = os[4]

	case 29: /* op Rn, Rd */
		fc := c.aclass(&p.From)
		tc := c.aclass(&p.To)
		if (p.As == AFMOVD || p.As == AFMOVS) && (fc == C_REG || fc == C_REGZR || tc == C_REG || tc == C_REGZR) {
			// FMOV Rx, Fy or FMOV Fy, Rx
			o1 = FPCVTI(0, 0, 0, 0, 6)
			if p.As == AFMOVD {
				o1 |= 1<<31 | 1<<22 // 64-bit
			}
			if fc == C_REG || fc == C_REGZR {
				o1 |= 1 << 16 // FMOV Rx, Fy
			}
		} else {
			o1 = c.oprrr(p, p.As)
		}
		o1 |= uint32(p.From.Reg&31)<<5 | uint32(p.To.Reg&31)

	case 30: /* movT R,L(R) -> strT */
		// If offset L fits in a 12 bit unsigned immediate:
		//	add $L, R, Rtmp  or  sub $L, R, Rtmp
		//	str R, (Rtmp)
		// Otherwise, if offset L can be split into hi+lo, and both fit into instructions:
		//	add $hi, R, Rtmp
		//	str R, lo(Rtmp)
		// Otherwise, use constant pool:
		//	mov $L, Rtmp (from constant pool)
		//	str R, (R+Rtmp)
		s := movesize(o.as)
		if s < 0 {
			c.ctxt.Diag("unexpected long move, op %v tab %v\n%v", p.As, o.as, p)
		}

		r := p.To.Reg
		if r == obj.REG_NONE {
			r = o.param
		}

		v := c.regoff(&p.To)
		if v >= -256 && v <= 256 {
			c.ctxt.Diag("%v: bad type for offset %d (should be 9 bit signed immediate store)", p, v)
		}
		if v >= 0 && v <= 4095 && v&((1<<int32(s))-1) == 0 {
			c.ctxt.Diag("%v: bad type for offset %d (should be 12 bit unsigned immediate store)", p, v)
		}

		// Handle smaller unaligned and negative offsets via addition or subtraction.
		if v >= -4095 && v <= 4095 {
			o1 = c.oaddi12(p, v, REGTMP, int16(r))
			o2 = c.olsr12u(p, c.opstr(p, p.As), 0, REGTMP, p.From.Reg)
			break
		}

		hi, lo, err := splitImm24uScaled(v, s)
		if err != nil {
			goto storeusepool
		}
		if p.Pool != nil {
			c.ctxt.Diag("%v: unused constant in pool (%v)\n", p, v)
		}
		o1 = c.oaddi(p, AADD, hi, REGTMP, r)
		o2 = c.olsr12u(p, c.opstr(p, p.As), lo, REGTMP, p.From.Reg)
		break

	storeusepool:
		if p.Pool == nil {
			c.ctxt.Diag("%v: constant is not in pool", p)
		}
		if r == REGTMP || p.From.Reg == REGTMP {
			c.ctxt.Diag("REGTMP used in large offset store: %v", p)
		}
		o1 = c.omovlit(AMOVD, p, &p.To, REGTMP)
		o2 = c.olsxrr(p, int32(c.opstrr(p, p.As, false)), int(p.From.Reg), int(r), REGTMP)

	case 31: /* movT L(R), R -> ldrT */
		// If offset L fits in a 12 bit unsigned immediate:
		//	add $L, R, Rtmp  or  sub $L, R, Rtmp
		//	ldr R, (Rtmp)
		// Otherwise, if offset L can be split into hi+lo, and both fit into instructions:
		//	add $hi, R, Rtmp
		//	ldr lo(Rtmp), R
		// Otherwise, use constant pool:
		//	mov $L, Rtmp (from constant pool)
		//	ldr (R+Rtmp), R
		s := movesize(o.as)
		if s < 0 {
			c.ctxt.Diag("unexpected long move, op %v tab %v\n%v", p.As, o.as, p)
		}

		r := p.From.Reg
		if r == obj.REG_NONE {
			r = o.param
		}

		v := c.regoff(&p.From)
		if v >= -256 && v <= 256 {
			c.ctxt.Diag("%v: bad type for offset %d (should be 9 bit signed immediate load)", p, v)
		}
		if v >= 0 && v <= 4095 && v&((1<<int32(s))-1) == 0 {
			c.ctxt.Diag("%v: bad type for offset %d (should be 12 bit unsigned immediate load)", p, v)
		}

		// Handle smaller unaligned and negative offsets via addition or subtraction.
		if v >= -4095 && v <= 4095 {
			o1 = c.oaddi12(p, v, REGTMP, int16(r))
			o2 = c.olsr12u(p, c.opldr(p, p.As), 0, REGTMP, p.To.Reg)
			break
		}

		hi, lo, err := splitImm24uScaled(v, s)
		if err != nil {
			goto loadusepool
		}
		if p.Pool != nil {
			c.ctxt.Diag("%v: unused constant in pool (%v)\n", p, v)
		}
		o1 = c.oaddi(p, AADD, hi, REGTMP, r)
		o2 = c.olsr12u(p, c.opldr(p, p.As), lo, REGTMP, p.To.Reg)
		break

	loadusepool:
		if p.Pool == nil {
			c.ctxt.Diag("%v: constant is not in pool", p)
		}
		if r == REGTMP || p.From.Reg == REGTMP {
			c.ctxt.Diag("REGTMP used in large offset load: %v", p)
		}
		o1 = c.omovlit(AMOVD, p, &p.From, REGTMP)
		o2 = c.olsxrr(p, int32(c.opldrr(p, p.As, false)), int(p.To.Reg), int(r), REGTMP)

	case 32: /* mov $con, R -> movz/movn */
		o1 = c.omovconst(p.As, p, &p.From, int(p.To.Reg))

	case 33: /* movk $uimm16 << pos */
		o1 = c.opirr(p, p.As)

		d := p.From.Offset
		if d == 0 {
			c.ctxt.Diag("zero shifts cannot be handled correctly: %v", p)
		}
		s := movcon(d)
		if s < 0 || s >= 4 {
			c.ctxt.Diag("bad constant for MOVK: %#x\n%v", uint64(d), p)
		}
		if (o1&S64) == 0 && s >= 2 {
			c.ctxt.Diag("illegal bit position\n%v", p)
		}
		if ((uint64(d) >> uint(s*16)) >> 16) != 0 {
			c.ctxt.Diag("requires uimm16\n%v", p)
		}
		rt := int(p.To.Reg)

		o1 |= uint32((((d >> uint(s*16)) & 0xFFFF) << 5) | int64((uint32(s)&3)<<21) | int64(rt&31))

	case 34: /* mov $lacon,R */
		o1 = c.omovlit(AMOVD, p, &p.From, REGTMP)
		rt, r, rf := p.To.Reg, p.From.Reg, int16(REGTMP)
		if r == obj.REG_NONE {
			r = o.param
		}
		o2 = c.opxrrr(p, AADD, rt, r, rf, false)
		o2 |= LSL0_64

	case 35: /* mov SPR,R -> mrs */
		o1 = c.oprrr(p, AMRS)

		// SysRegEnc function returns the system register encoding and accessFlags.
		_, v, accessFlags := SysRegEnc(p.From.Reg)
		if v == 0 {
			c.ctxt.Diag("illegal system register:\n%v", p)
		}
		if (o1 & (v &^ (3 << 19))) != 0 {
			c.ctxt.Diag("MRS register value overlap\n%v", p)
		}
		if accessFlags&SR_READ == 0 {
			c.ctxt.Diag("system register is not readable: %v", p)
		}

		o1 |= v
		o1 |= uint32(p.To.Reg & 31)

	case 36: /* mov R,SPR */
		o1 = c.oprrr(p, AMSR)

		// SysRegEnc function returns the system register encoding and accessFlags.
		_, v, accessFlags := SysRegEnc(p.To.Reg)
		if v == 0 {
			c.ctxt.Diag("illegal system register:\n%v", p)
		}
		if (o1 & (v &^ (3 << 19))) != 0 {
			c.ctxt.Diag("MSR register value overlap\n%v", p)
		}
		if accessFlags&SR_WRITE == 0 {
			c.ctxt.Diag("system register is not writable: %v", p)
		}

		o1 |= v
		o1 |= uint32(p.From.Reg & 31)

	case 37: /* mov $con,PSTATEfield -> MSR [immediate] */
		if (uint64(p.From.Offset) &^ uint64(0xF)) != 0 {
			c.ctxt.Diag("illegal immediate for PSTATE field\n%v", p)
		}
		o1 = c.opirr(p, AMSR)
		o1 |= uint32((p.From.Offset & 0xF) << 8) /* Crm */
		v := uint32(0)
		// PSTATEfield can be special registers and special operands.
		if p.To.Type == obj.TYPE_REG && p.To.Reg == REG_SPSel {
			v = 0<<16 | 4<<12 | 5<<5
		} else if p.To.Type == obj.TYPE_REG && p.To.Reg == REG_DIT {
			// op1 = 011 (3) op2 = 010 (2)
			v = 3<<16 | 2<<5
		} else if p.To.Type == obj.TYPE_SPECIAL {
			opd := SpecialOperand(p.To.Offset)
			for _, pf := range pstatefield {
				if pf.opd == opd {
					v = pf.enc
					break
				}
			}
		}

		if v == 0 {
			c.ctxt.Diag("illegal PSTATE field for immediate move\n%v", p)
		}
		o1 |= v

	case 38: /* clrex [$imm] */
		o1 = c.opimm(p, p.As)

		if p.To.Type == obj.TYPE_NONE {
			o1 |= 0xF << 8
		} else {
			o1 |= uint32((p.To.Offset & 0xF) << 8)
		}

	case 39: /* cbz R, rel */
		o1 = c.opirr(p, p.As)

		o1 |= uint32(p.From.Reg & 31)
		o1 |= uint32(c.brdist(p, 0, 19, 2) << 5)

	case 40: /* tbz */
		o1 = c.opirr(p, p.As)

		v := int32(p.From.Offset)
		if v < 0 || v > 63 {
			c.ctxt.Diag("illegal bit number\n%v", p)
		}
		o1 |= ((uint32(v) & 0x20) << (31 - 5)) | ((uint32(v) & 0x1F) << 19)
		o1 |= uint32(c.brdist(p, 0, 14, 2) << 5)
		o1 |= uint32(p.Reg & 31)

	case 41: /* eret, nop, others with no operands */
		o1 = c.op0(p, p.As)

	case 42: /* bfm R,r,s,R */
		o1 = c.opbfm(p, p.As, p.From.Offset, p.GetFrom3().Offset, p.Reg, p.To.Reg)

	case 43: /* bfm aliases */
		rt, rf := p.To.Reg, p.Reg
		if rf == obj.REG_NONE {
			rf = rt
		}
		r, s := p.From.Offset, p.GetFrom3().Offset
		switch p.As {
		case ABFI:
			if r != 0 {
				r = 64 - r
			}
			o1 = c.opbfm(p, ABFM, r, s-1, rf, rt)

		case ABFIW:
			if r != 0 {
				r = 32 - r
			}
			o1 = c.opbfm(p, ABFMW, r, s-1, rf, rt)

		case ABFXIL:
			o1 = c.opbfm(p, ABFM, r, r+s-1, rf, rt)

		case ABFXILW:
			o1 = c.opbfm(p, ABFMW, r, r+s-1, rf, rt)

		case ASBFIZ:
			if r != 0 {
				r = 64 - r
			}
			o1 = c.opbfm(p, ASBFM, r, s-1, rf, rt)

		case ASBFIZW:
			if r != 0 {
				r = 32 - r
			}
			o1 = c.opbfm(p, ASBFMW, r, s-1, rf, rt)

		case ASBFX:
			o1 = c.opbfm(p, ASBFM, r, r+s-1, rf, rt)

		case ASBFXW:
			o1 = c.opbfm(p, ASBFMW, r, r+s-1, rf, rt)

		case AUBFIZ:
			if r != 0 {
				r = 64 - r
			}
			o1 = c.opbfm(p, AUBFM, r, s-1, rf, rt)

		case AUBFIZW:
			if r != 0 {
				r = 32 - r
			}
			o1 = c.opbfm(p, AUBFMW, r, s-1, rf, rt)

		case AUBFX:
			o1 = c.opbfm(p, AUBFM, r, r+s-1, rf, rt)

		case AUBFXW:
			o1 = c.opbfm(p, AUBFMW, r, r+s-1, rf, rt)

		default:
			c.ctxt.Diag("bad bfm alias\n%v", p)
			break
		}

	case 44: /* extr $b, Rn, Rm, Rd */
		o1 = c.opextr(p, p.As, p.From.Offset, p.GetFrom3().Reg, p.Reg, p.To.Reg)

	case 45: /* sxt/uxt[bhw] R,R; movT R,R -> sxtT R,R */
		as := p.As
		rt, rf := p.To.Reg, p.From.Reg
		if rf == REGZERO {
			as = AMOVWU /* clearer in disassembly */
		}
		switch as {
		case AMOVB, ASXTB:
			o1 = c.opbfm(p, ASBFM, 0, 7, rf, rt)

		case AMOVH, ASXTH:
			o1 = c.opbfm(p, ASBFM, 0, 15, rf, rt)

		case AMOVW, ASXTW:
			o1 = c.opbfm(p, ASBFM, 0, 31, rf, rt)

		case AMOVBU, AUXTB:
			o1 = c.opbfm(p, AUBFM, 0, 7, rf, rt)

		case AMOVHU, AUXTH:
			o1 = c.opbfm(p, AUBFM, 0, 15, rf, rt)

		case AMOVWU:
			o1 = c.oprrr(p, as) | (uint32(rf&31) << 16) | (REGZERO & 31 << 5) | uint32(rt&31)

		case AUXTW:
			o1 = c.opbfm(p, AUBFM, 0, 31, rf, rt)

		case ASXTBW:
			o1 = c.opbfm(p, ASBFMW, 0, 7, rf, rt)

		case ASXTHW:
			o1 = c.opbfm(p, ASBFMW, 0, 15, rf, rt)

		case AUXTBW:
			o1 = c.opbfm(p, AUBFMW, 0, 7, rf, rt)

		case AUXTHW:
			o1 = c.opbfm(p, AUBFMW, 0, 15, rf, rt)

		default:
			c.ctxt.Diag("bad sxt %v", as)
			break
		}

	case 46: /* cls */
		o1 = c.opbit(p, p.As)

		o1 |= uint32(p.From.Reg&31) << 5
		o1 |= uint32(p.To.Reg & 31)

	case 47: // SWPx/LDADDx/LDCLRx/LDEORx/LDORx/CASx Rs, (Rb), Rt
		rs := p.From.Reg
		rt := p.RegTo2
		rb := p.To.Reg

		// rt can't be sp.
		if rt == REG_RSP {
			c.ctxt.Diag("illegal destination register: %v\n", p)
		}

		o1 = atomicLDADD[p.As] | atomicSWP[p.As]
		o1 |= uint32(rs&31)<<16 | uint32(rb&31)<<5 | uint32(rt&31)

	case 48: /* ADD $C_ADDCON2, Rm, Rd */
		// NOTE: this case does not use REGTMP. If it ever does,
		// remove the NOTUSETMP flag in optab.
		op := c.opirr(p, p.As)
		if op&Sbit != 0 {
			c.ctxt.Diag("can not break addition/subtraction when S bit is set", p)
		}
		rt, r := p.To.Reg, p.Reg
		if r == obj.REG_NONE {
			r = rt
		}
		o1 = c.oaddi(p, p.As, c.regoff(&p.From)&0x000fff, rt, r)
		o2 = c.oaddi(p, p.As, c.regoff(&p.From)&0xfff000, rt, rt)

	case 49: /* op Vm.<T>, Vn, Vd */
		o1 = c.oprrr(p, p.As)
		cf := c.aclass(&p.From)
		af := (p.From.Reg >> 5) & 15
		sz := ARNG_4S
		if p.As == ASHA512H || p.As == ASHA512H2 {
			sz = ARNG_2D
		}
		if cf == C_ARNG && af != int16(sz) {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		o1 |= uint32(p.From.Reg&31)<<16 | uint32(p.Reg&31)<<5 | uint32(p.To.Reg&31)

	case 50: /* sys/sysl */
		o1 = c.opirr(p, p.As)

		if (p.From.Offset &^ int64(SYSARG4(0x7, 0xF, 0xF, 0x7))) != 0 {
			c.ctxt.Diag("illegal SYS argument\n%v", p)
		}
		o1 |= uint32(p.From.Offset)
		if p.To.Type == obj.TYPE_REG {
			o1 |= uint32(p.To.Reg & 31)
		} else {
			o1 |= 0x1F
		}

	case 51: /* dmb */
		o1 = c.opirr(p, p.As)

		if p.From.Type == obj.TYPE_CONST {
			o1 |= uint32((p.From.Offset & 0xF) << 8)
		}

	case 52: /* hint */
		o1 = c.opirr(p, p.As)

		o1 |= uint32((p.From.Offset & 0x7F) << 5)

	case 53: /* and/or/eor/bic/tst/... $bitcon, Rn, Rd */
		a := p.As
		rt := int(p.To.Reg)
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		r := int(p.Reg)
		if r == obj.REG_NONE {
			r = rt
		}
		if r == REG_RSP {
			c.ctxt.Diag("illegal source register: %v", p)
			break
		}
		mode := 64
		v := uint64(p.From.Offset)
		switch p.As {
		case AANDW, AORRW, AEORW, AANDSW, ATSTW:
			mode = 32
		case ABIC, AORN, AEON, ABICS:
			v = ^v
		case ABICW, AORNW, AEONW, ABICSW:
			v = ^v
			mode = 32
		}
		o1 = c.opirr(p, a)
		o1 |= bitconEncode(v, mode) | uint32(r&31)<<5 | uint32(rt&31)

	case 54: /* floating point arith */
		o1 = c.oprrr(p, p.As)
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		r := int(p.Reg)
		if (o1&(0x1F<<24)) == (0x1E<<24) && (o1&(1<<11)) == 0 { /* monadic */
			r = rf
			rf = 0
		} else if r == obj.REG_NONE {
			r = rt
		}
		o1 |= (uint32(rf&31) << 16) | (uint32(r&31) << 5) | uint32(rt&31)

	case 55: /* floating-point constant */
		var rf int
		o1 = 0xf<<25 | 1<<21 | 1<<12
		rf = c.chipfloat7(p.From.Val.(float64))
		if rf < 0 {
			c.ctxt.Diag("invalid floating-point immediate\n%v", p)
		}
		if p.As == AFMOVD {
			o1 |= 1 << 22
		}
		o1 |= (uint32(rf&0xff) << 13) | uint32(p.To.Reg&31)

	case 56: /* floating point compare */
		o1 = c.oprrr(p, p.As)

		var rf int
		if p.From.Type == obj.TYPE_FCONST {
			o1 |= 8 /* zero */
			rf = 0
		} else {
			rf = int(p.From.Reg)
		}
		rt := int(p.Reg)
		o1 |= uint32(rf&31)<<16 | uint32(rt&31)<<5

	case 57: /* floating point conditional compare */
		o1 = c.oprrr(p, p.As)

		cond := SpecialOperand(p.From.Offset)
		if cond < SPOP_EQ || cond > SPOP_NV {
			c.ctxt.Diag("invalid condition\n%v", p)
		} else {
			cond -= SPOP_EQ
		}

		nzcv := int(p.To.Offset)
		if nzcv&^0xF != 0 {
			c.ctxt.Diag("implausible condition\n%v", p)
		}
		rf := int(p.Reg)
		if p.GetFrom3() == nil || p.GetFrom3().Reg < REG_F0 || p.GetFrom3().Reg > REG_F31 {
			c.ctxt.Diag("illegal FCCMP\n%v", p)
			break
		}
		rt := int(p.GetFrom3().Reg)
		o1 |= uint32(rf&31)<<16 | uint32(cond&15)<<12 | uint32(rt&31)<<5 | uint32(nzcv)

	case 58: /* ldar/ldarb/ldarh/ldaxp/ldxp/ldaxr/ldxr */
		o1 = c.opload(p, p.As)

		if p.As == ALDAPR || p.As == ALDAPRW || p.As == ALDAPRB || p.As == ALDAPRH {
			o1 |= uint32(p.From.Reg&31) << 5
			o1 |= uint32(p.To.Reg & 31)
			break
		}

		o1 |= 0x1F << 16
		o1 |= uint32(p.From.Reg&31) << 5
		if p.As == ALDXP || p.As == ALDXPW || p.As == ALDAXP || p.As == ALDAXPW {
			if int(p.To.Reg) == int(p.To.Offset) {
				c.ctxt.Diag("constrained unpredictable behavior: %v", p)
			}
			o1 |= uint32(p.To.Offset&31) << 10
		} else {
			o1 |= 0x1F << 10
		}
		o1 |= uint32(p.To.Reg & 31)

	case 59: /* stxr/stlxr/stxp/stlxp */
		s := p.RegTo2
		n := p.To.Reg
		t := p.From.Reg
		if isSTLXRop(p.As) {
			if s == t || (s == n && n != REGSP) {
				c.ctxt.Diag("constrained unpredictable behavior: %v", p)
			}
		} else if isSTXPop(p.As) {
			t2 := int16(p.From.Offset)
			if (s == t || s == t2) || (s == n && n != REGSP) {
				c.ctxt.Diag("constrained unpredictable behavior: %v", p)
			}
		}
		if s == REG_RSP {
			c.ctxt.Diag("illegal destination register: %v\n", p)
		}
		o1 = c.opstore(p, p.As)

		if p.RegTo2 != obj.REG_NONE {
			o1 |= uint32(p.RegTo2&31) << 16
		} else {
			o1 |= 0x1F << 16
		}
		if isSTXPop(p.As) {
			o1 |= uint32(p.From.Offset&31) << 10
		}
		o1 |= uint32(p.To.Reg&31)<<5 | uint32(p.From.Reg&31)

	case 60: /* adrp label,r */
		d := c.brdist(p, 12, 21, 0)

		o1 = ADR(1, uint32(d), uint32(p.To.Reg))

	case 61: /* adr label, r */
		d := c.brdist(p, 0, 21, 0)

		o1 = ADR(0, uint32(d), uint32(p.To.Reg))

	case 62: /* op $movcon, [R], R -> mov $movcon, REGTMP + op REGTMP, [R], R */
		if p.Reg == REGTMP {
			c.ctxt.Diag("cannot use REGTMP as source: %v\n", p)
		}
		if p.To.Reg == REG_RSP && isADDSop(p.As) {
			c.ctxt.Diag("illegal destination register: %v\n", p)
		}
		lsl0 := LSL0_64
		if isADDWop(p.As) || isANDWop(p.As) {
			o1 = c.omovconst(AMOVW, p, &p.From, REGTMP)
			lsl0 = LSL0_32
		} else {
			o1 = c.omovconst(AMOVD, p, &p.From, REGTMP)
		}

		rt, r, rf := p.To.Reg, p.Reg, int16(REGTMP)
		if p.To.Type == obj.TYPE_NONE {
			rt = REGZERO
		}
		if r == obj.REG_NONE {
			r = rt
		}
		if rt == REGSP || r == REGSP {
			o2 = c.opxrrr(p, p.As, rt, r, rf, false)
			o2 |= uint32(lsl0)
		} else {
			o2 = c.oprrr(p, p.As)
			o2 |= uint32(rf&31) << 16 /* shift is 0 */
			o2 |= uint32(r&31) << 5
			o2 |= uint32(rt & 31)
		}

	case 63: /* op Vm.<t>, Vn.<T>, Vd.<T> */
		o1 |= c.oprrr(p, p.As)
		af := (p.From.Reg >> 5) & 15
		at := (p.To.Reg >> 5) & 15
		ar := (p.Reg >> 5) & 15
		sz := ARNG_4S
		if p.As == ASHA512SU1 {
			sz = ARNG_2D
		}
		if af != at || af != ar || af != int16(sz) {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		o1 |= uint32(p.From.Reg&31)<<16 | uint32(p.Reg&31)<<5 | uint32(p.To.Reg&31)

	/* reloc ops */
	case 64: /* movT R,addr -> adrp + movT R, (REGTMP) */
		if p.From.Reg == REGTMP {
			c.ctxt.Diag("cannot use REGTMP as source: %v\n", p)
		}
		o1 = ADR(1, 0, REGTMP)
		var typ objabi.RelocType
		// For unaligned access, fall back to adrp + add + movT R, (REGTMP).
		if o.size(c.ctxt, p) != 8 {
			o2 = c.opirr(p, AADD) | REGTMP&31<<5 | REGTMP&31
			o3 = c.olsr12u(p, c.opstr(p, p.As), 0, REGTMP, p.From.Reg)
			typ = objabi.R_ADDRARM64
		} else {
			o2 = c.olsr12u(p, c.opstr(p, p.As), 0, REGTMP, p.From.Reg)
			typ = c.addrRelocType(p)
		}
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: typ,
			Off:  int32(c.pc),
			Siz:  8,
			Sym:  p.To.Sym,
			Add:  p.To.Offset,
		})

	case 65: /* movT addr,R -> adrp + movT (REGTMP), R */
		o1 = ADR(1, 0, REGTMP)
		var typ objabi.RelocType
		// For unaligned access, fall back to adrp + add + movT (REGTMP), R.
		if o.size(c.ctxt, p) != 8 {
			o2 = c.opirr(p, AADD) | REGTMP&31<<5 | REGTMP&31
			o3 = c.olsr12u(p, c.opldr(p, p.As), 0, REGTMP, p.To.Reg)
			typ = objabi.R_ADDRARM64
		} else {
			o2 = c.olsr12u(p, c.opldr(p, p.As), 0, REGTMP, p.To.Reg)
			typ = c.addrRelocType(p)
		}
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: typ,
			Off:  int32(c.pc),
			Siz:  8,
			Sym:  p.From.Sym,
			Add:  p.From.Offset,
		})

	case 66: /* ldp O(R)!, (r1, r2); ldp (R)O!, (r1, r2) */
		rf, rt1, rt2 := p.From.Reg, p.To.Reg, int16(p.To.Offset)
		if rf == obj.REG_NONE {
			rf = o.param
		}
		if rf == obj.REG_NONE {
			c.ctxt.Diag("invalid ldp source: %v\n", p)
		}
		v := c.regoff(&p.From)
		o1 = c.opldpstp(p, o, v, rf, rt1, rt2, 1)

	case 67: /* stp (r1, r2), O(R)!; stp (r1, r2), (R)O! */
		rt, rf1, rf2 := p.To.Reg, p.From.Reg, int16(p.From.Offset)
		if rt == obj.REG_NONE {
			rt = o.param
		}
		if rt == obj.REG_NONE {
			c.ctxt.Diag("invalid stp destination: %v\n", p)
		}
		v := c.regoff(&p.To)
		o1 = c.opldpstp(p, o, v, rt, rf1, rf2, 0)

	case 68: /* movT $vconaddr(SB), reg -> adrp + add + reloc */
		// NOTE: this case does not use REGTMP. If it ever does,
		// remove the NOTUSETMP flag in optab.
		if p.As == AMOVW {
			c.ctxt.Diag("invalid load of 32-bit address: %v", p)
		}
		o1 = ADR(1, 0, uint32(p.To.Reg))
		o2 = c.opirr(p, AADD) | uint32(p.To.Reg&31)<<5 | uint32(p.To.Reg&31)
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: objabi.R_ADDRARM64,
			Off:  int32(c.pc),
			Siz:  8,
			Sym:  p.From.Sym,
			Add:  p.From.Offset,
		})

	case 69: /* LE model movd $tlsvar, reg -> movz reg, 0 + reloc */
		o1 = c.opirr(p, AMOVZ)
		o1 |= uint32(p.To.Reg & 31)
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: objabi.R_ARM64_TLS_LE,
			Off:  int32(c.pc),
			Siz:  4,
			Sym:  p.From.Sym,
		})
		if p.From.Offset != 0 {
			c.ctxt.Diag("invalid offset on MOVW $tlsvar")
		}

	case 70: /* IE model movd $tlsvar, reg -> adrp REGTMP, 0; ldr reg, [REGTMP, #0] + relocs */
		o1 = ADR(1, 0, REGTMP)
		o2 = c.olsr12u(p, c.opldr(p, AMOVD), 0, REGTMP, p.To.Reg)
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: objabi.R_ARM64_TLS_IE,
			Off:  int32(c.pc),
			Siz:  8,
			Sym:  p.From.Sym,
		})
		if p.From.Offset != 0 {
			c.ctxt.Diag("invalid offset on MOVW $tlsvar")
		}

	case 71: /* movd sym@GOT, reg -> adrp REGTMP, #0; ldr reg, [REGTMP, #0] + relocs */
		o1 = ADR(1, 0, REGTMP)
		o2 = c.olsr12u(p, c.opldr(p, AMOVD), 0, REGTMP, p.To.Reg)
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: objabi.R_ARM64_GOTPCREL,
			Off:  int32(c.pc),
			Siz:  8,
			Sym:  p.From.Sym,
		})

	case 72: /* vaddp/vand/vcmeq/vorr/vadd/veor/vfmla/vfmls/vbit/vbsl/vcmtst/vsub/vbif/vuzip1/vuzip2/vrax1 Vm.<T>, Vn.<T>, Vd.<T> */
		af := int((p.From.Reg >> 5) & 15)
		af3 := int((p.Reg >> 5) & 15)
		at := int((p.To.Reg >> 5) & 15)
		if af != af3 || af != at {
			c.ctxt.Diag("operand mismatch: %v", p)
			break
		}
		o1 = c.oprrr(p, p.As)
		rf := int((p.From.Reg) & 31)
		rt := int((p.To.Reg) & 31)
		r := int((p.Reg) & 31)

		Q := 0
		size := 0
		switch af {
		case ARNG_16B:
			Q = 1
			size = 0
		case ARNG_2D:
			Q = 1
			size = 3
		case ARNG_2S:
			Q = 0
			size = 2
		case ARNG_4H:
			Q = 0
			size = 1
		case ARNG_4S:
			Q = 1
			size = 2
		case ARNG_8B:
			Q = 0
			size = 0
		case ARNG_8H:
			Q = 1
			size = 1
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		switch p.As {
		case AVORR, AVAND, AVEOR, AVBIT, AVBSL, AVBIF:
			if af != ARNG_16B && af != ARNG_8B {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		case AVFMLA, AVFMLS:
			if af != ARNG_2D && af != ARNG_2S && af != ARNG_4S {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		case AVUMAX, AVUMIN:
			if af == ARNG_2D {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		}
		switch p.As {
		case AVAND, AVEOR:
			size = 0
		case AVBSL:
			size = 1
		case AVORR, AVBIT, AVBIF:
			size = 2
		case AVFMLA, AVFMLS:
			if af == ARNG_2D {
				size = 1
			} else {
				size = 0
			}
		case AVRAX1:
			if af != ARNG_2D {
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
			size = 0
			Q = 0
		}

		o1 |= (uint32(Q&1) << 30) | (uint32(size&3) << 22) | (uint32(rf&31) << 16) | (uint32(r&31) << 5) | uint32(rt&31)

	case 73: /* vmov V.<T>[index], R */
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		imm5 := 0
		o1 = 7<<25 | 0xf<<10
		index := int(p.From.Index)
		switch (p.From.Reg >> 5) & 15 {
		case ARNG_B:
			c.checkindex(p, index, 15)
			imm5 |= 1
			imm5 |= index << 1
		case ARNG_H:
			c.checkindex(p, index, 7)
			imm5 |= 2
			imm5 |= index << 2
		case ARNG_S:
			c.checkindex(p, index, 3)
			imm5 |= 4
			imm5 |= index << 3
		case ARNG_D:
			c.checkindex(p, index, 1)
			imm5 |= 8
			imm5 |= index << 4
			o1 |= 1 << 30
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		o1 |= (uint32(imm5&0x1f) << 16) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 74:
		//	add $O, R, Rtmp or sub $O, R, Rtmp
		//	ldp (Rtmp), (R1, R2)
		rf, rt1, rt2 := p.From.Reg, p.To.Reg, int16(p.To.Offset)
		if rf == obj.REG_NONE {
			rf = o.param
		}
		if rf == obj.REG_NONE {
			c.ctxt.Diag("invalid ldp source: %v", p)
		}
		v := c.regoff(&p.From)
		o1 = c.oaddi12(p, v, REGTMP, rf)
		o2 = c.opldpstp(p, o, 0, REGTMP, rt1, rt2, 1)

	case 75:
		// If offset L fits in a 24 bit unsigned immediate:
		//	add $lo, R, Rtmp
		//	add $hi, Rtmp, Rtmp
		//	ldr (Rtmp), R
		// Otherwise, use constant pool:
		//	mov $L, Rtmp (from constant pool)
		//	add Rtmp, R, Rtmp
		//	ldp (Rtmp), (R1, R2)
		rf, rt1, rt2 := p.From.Reg, p.To.Reg, int16(p.To.Offset)
		if rf == REGTMP {
			c.ctxt.Diag("REGTMP used in large offset load: %v", p)
		}
		if rf == obj.REG_NONE {
			rf = o.param
		}
		if rf == obj.REG_NONE {
			c.ctxt.Diag("invalid ldp source: %v", p)
		}

		v := c.regoff(&p.From)
		if v >= -4095 && v <= 4095 {
			c.ctxt.Diag("%v: bad type for offset %d (should be add/sub+ldp)", p, v)
		}

		hi, lo, err := splitImm24uScaled(v, 0)
		if err != nil {
			goto loadpairusepool
		}
		if p.Pool != nil {
			c.ctxt.Diag("%v: unused constant in pool (%v)\n", p, v)
		}
		o1 = c.oaddi(p, AADD, lo, REGTMP, int16(rf))
		o2 = c.oaddi(p, AADD, hi, REGTMP, REGTMP)
		o3 = c.opldpstp(p, o, 0, REGTMP, rt1, rt2, 1)
		break

	loadpairusepool:
		if p.Pool == nil {
			c.ctxt.Diag("%v: constant is not in pool", p)
		}
		if rf == REGTMP || p.From.Reg == REGTMP {
			c.ctxt.Diag("REGTMP used in large offset load: %v", p)
		}
		o1 = c.omovlit(AMOVD, p, &p.From, REGTMP)
		o2 = c.opxrrr(p, AADD, REGTMP, rf, REGTMP, false)
		o3 = c.opldpstp(p, o, 0, REGTMP, rt1, rt2, 1)

	case 76:
		//	add $O, R, Rtmp or sub $O, R, Rtmp
		//	stp (R1, R2), (Rtmp)
		rt, rf1, rf2 := p.To.Reg, p.From.Reg, int16(p.From.Offset)
		if rf1 == REGTMP || rf2 == REGTMP {
			c.ctxt.Diag("cannot use REGTMP as source: %v", p)
		}
		if rt == obj.REG_NONE {
			rt = o.param
		}
		if rt == obj.REG_NONE {
			c.ctxt.Diag("invalid stp destination: %v", p)
		}
		v := c.regoff(&p.To)
		o1 = c.oaddi12(p, v, REGTMP, rt)
		o2 = c.opldpstp(p, o, 0, REGTMP, rf1, rf2, 0)

	case 77:
		// If offset L fits in a 24 bit unsigned immediate:
		//	add $lo, R, Rtmp
		//	add $hi, Rtmp, Rtmp
		//	stp (R1, R2), (Rtmp)
		// Otherwise, use constant pool:
		//	mov $L, Rtmp (from constant pool)
		//	add Rtmp, R, Rtmp
		//	stp (R1, R2), (Rtmp)
		rt, rf1, rf2 := p.To.Reg, p.From.Reg, int16(p.From.Offset)
		if rt == REGTMP || rf1 == REGTMP || rf2 == REGTMP {
			c.ctxt.Diag("REGTMP used in large offset store: %v", p)
		}
		if rt == obj.REG_NONE {
			rt = o.param
		}
		if rt == obj.REG_NONE {
			c.ctxt.Diag("invalid stp destination: %v", p)
		}

		v := c.regoff(&p.To)
		if v >= -4095 && v <= 4095 {
			c.ctxt.Diag("%v: bad type for offset %d (should be add/sub+stp)", p, v)
		}

		hi, lo, err := splitImm24uScaled(v, 0)
		if err != nil {
			goto storepairusepool
		}
		if p.Pool != nil {
			c.ctxt.Diag("%v: unused constant in pool (%v)\n", p, v)
		}
		o1 = c.oaddi(p, AADD, lo, REGTMP, int16(rt))
		o2 = c.oaddi(p, AADD, hi, REGTMP, REGTMP)
		o3 = c.opldpstp(p, o, 0, REGTMP, rf1, rf2, 0)
		break

	storepairusepool:
		if p.Pool == nil {
			c.ctxt.Diag("%v: constant is not in pool", p)
		}
		if rt == REGTMP || p.From.Reg == REGTMP {
			c.ctxt.Diag("REGTMP used in large offset store: %v", p)
		}
		o1 = c.omovlit(AMOVD, p, &p.To, REGTMP)
		o2 = c.opxrrr(p, AADD, REGTMP, rt, REGTMP, false)
		o3 = c.opldpstp(p, o, 0, REGTMP, rf1, rf2, 0)

	case 78: /* vmov R, V.<T>[index] */
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		imm5 := 0
		o1 = 1<<30 | 7<<25 | 7<<10
		index := int(p.To.Index)
		switch (p.To.Reg >> 5) & 15 {
		case ARNG_B:
			c.checkindex(p, index, 15)
			imm5 |= 1
			imm5 |= index << 1
		case ARNG_H:
			c.checkindex(p, index, 7)
			imm5 |= 2
			imm5 |= index << 2
		case ARNG_S:
			c.checkindex(p, index, 3)
			imm5 |= 4
			imm5 |= index << 3
		case ARNG_D:
			c.checkindex(p, index, 1)
			imm5 |= 8
			imm5 |= index << 4
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		o1 |= (uint32(imm5&0x1f) << 16) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 79: /* vdup Vn.<T>[index], Vd.<T> */
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		o1 = 7<<25 | 1<<10
		var imm5, Q int
		index := int(p.From.Index)
		switch (p.To.Reg >> 5) & 15 {
		case ARNG_16B:
			c.checkindex(p, index, 15)
			Q = 1
			imm5 = 1
			imm5 |= index << 1
		case ARNG_2D:
			c.checkindex(p, index, 1)
			Q = 1
			imm5 = 8
			imm5 |= index << 4
		case ARNG_2S:
			c.checkindex(p, index, 3)
			Q = 0
			imm5 = 4
			imm5 |= index << 3
		case ARNG_4H:
			c.checkindex(p, index, 7)
			Q = 0
			imm5 = 2
			imm5 |= index << 2
		case ARNG_4S:
			c.checkindex(p, index, 3)
			Q = 1
			imm5 = 4
			imm5 |= index << 3
		case ARNG_8B:
			c.checkindex(p, index, 15)
			Q = 0
			imm5 = 1
			imm5 |= index << 1
		case ARNG_8H:
			c.checkindex(p, index, 7)
			Q = 1
			imm5 = 2
			imm5 |= index << 2
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		o1 |= (uint32(Q&1) << 30) | (uint32(imm5&0x1f) << 16)
		o1 |= (uint32(rf&31) << 5) | uint32(rt&31)

	case 80: /* vmov/vdup V.<T>[index], Vn */
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		imm5 := 0
		index := int(p.From.Index)
		switch p.As {
		case AVMOV, AVDUP:
			o1 = 1<<30 | 15<<25 | 1<<10
			switch (p.From.Reg >> 5) & 15 {
			case ARNG_B:
				c.checkindex(p, index, 15)
				imm5 |= 1
				imm5 |= index << 1
			case ARNG_H:
				c.checkindex(p, index, 7)
				imm5 |= 2
				imm5 |= index << 2
			case ARNG_S:
				c.checkindex(p, index, 3)
				imm5 |= 4
				imm5 |= index << 3
			case ARNG_D:
				c.checkindex(p, index, 1)
				imm5 |= 8
				imm5 |= index << 4
			default:
				c.ctxt.Diag("invalid arrangement: %v", p)
			}
		default:
			c.ctxt.Diag("unsupported op %v", p.As)
		}
		o1 |= (uint32(imm5&0x1f) << 16) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 81: /* vld[1-4]|vld[1-4]r (Rn), [Vt1.<T>, Vt2.<T>, ...] */
		c.checkoffset(p, p.As)
		r := int(p.From.Reg)
		o1 = c.oprrr(p, p.As)
		if o.scond == C_XPOST {
			o1 |= 1 << 23
			if p.From.Index == 0 {
				// immediate offset variant
				o1 |= 0x1f << 16
			} else {
				// register offset variant
				if isRegShiftOrExt(&p.From) {
					c.ctxt.Diag("invalid extended register op: %v\n", p)
				}
				o1 |= uint32(p.From.Index&0x1f) << 16
			}
		}
		o1 |= uint32(p.To.Offset)
		// cmd/asm/internal/arch/arm64.go:ARM64RegisterListOffset
		// add opcode(bit 12-15) for vld1, mask it off if it's not vld1
		o1 = c.maskOpvldvst(p, o1)
		o1 |= uint32(r&31) << 5

	case 82: /* vmov/vdup Rn, Vd.<T> */
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		o1 = 7<<25 | 3<<10
		var imm5, Q uint32
		switch (p.To.Reg >> 5) & 15 {
		case ARNG_16B:
			Q = 1
			imm5 = 1
		case ARNG_2D:
			Q = 1
			imm5 = 8
		case ARNG_2S:
			Q = 0
			imm5 = 4
		case ARNG_4H:
			Q = 0
			imm5 = 2
		case ARNG_4S:
			Q = 1
			imm5 = 4
		case ARNG_8B:
			Q = 0
			imm5 = 1
		case ARNG_8H:
			Q = 1
			imm5 = 2
		default:
			c.ctxt.Diag("invalid arrangement: %v\n", p)
		}
		o1 |= (Q & 1 << 30) | (imm5 & 0x1f << 16)
		o1 |= (uint32(rf&31) << 5) | uint32(rt&31)

	case 83: /* vmov Vn.<T>, Vd.<T> */
		af := int((p.From.Reg >> 5) & 15)
		at := int((p.To.Reg >> 5) & 15)
		if af != at {
			c.ctxt.Diag("invalid arrangement: %v\n", p)
		}
		o1 = c.oprrr(p, p.As)
		rf := int((p.From.Reg) & 31)
		rt := int((p.To.Reg) & 31)

		var Q, size uint32
		switch af {
		case ARNG_8B:
			Q = 0
			size = 0
		case ARNG_16B:
			Q = 1
			size = 0
		case ARNG_4H:
			Q = 0
			size = 1
		case ARNG_8H:
			Q = 1
			size = 1
		case ARNG_2S:
			Q = 0
			size = 2
		case ARNG_4S:
			Q = 1
			size = 2
		default:
			c.ctxt.Diag("invalid arrangement: %v\n", p)
		}

		if (p.As == AVMOV || p.As == AVRBIT || p.As == AVCNT) && (af != ARNG_16B && af != ARNG_8B) {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		if p.As == AVREV32 && (af == ARNG_2S || af == ARNG_4S) {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		if p.As == AVREV16 && af != ARNG_8B && af != ARNG_16B {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		if p.As == AVMOV {
			o1 |= uint32(rf&31) << 16
		}

		if p.As == AVRBIT {
			size = 1
		}

		o1 |= (Q&1)<<30 | (size&3)<<22 | uint32(rf&31)<<5 | uint32(rt&31)

	case 84: /* vst[1-4] [Vt1.<T>, Vt2.<T>, ...], (Rn) */
		c.checkoffset(p, p.As)
		r := int(p.To.Reg)
		o1 = 3 << 26
		if o.scond == C_XPOST {
			o1 |= 1 << 23
			if p.To.Index == 0 {
				// immediate offset variant
				o1 |= 0x1f << 16
			} else {
				// register offset variant
				if isRegShiftOrExt(&p.To) {
					c.ctxt.Diag("invalid extended register: %v\n", p)
				}
				o1 |= uint32(p.To.Index&31) << 16
			}
		}
		o1 |= uint32(p.From.Offset)
		// cmd/asm/internal/arch/arm64.go:ARM64RegisterListOffset
		// add opcode(bit 12-15) for vst1, mask it off if it's not vst1
		o1 = c.maskOpvldvst(p, o1)
		o1 |= uint32(r&31) << 5

	case 85: /* vaddv/vuaddlv Vn.<T>, Vd*/
		af := int((p.From.Reg >> 5) & 15)
		o1 = c.oprrr(p, p.As)
		rf := int((p.From.Reg) & 31)
		rt := int((p.To.Reg) & 31)
		Q := 0
		size := 0
		switch af {
		case ARNG_8B:
			Q = 0
			size = 0
		case ARNG_16B:
			Q = 1
			size = 0
		case ARNG_4H:
			Q = 0
			size = 1
		case ARNG_8H:
			Q = 1
			size = 1
		case ARNG_4S:
			Q = 1
			size = 2
		default:
			c.ctxt.Diag("invalid arrangement: %v\n", p)
		}
		o1 |= (uint32(Q&1) << 30) | (uint32(size&3) << 22) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 86: /* vmovi $imm8, Vd.<T>*/
		at := int((p.To.Reg >> 5) & 15)
		r := int(p.From.Offset)
		if r > 255 || r < 0 {
			c.ctxt.Diag("immediate constant out of range: %v\n", p)
		}
		rt := int((p.To.Reg) & 31)
		Q := 0
		switch at {
		case ARNG_8B:
			Q = 0
		case ARNG_16B:
			Q = 1
		default:
			c.ctxt.Diag("invalid arrangement: %v\n", p)
		}
		o1 = 0xf<<24 | 0xe<<12 | 1<<10
		o1 |= (uint32(Q&1) << 30) | (uint32((r>>5)&7) << 16) | (uint32(r&0x1f) << 5) | uint32(rt&31)

	case 87: /* stp (r,r), addr(SB) -> adrp + add + stp */
		rf1, rf2 := p.From.Reg, int16(p.From.Offset)
		if rf1 == REGTMP || rf2 == REGTMP {
			c.ctxt.Diag("cannot use REGTMP as source: %v", p)
		}
		o1 = ADR(1, 0, REGTMP)
		o2 = c.opirr(p, AADD) | REGTMP&31<<5 | REGTMP&31
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: objabi.R_ADDRARM64,
			Off:  int32(c.pc),
			Siz:  8,
			Sym:  p.To.Sym,
			Add:  p.To.Offset,
		})
		o3 = c.opldpstp(p, o, 0, REGTMP, rf1, rf2, 0)

	case 88: /* ldp addr(SB), (r,r) -> adrp + add + ldp */
		rt1, rt2 := p.To.Reg, int16(p.To.Offset)
		o1 = ADR(1, 0, REGTMP)
		o2 = c.opirr(p, AADD) | REGTMP&31<<5 | REGTMP&31
		c.cursym.AddRel(c.ctxt, obj.Reloc{
			Type: objabi.R_ADDRARM64,
			Off:  int32(c.pc),
			Siz:  8,
			Sym:  p.From.Sym,
			Add:  p.From.Offset,
		})
		o3 = c.opldpstp(p, o, 0, REGTMP, rt1, rt2, 1)

	case 89: /* vadd/vsub Vm, Vn, Vd */
		switch p.As {
		case AVADD:
			o1 = 5<<28 | 7<<25 | 7<<21 | 1<<15 | 1<<10

		case AVSUB:
			o1 = 7<<28 | 7<<25 | 7<<21 | 1<<15 | 1<<10

		default:
			c.ctxt.Diag("bad opcode: %v\n", p)
			break
		}

		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		r := int(p.Reg)
		if r == obj.REG_NONE {
			r = rt
		}
		o1 |= (uint32(rf&31) << 16) | (uint32(r&31) << 5) | uint32(rt&31)

	// This is supposed to be something that stops execution.
	// It's not supposed to be reached, ever, but if it is, we'd
	// like to be able to tell how we got there. Assemble as
	// UDF which is guaranteed to raise the undefined instruction
	// exception.
	case 90:
		o1 = 0x0

	case 91: /* prfm imm(Rn), <prfop | $imm5> */
		imm := uint32(p.From.Offset)
		r := p.From.Reg
		var v uint32
		var ok bool
		if p.To.Type == obj.TYPE_CONST {
			v = uint32(p.To.Offset)
			ok = v <= 31
		} else {
			v, ok = prfopfield[SpecialOperand(p.To.Offset)]
		}
		if !ok {
			c.ctxt.Diag("illegal prefetch operation:\n%v", p)
		}

		o1 = c.opirr(p, p.As)
		o1 |= (uint32(r&31) << 5) | (uint32((imm>>3)&0xfff) << 10) | (uint32(v & 31))

	case 92: /* vmov Vn.<T>[index], Vd.<T>[index] */
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		imm4 := 0
		imm5 := 0
		o1 = 3<<29 | 7<<25 | 1<<10
		index1 := int(p.To.Index)
		index2 := int(p.From.Index)
		if ((p.To.Reg >> 5) & 15) != ((p.From.Reg >> 5) & 15) {
			c.ctxt.Diag("operand mismatch: %v", p)
		}
		switch (p.To.Reg >> 5) & 15 {
		case ARNG_B:
			c.checkindex(p, index1, 15)
			c.checkindex(p, index2, 15)
			imm5 |= 1
			imm5 |= index1 << 1
			imm4 |= index2
		case ARNG_H:
			c.checkindex(p, index1, 7)
			c.checkindex(p, index2, 7)
			imm5 |= 2
			imm5 |= index1 << 2
			imm4 |= index2 << 1
		case ARNG_S:
			c.checkindex(p, index1, 3)
			c.checkindex(p, index2, 3)
			imm5 |= 4
			imm5 |= index1 << 3
			imm4 |= index2 << 2
		case ARNG_D:
			c.checkindex(p, index1, 1)
			c.checkindex(p, index2, 1)
			imm5 |= 8
			imm5 |= index1 << 4
			imm4 |= index2 << 3
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		o1 |= (uint32(imm5&0x1f) << 16) | (uint32(imm4&0xf) << 11) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 93: /* vpmull{2} Vm.<Tb>, Vn.<Tb>, Vd.<Ta> */
		af := uint8((p.From.Reg >> 5) & 15)
		at := uint8((p.To.Reg >> 5) & 15)
		a := uint8((p.Reg >> 5) & 15)
		if af != a {
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		var Q, size uint32
		if p.As == AVPMULL2 {
			Q = 1
		}
		switch pack(Q, at, af) {
		case pack(0, ARNG_8H, ARNG_8B), pack(1, ARNG_8H, ARNG_16B):
			size = 0
		case pack(0, ARNG_1Q, ARNG_1D), pack(1, ARNG_1Q, ARNG_2D):
			size = 3
		default:
			c.ctxt.Diag("operand mismatch: %v\n", p)
		}

		o1 = c.oprrr(p, p.As)
		rf := int((p.From.Reg) & 31)
		rt := int((p.To.Reg) & 31)
		r := int((p.Reg) & 31)
		o1 |= ((Q & 1) << 30) | ((size & 3) << 22) | (uint32(rf&31) << 16) | (uint32(r&31) << 5) | uint32(rt&31)

	case 94: /* vext $imm4, Vm.<T>, Vn.<T>, Vd.<T> */
		af := int(((p.GetFrom3().Reg) >> 5) & 15)
		at := int((p.To.Reg >> 5) & 15)
		a := int((p.Reg >> 5) & 15)
		index := int(p.From.Offset)

		if af != a || af != at {
			c.ctxt.Diag("invalid arrangement: %v", p)
			break
		}

		var Q uint32
		var b int
		if af == ARNG_8B {
			Q = 0
			b = 7
		} else if af == ARNG_16B {
			Q = 1
			b = 15
		} else {
			c.ctxt.Diag("invalid arrangement, should be B8 or B16: %v", p)
			break
		}

		if index < 0 || index > b {
			c.ctxt.Diag("illegal offset: %v", p)
		}

		o1 = c.opirr(p, p.As)
		rf := int((p.GetFrom3().Reg) & 31)
		rt := int((p.To.Reg) & 31)
		r := int((p.Reg) & 31)

		o1 |= ((Q & 1) << 30) | (uint32(r&31) << 16) | (uint32(index&15) << 11) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 95: /* vushr/vshl/vsri/vsli/vusra $shift, Vn.<T>, Vd.<T> */
		at := int((p.To.Reg >> 5) & 15)
		af := int((p.Reg >> 5) & 15)
		shift := int(p.From.Offset)

		if af != at {
			c.ctxt.Diag("invalid arrangement on op Vn.<T>, Vd.<T>: %v", p)
		}

		var Q uint32
		var imax, esize int

		switch af {
		case ARNG_8B, ARNG_4H, ARNG_2S:
			Q = 0
		case ARNG_16B, ARNG_8H, ARNG_4S, ARNG_2D:
			Q = 1
		default:
			c.ctxt.Diag("invalid arrangement on op Vn.<T>, Vd.<T>: %v", p)
		}

		switch af {
		case ARNG_8B, ARNG_16B:
			imax = 15
			esize = 8
		case ARNG_4H, ARNG_8H:
			imax = 31
			esize = 16
		case ARNG_2S, ARNG_4S:
			imax = 63
			esize = 32
		case ARNG_2D:
			imax = 127
			esize = 64
		}

		imm := 0
		switch p.As {
		case AVUSHR, AVSRI, AVUSRA:
			imm = esize*2 - shift
			if imm < esize || imm > imax {
				c.ctxt.Diag("shift out of range: %v", p)
			}
		case AVSHL, AVSLI:
			imm = esize + shift
			if imm > imax {
				c.ctxt.Diag("shift out of range: %v", p)
			}
		default:
			c.ctxt.Diag("invalid instruction %v\n", p)
		}

		o1 = c.opirr(p, p.As)
		rt := int((p.To.Reg) & 31)
		rf := int((p.Reg) & 31)

		o1 |= ((Q & 1) << 30) | (uint32(imm&0x7f) << 16) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 96: /* vst1 Vt1.<T>[index], offset(Rn) */
		af := int((p.From.Reg >> 5) & 15)
		rt := int((p.From.Reg) & 31)
		rf := int((p.To.Reg) & 31)
		r := int(p.To.Index & 31)
		index := int(p.From.Index)
		offset := c.regoff(&p.To)

		if o.scond == C_XPOST {
			if (p.To.Index != 0) && (offset != 0) {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			if p.To.Index == 0 && offset == 0 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
		}

		if offset != 0 {
			r = 31
		}

		var Q, S, size int
		var opcode uint32
		switch af {
		case ARNG_B:
			c.checkindex(p, index, 15)
			if o.scond == C_XPOST && offset != 0 && offset != 1 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index >> 3
			S = (index >> 2) & 1
			size = index & 3
			opcode = 0
		case ARNG_H:
			c.checkindex(p, index, 7)
			if o.scond == C_XPOST && offset != 0 && offset != 2 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index >> 2
			S = (index >> 1) & 1
			size = (index & 1) << 1
			opcode = 2
		case ARNG_S:
			c.checkindex(p, index, 3)
			if o.scond == C_XPOST && offset != 0 && offset != 4 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index >> 1
			S = index & 1
			size = 0
			opcode = 4
		case ARNG_D:
			c.checkindex(p, index, 1)
			if o.scond == C_XPOST && offset != 0 && offset != 8 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index
			S = 0
			size = 1
			opcode = 4
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		if o.scond == C_XPOST {
			o1 |= 27 << 23
		} else {
			o1 |= 26 << 23
		}

		o1 |= (uint32(Q&1) << 30) | (uint32(r&31) << 16) | ((opcode & 7) << 13) | (uint32(S&1) << 12) | (uint32(size&3) << 10) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 97: /* vld1 offset(Rn), vt.<T>[index] */
		at := int((p.To.Reg >> 5) & 15)
		rt := int((p.To.Reg) & 31)
		rf := int((p.From.Reg) & 31)
		r := int(p.From.Index & 31)
		index := int(p.To.Index)
		offset := c.regoff(&p.From)

		if o.scond == C_XPOST {
			if (p.From.Index != 0) && (offset != 0) {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			if p.From.Index == 0 && offset == 0 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
		}

		if offset != 0 {
			r = 31
		}

		Q := 0
		S := 0
		size := 0
		var opcode uint32
		switch at {
		case ARNG_B:
			c.checkindex(p, index, 15)
			if o.scond == C_XPOST && offset != 0 && offset != 1 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index >> 3
			S = (index >> 2) & 1
			size = index & 3
			opcode = 0
		case ARNG_H:
			c.checkindex(p, index, 7)
			if o.scond == C_XPOST && offset != 0 && offset != 2 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index >> 2
			S = (index >> 1) & 1
			size = (index & 1) << 1
			opcode = 2
		case ARNG_S:
			c.checkindex(p, index, 3)
			if o.scond == C_XPOST && offset != 0 && offset != 4 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index >> 1
			S = index & 1
			size = 0
			opcode = 4
		case ARNG_D:
			c.checkindex(p, index, 1)
			if o.scond == C_XPOST && offset != 0 && offset != 8 {
				c.ctxt.Diag("invalid offset: %v", p)
			}
			Q = index
			S = 0
			size = 1
			opcode = 4
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}

		if o.scond == C_XPOST {
			o1 |= 110 << 21
		} else {
			o1 |= 106 << 21
		}

		o1 |= (uint32(Q&1) << 30) | (uint32(r&31) << 16) | ((opcode & 7) << 13) | (uint32(S&1) << 12) | (uint32(size&3) << 10) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 98: /* MOVD (Rn)(Rm.SXTW[<<amount]),Rd */
		if isRegShiftOrExt(&p.From) {
			// extended or shifted offset register.
			c.checkShiftAmount(p, &p.From)

			o1 = c.opldrr(p, p.As, true)
			o1 |= c.encRegShiftOrExt(p, &p.From, p.From.Index) /* includes reg, op, etc */
		} else {
			// (Rn)(Rm), no extension or shift.
			o1 = c.opldrr(p, p.As, false)
			o1 |= uint32(p.From.Index&31) << 16
		}
		o1 |= uint32(p.From.Reg&31) << 5
		rt := int(p.To.Reg)
		o1 |= uint32(rt & 31)

	case 99: /* MOVD Rt, (Rn)(Rm.SXTW[<<amount]) */
		if isRegShiftOrExt(&p.To) {
			// extended or shifted offset register.
			c.checkShiftAmount(p, &p.To)

			o1 = c.opstrr(p, p.As, true)
			o1 |= c.encRegShiftOrExt(p, &p.To, p.To.Index) /* includes reg, op, etc */
		} else {
			// (Rn)(Rm), no extension or shift.
			o1 = c.opstrr(p, p.As, false)
			o1 |= uint32(p.To.Index&31) << 16
		}
		o1 |= uint32(p.To.Reg&31) << 5
		rf := int(p.From.Reg)
		o1 |= uint32(rf & 31)

	case 100: /* VTBL/VTBX Vn.<T>, [Vt1.<T>, Vt2.<T>, ...], Vd.<T> */
		af := int((p.From.Reg >> 5) & 15)
		at := int((p.To.Reg >> 5) & 15)
		if af != at {
			c.ctxt.Diag("invalid arrangement: %v\n", p)
		}
		var q, len uint32
		switch af {
		case ARNG_8B:
			q = 0
		case ARNG_16B:
			q = 1
		default:
			c.ctxt.Diag("invalid arrangement: %v", p)
		}
		rf := int(p.From.Reg)
		rt := int(p.To.Reg)
		offset := int(p.GetFrom3().Offset)
		opcode := (offset >> 12) & 15
		switch opcode {
		case 0x7:
			len = 0 // one register
		case 0xa:
			len = 1 // two register
		case 0x6:
			len = 2 // three registers
		case 0x2:
			len = 3 // four registers
		default:
			c.ctxt.Diag("invalid register numbers in ARM64 register list: %v", p)
		}
		var op uint32
		switch p.As {
		case AVTBL:
			op = 0
		case AVTBX:
			op = 1
		}
		o1 = q<<30 | 0xe<<24 | len<<13 | op<<12
		o1 |= (uint32(rf&31) << 16) | uint32(offset&31)<<5 | uint32(rt&31)

	case 102: /* vushll, vushll2, vuxtl, vuxtl2 */
		o1 = c.opirr(p, p.As)
		rf := p.Reg
		af := uint8((p.Reg >> 5) & 15)
		at := uint8((p.To.Reg >> 5) & 15)
		shift := int(p.From.Offset)
		if p.As == AVUXTL || p.As == AVUXTL2 {
			rf = p.From.Reg
			af = uint8((p.From.Reg >> 5) & 15)
			shift = 0
		}

		Q := (o1 >> 30) & 1
		var immh, width uint8
		switch pack(Q, af, at) {
		case pack(0, ARNG_8B, ARNG_8H):
			immh, width = 1, 8
		case pack(1, ARNG_16B, ARNG_8H):
			immh, width = 1, 8
		case pack(0, ARNG_4H, ARNG_4S):
			immh, width = 2, 16
		case pack(1, ARNG_8H, ARNG_4S):
			immh, width = 2, 16
		case pack(0, ARNG_2S, ARNG_2D):
			immh, width = 4, 32
		case pack(1, ARNG_4S, ARNG_2D):
			immh, width = 4, 32
		default:
			c.ctxt.Diag("operand mismatch: %v\n", p)
		}
		if !(0 <= shift && shift <= int(width-1)) {
			c.ctxt.Diag("shift amount out of range: %v\n", p)
		}
		o1 |= uint32(immh)<<19 | uint32(shift)<<16 | uint32(rf&31)<<5 | uint32(p.To.Reg&31)

	case 103: /* VEOR3/VBCAX Va.B16, Vm.B16, Vn.B16, Vd.B16 */
		ta := (p.From.Reg >> 5) & 15
		tm := (p.Reg >> 5) & 15
		td := (p.To.Reg >> 5) & 15
		tn := ((p.GetFrom3().Reg) >> 5) & 15

		if ta != tm || ta != tn || ta != td || ta != ARNG_16B {
			c.ctxt.Diag("invalid arrangement: %v", p)
			break
		}

		o1 = c.oprrr(p, p.As)
		ra := int(p.From.Reg)
		rm := int(p.Reg)
		rn := int(p.GetFrom3().Reg)
		rd := int(p.To.Reg)
		o1 |= uint32(rm&31)<<16 | uint32(ra&31)<<10 | uint32(rn&31)<<5 | uint32(rd)&31

	case 104: /* vxar $imm4, Vm.<T>, Vn.<T>, Vd.<T> */
		af := ((p.GetFrom3().Reg) >> 5) & 15
		at := (p.To.Reg >> 5) & 15
		a := (p.Reg >> 5) & 15
		index := int(p.From.Offset)

		if af != a || af != at {
			c.ctxt.Diag("invalid arrangement: %v", p)
			break
		}

		if af != ARNG_2D {
			c.ctxt.Diag("invalid arrangement, should be D2: %v", p)
			break
		}

		if index < 0 || index > 63 {
			c.ctxt.Diag("illegal offset: %v", p)
		}

		o1 = c.opirr(p, p.As)
		rf := (p.GetFrom3().Reg) & 31
		rt := (p.To.Reg) & 31
		r := (p.Reg) & 31

		o1 |= (uint32(r&31) << 16) | (uint32(index&63) << 10) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 105: /* vuaddw{2} Vm.<Tb>, Vn.<Ta>, Vd.<Ta> */
		af := uint8((p.From.Reg >> 5) & 15)
		at := uint8((p.To.Reg >> 5) & 15)
		a := uint8((p.Reg >> 5) & 15)
		if at != a {
			c.ctxt.Diag("invalid arrangement: %v", p)
			break
		}

		var Q, size uint32
		if p.As == AVUADDW2 {
			Q = 1
		}
		switch pack(Q, at, af) {
		case pack(0, ARNG_8H, ARNG_8B), pack(1, ARNG_8H, ARNG_16B):
			size = 0
		case pack(0, ARNG_4S, ARNG_4H), pack(1, ARNG_4S, ARNG_8H):
			size = 1
		case pack(0, ARNG_2D, ARNG_2S), pack(1, ARNG_2D, ARNG_4S):
			size = 2
		default:
			c.ctxt.Diag("operand mismatch: %v\n", p)
		}

		o1 = c.oprrr(p, p.As)
		rf := int((p.From.Reg) & 31)
		rt := int((p.To.Reg) & 31)
		r := int((p.Reg) & 31)
		o1 |= ((Q & 1) << 30) | ((size & 3) << 22) | (uint32(rf&31) << 16) | (uint32(r&31) << 5) | uint32(rt&31)

	case 106: // CASPx (Rs, Rs+1), (Rb), (Rt, Rt+1)
		rs := p.From.Reg
		rt := p.GetTo2().Reg
		rb := p.To.Reg
		rs1 := int16(p.From.Offset)
		rt1 := int16(p.GetTo2().Offset)

		enc, ok := atomicCASP[p.As]
		if !ok {
			c.ctxt.Diag("invalid CASP-like atomic instructions: %v\n", p)
		}
		// for CASPx-like instructions, Rs<0> != 1 && Rt<0> != 1
		switch {
		case rs&1 != 0:
			c.ctxt.Diag("source register pair must start from even register: %v\n", p)
			break
		case rt&1 != 0:
			c.ctxt.Diag("destination register pair must start from even register: %v\n", p)
			break
		case rs != rs1-1:
			c.ctxt.Diag("source register pair must be contiguous: %v\n", p)
			break
		case rt != rt1-1:
			c.ctxt.Diag("destination register pair must be contiguous: %v\n", p)
			break
		}
		// rt can't be sp.
		if rt == REG_RSP {
			c.ctxt.Diag("illegal destination register: %v\n", p)
		}
		o1 |= enc | uint32(rs&31)<<16 | uint32(rb&31)<<5 | uint32(rt&31)

	case 107: /* tlbi, dc */
		op, ok := sysInstFields[SpecialOperand(p.From.Offset)]
		if !ok || (p.As == ATLBI && op.cn != 8) || (p.As == ADC && op.cn != 7) {
			c.ctxt.Diag("illegal argument: %v\n", p)
			break
		}
		o1 = c.opirr(p, p.As)
		if op.hasOperand2 {
			if p.To.Reg == obj.REG_NONE {
				c.ctxt.Diag("missing register at operand 2: %v\n", p)
			}
			o1 |= uint32(p.To.Reg & 0x1F)
		} else {
			if p.To.Reg != obj.REG_NONE || p.Reg != obj.REG_NONE {
				c.ctxt.Diag("extraneous register at operand 2: %v\n", p)
			}
			o1 |= uint32(0x1F)
		}
		o1 |= uint32(SYSARG4(int(op.op1), int(op.cn), int(op.cm), int(op.op2)))

	case 108: /* vshrn{2} $shift, Vn.<Tb>, Vd.<Ta> */
		at := uint8((p.To.Reg >> 5) & 15)
		af := uint8((p.Reg >> 5) & 15)
		shift := uint32(p.From.Offset)

		var Q, imax, size uint32

		if p.As == AVSHRN2 {
			Q = 1
		}
		switch pack(Q, af, at) {
		case pack(0, ARNG_8H, ARNG_8B), pack(1, ARNG_8H, ARNG_16B):
			size, imax = 8, 15
		case pack(0, ARNG_4S, ARNG_4H), pack(1, ARNG_4S, ARNG_8H):
			size, imax = 16, 31
		case pack(0, ARNG_2D, ARNG_2S), pack(1, ARNG_2D, ARNG_4S):
			size, imax = 32, 63
		default:
			c.ctxt.Diag("operand mismatch: %v\n", p)
		}

		imm := size + shift
		if imm > imax {
			c.ctxt.Diag("shift out of range: %v", p)
		}

		o1 = c.opirr(p, p.As)
		rt := int((p.To.Reg) & 31)
		rf := int((p.Reg) & 31)

		o1 |= ((Q & 1) << 30) | (imm & 0x7f << 16) | (uint32(rf&31) << 5) | uint32(rt&31)

	case 109: /* rprfm (Rn), Rm, <rprfop | $imm6> */
		srcReg := p.From.Reg  // Rn - register with base address
		rangeReg := p.Reg     // Rm - register with prefetch metadata
		var operation uint32
		var ok bool

		// Operation is either a 6-bit immediate or a named prefetch op.
		if p.To.Type == obj.TYPE_CONST {
			operation = uint32(p.To.Offset)
			if operation > 63 {
				c.ctxt.Diag("range prefetch immediate not in the range 0 to 63: %v", p)
			}
		} else {
			operation, ok = rprfopfield[SpecialOperand(p.To.Offset)]
			if !ok {
				c.ctxt.Diag("illegal range prefetch operand, expected PLDKEEP, PSTKEEP, PLDSTRM or PSTSTRM: %v", p)
			}
		}

		// Bit placement: the 6-bit value is scattered to match the
		// architectural encoding (bits 15,13,12,2-0). This is because the
		// instruction word reuses fields from the base load/store hint space.
		//   option2 (bit5) -> bit15 (<<10)
		//   option0 (bit4) -> bit13 (<<9)
		//   S       (bit3) -> bit12 (<<9)
		//   Rt<2:0> (bits2-0) -> bits2-0 (no shift)
		// Rt<4:3> are already set by c.opirr() and are fixed for RPRFM.
		option2 := (operation & 32) << 10
		option0 := (operation & 16) << 9
		s := (operation & 8) << 9
		rt := operation & 7

		o1 = c.opirr(p, p.As)
		o1 |= (uint32(rangeReg&31)<<16) | (uint32(srcReg&31)<<5) | option2 | option0 | s | rt
	}
	out[0] = o1
	out[1] = o2
	out[2] = o3
	out[3] = o4
	out[4] = o5

	return int(o.size(c.ctxt, p) / 4)
}

func (c *ctxt7) addrRelocType(p *obj.Prog) objabi.RelocType {
	switch movesize(p.As) {
	case 0:
		return objabi.R_ARM64_PCREL_LDST8
	case 1:
		return objabi.R_ARM64_PCREL_LDST16
	case 2:
		return objabi.R_ARM64_PCREL_LDST32
	case 3:
		return objabi.R_ARM64_PCREL_LDST64
	default:
		c.ctxt.Diag("use R_ADDRARM64 relocation type for: %v\n", p)
	}
	return -1
}

/*
 * basic Rm op Rn -> Rd (using shifted register with 0)
 * also op Rn -> Rt
 * also Rm*Rn op Ra -> Rd
 * also Vm op Vn -> Vd
 */
func (c *ctxt7) oprrr(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case AADC:
		return S64 | 0<<30 | 0<<29 | 0xd0<<21 | 0<<10

	case AADCW:
		return S32 | 0<<30 | 0<<29 | 0xd0<<21 | 0<<10

	case AADCS:
		return S64 | 0<<30 | 1<<29 | 0xd0<<21 | 0<<10

	case AADCSW:
		return S32 | 0<<30 | 1<<29 | 0xd0<<21 | 0<<10

	case ANGC, ASBC:
		return S64 | 1<<30 | 0<<29 | 0xd0<<21 | 0<<10

	case ANGCS, ASBCS:
		return S64 | 1<<30 | 1<<29 | 0xd0<<21 | 0<<10

	case ANGCW, ASBCW:
		return S32 | 1<<30 | 0<<29 | 0xd0<<21 | 0<<10

	case ANGCSW, ASBCSW:
		return S32 | 1<<30 | 1<<29 | 0xd0<<21 | 0<<10

	case AADD:
		return S64 | 0<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case AADDW:
		return S32 | 0<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case ACMN, AADDS:
		return S64 | 0<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case ACMNW, AADDSW:
		return S32 | 0<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case ASUB:
		return S64 | 1<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case ASUBW:
		return S32 | 1<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case ACMP, ASUBS:
		return S64 | 1<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case ACMPW, ASUBSW:
		return S32 | 1<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 0<<21 | 0<<10

	case AAND:
		return S64 | 0<<29 | 0xA<<24

	case AANDW:
		return S32 | 0<<29 | 0xA<<24

	case AMOVD, AORR:
		return S64 | 1<<29 | 0xA<<24

		//	case AMOVW:
	case AMOVWU, AORRW:
		return S32 | 1<<29 | 0xA<<24

	case AEOR:
		return S64 | 2<<29 | 0xA<<24

	case AEORW:
		return S32 | 2<<29 | 0xA<<24

	case AANDS, ATST:
		return S64 | 3<<29 | 0xA<<24

	case AANDSW, ATSTW:
		return S32 | 3<<29 | 0xA<<24

	case ABIC:
		return S64 | 0<<29 | 0xA<<24 | 1<<21

	case ABICW:
		return S32 | 0<<29 | 0xA<<24 | 1<<21

	case ABICS:
		return S64 | 3<<29 | 0xA<<24 | 1<<21

	case ABICSW:
		return S32 | 3<<29 | 0xA<<24 | 1<<21

	case AEON:
		return S64 | 2<<29 | 0xA<<24 | 1<<21

	case AEONW:
		return S32 | 2<<29 | 0xA<<24 | 1<<21

	case AMVN, AORN:
		return S64 | 1<<29 | 0xA<<24 | 1<<21

	case AMVNW, AORNW:
		return S32 | 1<<29 | 0xA<<24 | 1<<21

	case AASR:
		return S64 | OPDP2(10) /* also ASRV */

	case AASRW:
		return S32 | OPDP2(10)

	case ALSL:
		return S64 | OPDP2(8)

	case ALSLW:
		return S32 | OPDP2(8)

	case ALSR:
		return S64 | OPDP2(9)

	case ALSRW:
		return S32 | OPDP2(9)

	case AROR:
		return S64 | OPDP2(11)

	case ARORW:
		return S32 | OPDP2(11)

	case ACCMN:
		return S64 | 0<<30 | 1<<29 | 0xD2<<21 | 0<<11 | 0<<10 | 0<<4 /* cond<<12 | nzcv<<0 */

	case ACCMNW:
		return S32 | 0<<30 | 1<<29 | 0xD2<<21 | 0<<11 | 0<<10 | 0<<4

	case ACCMP:
		return S64 | 1<<30 | 1<<29 | 0xD2<<21 | 0<<11 | 0<<10 | 0<<4 /* imm5<<16 | cond<<12 | nzcv<<0 */

	case ACCMPW:
		return S32 | 1<<30 | 1<<29 | 0xD2<<21 | 0<<11 | 0<<10 | 0<<4

	case ACRC32B:
		return S32 | OPDP2(16)

	case ACRC32H:
		return S32 | OPDP2(17)

	case ACRC32W:
		return S32 | OPDP2(18)

	case ACRC32X:
		return S64 | OPDP2(19)

	case ACRC32CB:
		return S32 | OPDP2(20)

	case ACRC32CH:
		return S32 | OPDP2(21)

	case ACRC32CW:
		return S32 | OPDP2(22)

	case ACRC32CX:
		return S64 | OPDP2(23)

	case ACSEL:
		return S64 | 0<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 0<<10

	case ACSELW:
		return S32 | 0<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 0<<10

	case ACSET:
		return S64 | 0<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 1<<10

	case ACSETW:
		return S32 | 0<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 1<<10

	case ACSETM:
		return S64 | 1<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 0<<10

	case ACSETMW:
		return S32 | 1<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 0<<10

	case ACINC, ACSINC:
		return S64 | 0<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 1<<10

	case ACINCW, ACSINCW:
		return S32 | 0<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 1<<10

	case ACINV, ACSINV:
		return S64 | 1<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 0<<10

	case ACINVW, ACSINVW:
		return S32 | 1<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 0<<10

	case ACNEG, ACSNEG:
		return S64 | 1<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 1<<10

	case ACNEGW, ACSNEGW:
		return S32 | 1<<30 | 0<<29 | 0xD4<<21 | 0<<11 | 1<<10

	case AMUL, AMADD:
		return S64 | 0<<29 | 0x1B<<24 | 0<<21 | 0<<15

	case AMULW, AMADDW:
		return S32 | 0<<29 | 0x1B<<24 | 0<<21 | 0<<15

	case AMNEG, AMSUB:
		return S64 | 0<<29 | 0x1B<<24 | 0<<21 | 1<<15

	case AMNEGW, AMSUBW:
		return S32 | 0<<29 | 0x1B<<24 | 0<<21 | 1<<15

	case AMRS:
		return SYSOP(1, 2, 0, 0, 0, 0, 0)

	case AMSR:
		return SYSOP(0, 2, 0, 0, 0, 0, 0)

	case ANEG:
		return S64 | 1<<30 | 0<<29 | 0xB<<24 | 0<<21

	case ANEGW:
		return S32 | 1<<30 | 0<<29 | 0xB<<24 | 0<<21

	case ANEGS:
		return S64 | 1<<30 | 1<<29 | 0xB<<24 | 0<<21

	case ANEGSW:
		return S32 | 1<<30 | 1<<29 | 0xB<<24 | 0<<21

	case AREM, ASDIV:
		return S64 | OPDP2(3)

	case AREMW, ASDIVW:
		return S32 | OPDP2(3)

	case ASMULL, ASMADDL:
		return OPDP3(1, 0, 1, 0)

	case ASMNEGL, ASMSUBL:
		return OPDP3(1, 0, 1, 1)

	case ASMULH:
		return OPDP3(1, 0, 2, 0)

	case AUMULL, AUMADDL:
		return OPDP3(1, 0, 5, 0)

	case AUMNEGL, AUMSUBL:
		return OPDP3(1, 0, 5, 1)

	case AUMULH:
		return OPDP3(1, 0, 6, 0)

	case AUREM, AUDIV:
		return S64 | OPDP2(2)

	case AUREMW, AUDIVW:
		return S32 | OPDP2(2)

	case AAESE:
		return 0x4E<<24 | 2<<20 | 8<<16 | 4<<12 | 2<<10

	case AAESD:
		return 0x4E<<24 | 2<<20 | 8<<16 | 5<<12 | 2<<10

	case AAESMC:
		return 0x4E<<24 | 2<<20 | 8<<16 | 6<<12 | 2<<10

	case AAESIMC:
		return 0x4E<<24 | 2<<20 | 8<<16 | 7<<12 | 2<<10

	case ASHA1C:
		return 0x5E<<24 | 0<<12

	case ASHA1P:
		return 0x5E<<24 | 1<<12

	case ASHA1M:
		return 0x5E<<24 | 2<<12

	case ASHA1SU0:
		return 0x5E<<24 | 3<<12

	case ASHA256H:
		return 0x5E<<24 | 4<<12

	case ASHA256H2:
		return 0x5E<<24 | 5<<12

	case ASHA256SU1:
		return 0x5E<<24 | 6<<12

	case ASHA1H:
		return 0x5E<<24 | 2<<20 | 8<<16 | 0<<12 | 2<<10

	case ASHA1SU1:
		return 0x5E<<24 | 2<<20 | 8<<16 | 1<<12 | 2<<10

	case ASHA256SU0:
		return 0x5E<<24 | 2<<20 | 8<<16 | 2<<12 | 2<<10

	case ASHA512H:
		return 0xCE<<24 | 3<<21 | 8<<12

	case ASHA512H2:
		return 0xCE<<24 | 3<<21 | 8<<12 | 4<<8

	case ASHA512SU1:
		return 0xCE<<24 | 3<<21 | 8<<12 | 8<<8

	case ASHA512SU0:
		return 0xCE<<24 | 3<<22 | 8<<12

	case AFCVTZSD:
		return FPCVTI(1, 0, 1, 3, 0)

	case AFCVTZSDW:
		return FPCVTI(0, 0, 1, 3, 0)

	case AFCVTZSS:
		return FPCVTI(1, 0, 0, 3, 0)

	case AFCVTZSSW:
		return FPCVTI(0, 0, 0, 3, 0)

	case AFCVTZUD:
		return FPCVTI(1, 0, 1, 3, 1)

	case AFCVTZUDW:
		return FPCVTI(0, 0, 1, 3, 1)

	case AFCVTZUS:
		return FPCVTI(1, 0, 0, 3, 1)

	case AFCVTZUSW:
		return FPCVTI(0, 0, 0, 3, 1)

	case ASCVTFD:
		return FPCVTI(1, 0, 1, 0, 2)

	case ASCVTFS:
		return FPCVTI(1, 0, 0, 0, 2)

	case ASCVTFWD:
		return FPCVTI(0, 0, 1, 0, 2)

	case ASCVTFWS:
		return FPCVTI(0, 0, 0, 0, 2)

	case AUCVTFD:
		return FPCVTI(1, 0, 1, 0, 3)

	case AUCVTFS:
		return FPCVTI(1, 0, 0, 0, 3)

	case AUCVTFWD:
		return FPCVTI(0, 0, 1, 0, 3)

	case AUCVTFWS:
		return FPCVTI(0, 0, 0, 0, 3)

	case AFADDS:
		return FPOP2S(0, 0, 0, 2)

	case AFADDD:
		return FPOP2S(0, 0, 1, 2)

	case AFSUBS:
		return FPOP2S(0, 0, 0, 3)

	case AFSUBD:
		return FPOP2S(0, 0, 1, 3)

	case AFMADDD:
		return FPOP3S(0, 0, 1, 0, 0)

	case AFMADDS:
		return FPOP3S(0, 0, 0, 0, 0)

	case AFMSUBD:
		return FPOP3S(0, 0, 1, 0, 1)

	case AFMSUBS:
		return FPOP3S(0, 0, 0, 0, 1)

	case AFNMADDD:
		return FPOP3S(0, 0, 1, 1, 0)

	case AFNMADDS:
		return FPOP3S(0, 0, 0, 1, 0)

	case AFNMSUBD:
		return FPOP3S(0, 0, 1, 1, 1)

	case AFNMSUBS:
		return FPOP3S(0, 0, 0, 1, 1)

	case AFMULS:
		return FPOP2S(0, 0, 0, 0)

	case AFMULD:
		return FPOP2S(0, 0, 1, 0)

	case AFDIVS:
		return FPOP2S(0, 0, 0, 1)

	case AFDIVD:
		return FPOP2S(0, 0, 1, 1)

	case AFMAXS:
		return FPOP2S(0, 0, 0, 4)

	case AFMINS:
		return FPOP2S(0, 0, 0, 5)

	case AFMAXD:
		return FPOP2S(0, 0, 1, 4)

	case AFMIND:
		return FPOP2S(0, 0, 1, 5)

	case AFMAXNMS:
		return FPOP2S(0, 0, 0, 6)

	case AFMAXNMD:
		return FPOP2S(0, 0, 1, 6)

	case AFMINNMS:
		return FPOP2S(0, 0, 0, 7)

	case AFMINNMD:
		return FPOP2S(0, 0, 1, 7)

	case AFNMULS:
		return FPOP2S(0, 0, 0, 8)

	case AFNMULD:
		return FPOP2S(0, 0, 1, 8)

	case AFCMPS:
		return FPCMP(0, 0, 0, 0, 0)

	case AFCMPD:
		return FPCMP(0, 0, 1, 0, 0)

	case AFCMPES:
		return FPCMP(0, 0, 0, 0, 16)

	case AFCMPED:
		return FPCMP(0, 0, 1, 0, 16)

	case AFCCMPS:
		return FPCCMP(0, 0, 0, 0)

	case AFCCMPD:
		return FPCCMP(0, 0, 1, 0)

	case AFCCMPES:
		return FPCCMP(0, 0, 0, 1)

	case AFCCMPED:
		return FPCCMP(0, 0, 1, 1)

	case AFCSELS:
		return 0x1E<<24 | 0<<22 | 1<<21 | 3<<10

	case AFCSELD:
		return 0x1E<<24 | 1<<22 | 1<<21 | 3<<10

	case AFMOVS:
		return FPOP1S(0, 0, 0, 0)

	case AFABSS:
		return FPOP1S(0, 0, 0, 1)

	case AFNEGS:
		return FPOP1S(0, 0, 0, 2)

	case AFSQRTS:
		return FPOP1S(0, 0, 0, 3)

	case AFCVTSD:
		return FPOP1S(0, 0, 0, 5)

	case AFCVTSH:
		return FPOP1S(0, 0, 0, 7)

	case AFRINTNS:
		return FPOP1S(0, 0, 0, 8)

	case AFRINTPS:
		return FPOP1S(0, 0, 0, 9)

	case AFRINTMS:
		return FPOP1S(0, 0, 0, 10)

	case AFRINTZS:
		return FPOP1S(0, 0, 0, 11)

	case AFRINTAS:
		return FPOP1S(0, 0, 0, 12)

	case AFRINTXS:
		return FPOP1S(0, 0, 0, 14)

	case AFRINTIS:
		return FPOP1S(0, 0, 0, 15)

	case AFMOVD:
		return FPOP1S(0, 0, 1, 0)

	case AFABSD:
		return FPOP1S(0, 0, 1, 1)

	case AFNEGD:
		return FPOP1S(0, 0, 1, 2)

	case AFSQRTD:
		return FPOP1S(0, 0, 1, 3)

	case AFCVTDS:
		return FPOP1S(0, 0, 1, 4)

	case AFCVTDH:
		return FPOP1S(0, 0, 1, 7)

	case AFRINTND:
		return FPOP1S(0, 0, 1, 8)

	case AFRINTPD:
		return FPOP1S(0, 0, 1, 9)

	case AFRINTMD:
		return FPOP1S(0, 0, 1, 10)

	case AFRINTZD:
		return FPOP1S(0, 0, 1, 11)

	case AFRINTAD:
		return FPOP1S(0, 0, 1, 12)

	case AFRINTXD:
		return FPOP1S(0, 0, 1, 14)

	case AFRINTID:
		return FPOP1S(0, 0, 1, 15)

	case AFCVTHS:
		return FPOP1S(0, 0, 3, 4)

	case AFCVTHD:
		return FPOP1S(0, 0, 3, 5)

	case AVADD:
		return 7<<25 | 1<<21 | 1<<15 | 1<<10

	case AVSUB:
		return 0x17<<25 | 1<<21 | 1<<15 | 1<<10

	case AVADDP:
		return 7<<25 | 1<<21 | 1<<15 | 15<<10

	case AVAND:
		return 7<<25 | 1<<21 | 7<<10

	case AVBCAX:
		return 0xCE<<24 | 1<<21

	case AVCMEQ:
		return 1<<29 | 0x71<<21 | 0x23<<10

	case AVCNT:
		return 0xE<<24 | 0x10<<17 | 5<<12 | 2<<10

	case AVZIP1:
		return 0xE<<24 | 3<<12 | 2<<10

	case AVZIP2:
		return 0xE<<24 | 1<<14 | 3<<12 | 2<<10

	case AVEOR:
		return 1<<29 | 0x71<<21 | 7<<10

	case AVEOR3:
		return 0xCE << 24

	case AVORR:
		return 7<<25 | 5<<21 | 7<<10

	case AVREV16:
		return 3<<26 | 2<<24 | 1<<21 | 3<<11

	case AVRAX1:
		return 0xCE<<24 | 3<<21 | 1<<15 | 3<<10

	case AVREV32:
		return 11<<26 | 2<<24 | 1<<21 | 1<<11

	case AVREV64:
		return 3<<26 | 2<<24 | 1<<21 | 1<<11

	case AVMOV:
		return 7<<25 | 5<<21 | 7<<10

	case AVADDV:
		return 7<<25 | 3<<20 | 3<<15 | 7<<11

	case AVUADDLV:
		return 1<<29 | 7<<25 | 3<<20 | 7<<11

	case AVFMLA:
		return 7<<25 | 0<<23 | 1<<21 | 3<<14 | 3<<10

	case AVFMLS:
		return 7<<25 | 1<<23 | 1<<21 | 3<<14 | 3<<10

	case AVPMULL, AVPMULL2:
		return 0xE<<24 | 1<<21 | 0x38<<10

	case AVRBIT:
		return 0x2E<<24 | 1<<22 | 0x10<<17 | 5<<12 | 2<<10

	case AVLD1, AVLD2, AVLD3, AVLD4:
		return 3<<26 | 1<<22

	case AVLD1R, AVLD3R:
		return 0xD<<24 | 1<<22

	case AVLD2R, AVLD4R:
		return 0xD<<24 | 3<<21

	case AVBIF:
		return 1<<29 | 7<<25 | 7<<21 | 7<<10

	case AVBIT:
		return 1<<29 | 0x75<<21 | 7<<10

	case AVBSL:
		return 1<<29 | 0x73<<21 | 7<<10

	case AVCMTST:
		return 0xE<<24 | 1<<21 | 0x23<<10

	case AVUMAX:
		return 1<<29 | 7<<25 | 1<<21 | 0x19<<10

	case AVUMIN:
		return 1<<29 | 7<<25 | 1<<21 | 0x1b<<10

	case AVUZP1:
		return 7<<25 | 3<<11

	case AVUZP2:
		return 7<<25 | 1<<14 | 3<<11

	case AVUADDW, AVUADDW2:
		return 0x17<<25 | 1<<21 | 1<<12

	case AVTRN1:
		return 7<<25 | 5<<11

	case AVTRN2:
		return 7<<25 | 1<<14 | 5<<11
	}

	c.ctxt.Diag("%v: bad rrr %d %v", p, a, a)
	return 0
}

/*
 * imm -> Rd
 * imm op Rn -> Rd
 */
func (c *ctxt7) opirr(p *obj.Prog, a obj.As) uint32 {
	switch a {
	/* op $addcon, Rn, Rd */
	case AMOVD, AADD:
		return S64 | 0<<30 | 0<<29 | 0x11<<24

	case ACMN, AADDS:
		return S64 | 0<<30 | 1<<29 | 0x11<<24

	case AMOVW, AADDW:
		return S32 | 0<<30 | 0<<29 | 0x11<<24

	case ACMNW, AADDSW:
		return S32 | 0<<30 | 1<<29 | 0x11<<24

	case ASUB:
		return S64 | 1<<30 | 0<<29 | 0x11<<24

	case ACMP, ASUBS:
		return S64 | 1<<30 | 1<<29 | 0x11<<24

	case ASUBW:
		return S32 | 1<<30 | 0<<29 | 0x11<<24

	case ACMPW, ASUBSW:
		return S32 | 1<<30 | 1<<29 | 0x11<<24

		/* op $imm(SB), Rd; op label, Rd */
	case AADR:
		return 0<<31 | 0x10<<24

	case AADRP:
		return 1<<31 | 0x10<<24

		/* op $bimm, Rn, Rd */
	case AAND, ABIC:
		return S64 | 0<<29 | 0x24<<23

	case AANDW, ABICW:
		return S32 | 0<<29 | 0x24<<23 | 0<<22

	case AORR, AORN:
		return S64 | 1<<29 | 0x24<<23

	case AORRW, AORNW:
		return S32 | 1<<29 | 0x24<<23 | 0<<22

	case AEOR, AEON:
		return S64 | 2<<29 | 0x24<<23

	case AEORW, AEONW:
		return S32 | 2<<29 | 0x24<<23 | 0<<22

	case AANDS, ABICS, ATST:
		return S64 | 3<<29 | 0x24<<23

	case AANDSW, ABICSW, ATSTW:
		return S32 | 3<<29 | 0x24<<23 | 0<<22

	case AASR:
		return S64 | 0<<29 | 0x26<<23 /* alias of SBFM */

	case AASRW:
		return S32 | 0<<29 | 0x26<<23 | 0<<22

		/* op $width, $lsb, Rn, Rd */
	case ABFI:
		return S64 | 2<<29 | 0x26<<23 | 1<<22
		/* alias of BFM */

	case ABFIW:
		return S32 | 2<<29 | 0x26<<23 | 0<<22

		/* op $imms, $immr, Rn, Rd */
	case ABFM:
		return S64 | 1<<29 | 0x26<<23 | 1<<22

	case ABFMW:
		return S32 | 1<<29 | 0x26<<23 | 0<<22

	case ASBFM:
		return S64 | 0<<29 | 0x26<<23 | 1<<22

	case ASBFMW:
		return S32 | 0<<29 | 0x26<<23 | 0<<22

	case AUBFM:
		return S64 | 2<<29 | 0x26<<23 | 1<<22

	case AUBFMW:
		return S32 | 2<<29 | 0x26<<23 | 0<<22

	case ABFXIL:
		return S64 | 1<<29 | 0x26<<23 | 1<<22 /* alias of BFM */

	case ABFXILW:
		return S32 | 1<<29 | 0x26<<23 | 0<<22

	case AEXTR:
		return S64 | 0<<29 | 0x27<<23 | 1<<22 | 0<<21

	case AEXTRW:
		return S32 | 0<<29 | 0x27<<23 | 0<<22 | 0<<21

	case ACBNZ:
		return S64 | 0x1A<<25 | 1<<24

	case ACBNZW:
		return S32 | 0x1A<<25 | 1<<24

	case ACBZ:
		return S64 | 0x1A<<25 | 0<<24

	case ACBZW:
		return S32 | 0x1A<<25 | 0<<24

	case ACCMN:
		return S64 | 0<<30 | 1<<29 | 0xD2<<21 | 1<<11 | 0<<10 | 0<<4 /* imm5<<16 | cond<<12 | nzcv<<0 */

	case ACCMNW:
		return S32 | 0<<30 | 1<<29 | 0xD2<<21 | 1<<11 | 0<<10 | 0<<4

	case ACCMP:
		return S64 | 1<<30 | 1<<29 | 0xD2<<21 | 1<<11 | 0<<10 | 0<<4 /* imm5<<16 | cond<<12 | nzcv<<0 */

	case ACCMPW:
		return S32 | 1<<30 | 1<<29 | 0xD2<<21 | 1<<11 | 0<<10 | 0<<4

	case AMOVK:
		return S64 | 3<<29 | 0x25<<23

	case AMOVKW:
		return S32 | 3<<29 | 0x25<<23

	case AMOVN:
		return S64 | 0<<29 | 0x25<<23

	case AMOVNW:
		return S32 | 0<<29 | 0x25<<23

	case AMOVZ:
		return S64 | 2<<29 | 0x25<<23

	case AMOVZW:
		return S32 | 2<<29 | 0x25<<23

	case AMSR:
		return SYSOP(0, 0, 0, 4, 0, 0, 0x1F) /* MSR (immediate) */

	case AAT,
		ADC,
		AIC,
		ATLBI,
		ASYS:
		return SYSOP(0, 1, 0, 0, 0, 0, 0)

	case ASYSL:
		return SYSOP(1, 1, 0, 0, 0, 0, 0)

	case ATBZ:
		return 0x36 << 24

	case ATBNZ:
		return 0x37 << 24

	case ADSB:
		return SYSOP(0, 0, 3, 3, 0, 4, 0x1F)

	case ADMB:
		return SYSOP(0, 0, 3, 3, 0, 5, 0x1F)

	case AISB:
		return SYSOP(0, 0, 3, 3, 0, 6, 0x1F)

	case AHINT:
		return SYSOP(0, 0, 3, 2, 0, 0, 0x1F)

	case AVEXT:
		return 0x2E<<24 | 0<<23 | 0<<21 | 0<<15

	case AVUSHR:
		return 0x5E<<23 | 1<<10

	case AVSHL:
		return 0x1E<<23 | 21<<10

	case AVSRI:
		return 0x5E<<23 | 17<<10

	case AVSLI:
		return 0x5E<<23 | 21<<10

	case AVUSHLL, AVUXTL:
		return 1<<29 | 15<<24 | 0x29<<10

	case AVUSHLL2, AVUXTL2:
		return 3<<29 | 15<<24 | 0x29<<10

	case AVXAR:
		return 0xCE<<24 | 1<<23

	case AVUSRA:
		return 1<<29 | 15<<24 | 5<<10

	case APRFM:
		return 0xf9<<24 | 2<<22

	case ARPRFM:
		return 0xf8<<24 | 5<<21 | 18<<10 | 3<<3

	case AVSHRN, AVSHRN2:
		return 0xF<<24 | 0x21<<10
	}

	c.ctxt.Diag("%v: bad irr %v", p, a)
	return 0
}

func (c *ctxt7) opbit(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case ACLS:
		return S64 | OPBIT(5)

	case ACLSW:
		return S32 | OPBIT(5)

	case ACLZ:
		return S64 | OPBIT(4)

	case ACLZW:
		return S32 | OPBIT(4)

	case ARBIT:
		return S64 | OPBIT(0)

	case ARBITW:
		return S32 | OPBIT(0)

	case AREV:
		return S64 | OPBIT(3)

	case AREVW:
		return S32 | OPBIT(2)

	case AREV16:
		return S64 | OPBIT(1)

	case AREV16W:
		return S32 | OPBIT(1)

	case AREV32:
		return S64 | OPBIT(2)

	default:
		c.ctxt.Diag("bad bit op\n%v", p)
		return 0
	}
}

/*
 * add/subtract sign or zero-extended register
 */
func (c *ctxt7) opxrrr(p *obj.Prog, a obj.As, rd, rn, rm int16, extend bool) uint32 {
	extension := uint32(0)
	if !extend {
		if isADDop(a) {
			extension = LSL0_64
		}
		if isADDWop(a) {
			extension = LSL0_32
		}
	}

	var op uint32

	switch a {
	case AADD:
		op = S64 | 0<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	case AADDW:
		op = S32 | 0<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	case ACMN, AADDS:
		op = S64 | 0<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	case ACMNW, AADDSW:
		op = S32 | 0<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	case ASUB:
		op = S64 | 1<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	case ASUBW:
		op = S32 | 1<<30 | 0<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	case ACMP, ASUBS:
		op = S64 | 1<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	case ACMPW, ASUBSW:
		op = S32 | 1<<30 | 1<<29 | 0x0b<<24 | 0<<22 | 1<<21 | extension

	default:
		c.ctxt.Diag("bad opxrrr %v\n%v", a, p)
		return 0
	}

	op |= uint32(rm&0x1f)<<16 | uint32(rn&0x1f)<<5 | uint32(rd&0x1f)

	return op
}

func (c *ctxt7) opimm(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case ASVC:
		return 0xD4<<24 | 0<<21 | 1 /* imm16<<5 */

	case AHVC:
		return 0xD4<<24 | 0<<21 | 2

	case ASMC:
		return 0xD4<<24 | 0<<21 | 3

	case ABRK:
		return 0xD4<<24 | 1<<21 | 0

	case AHLT:
		return 0xD4<<24 | 2<<21 | 0

	case ADCPS1:
		return 0xD4<<24 | 5<<21 | 1

	case ADCPS2:
		return 0xD4<<24 | 5<<21 | 2

	case ADCPS3:
		return 0xD4<<24 | 5<<21 | 3

	case ACLREX:
		return SYSOP(0, 0, 3, 3, 0, 2, 0x1F)
	}

	c.ctxt.Diag("%v: bad imm %v", p, a)
	return 0
}

func (c *ctxt7) brdist(p *obj.Prog, preshift int, flen int, shift int) int64 {
	v := int64(0)
	t := int64(0)
	var q *obj.Prog
	if p.To.Type == obj.TYPE_BRANCH {
		q = p.To.Target()
	} else if p.From.Type == obj.TYPE_BRANCH { // adr, adrp
		q = p.From.Target()
	}
	if q == nil {
		// TODO: don't use brdist for this case, as it isn't a branch.
		// (Calls from omovlit, and maybe adr/adrp opcodes as well.)
		q = p.Pool
	}
	if q != nil {
		v = (q.Pc >> uint(preshift)) - (c.pc >> uint(preshift))
		if (v & ((1 << uint(shift)) - 1)) != 0 {
			c.ctxt.Diag("misaligned label\n%v", p)
		}
		v >>= uint(shift)
		t = int64(1) << uint(flen-1)
		if v < -t || v >= t {
			c.ctxt.Diag("branch too far %#x vs %#x [%p]\n%v\n%v", v, t, c.blitrl, p, q)
			panic("branch too far")
		}
	}

	return v & ((t << 1) - 1)
}

/*
 * pc-relative branches
 */
func (c *ctxt7) opbra(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case ABEQ:
		return OPBcc(0x0)

	case ABNE:
		return OPBcc(0x1)

	case ABCS:
		return OPBcc(0x2)

	case ABHS:
		return OPBcc(0x2)

	case ABCC:
		return OPBcc(0x3)

	case ABLO:
		return OPBcc(0x3)

	case ABMI:
		return OPBcc(0x4)

	case ABPL:
		return OPBcc(0x5)

	case ABVS:
		return OPBcc(0x6)

	case ABVC:
		return OPBcc(0x7)

	case ABHI:
		return OPBcc(0x8)

	case ABLS:
		return OPBcc(0x9)

	case ABGE:
		return OPBcc(0xa)

	case ABLT:
		return OPBcc(0xb)

	case ABGT:
		return OPBcc(0xc)

	case ABLE:
		return OPBcc(0xd) /* imm19<<5 | cond */

	case AB:
		return 0<<31 | 5<<26 /* imm26 */

	case obj.ADUFFZERO, obj.ADUFFCOPY, ABL:
		return 1<<31 | 5<<26
	}

	c.ctxt.Diag("%v: bad bra %v", p, a)
	return 0
}

func (c *ctxt7) opbrr(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case ABL:
		return OPBLR(1) /* BLR */

	case AB:
		return OPBLR(0) /* BR */

	case obj.ARET:
		return OPBLR(2) /* RET */
	}

	c.ctxt.Diag("%v: bad brr %v", p, a)
	return 0
}

func (c *ctxt7) op0(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case ADRPS:
		return 0x6B<<25 | 5<<21 | 0x1F<<16 | 0x1F<<5

	case AERET:
		return 0x6B<<25 | 4<<21 | 0x1F<<16 | 0<<10 | 0x1F<<5

	case ANOOP:
		return SYSHINT(0)

	case AYIELD:
		return SYSHINT(1)

	case AWFE:
		return SYSHINT(2)

	case AWFI:
		return SYSHINT(3)

	case ASEV:
		return SYSHINT(4)

	case ASEVL:
		return SYSHINT(5)
	}

	c.ctxt.Diag("%v: bad op0 %v", p, a)
	return 0
}

/*
 * register offset
 */
func (c *ctxt7) opload(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case ALDAPR:
		return LDAPR(3)

	case ALDAPRW:
		return LDAPR(2)

	case ALDAPRB:
		return LDAPR(0)

	case ALDAPRH:
		return LDAPR(1)

	case ALDAR:
		return LDSTX(3, 1, 1, 0, 1) | 0x1F<<10

	case ALDARW:
		return LDSTX(2, 1, 1, 0, 1) | 0x1F<<10

	case ALDARB:
		return LDSTX(0, 1, 1, 0, 1) | 0x1F<<10

	case ALDARH:
		return LDSTX(1, 1, 1, 0, 1) | 0x1F<<10

	case ALDAXP:
		return LDSTX(3, 0, 1, 1, 1)

	case ALDAXPW:
		return LDSTX(2, 0, 1, 1, 1)

	case ALDAXR:
		return LDSTX(3, 0, 1, 0, 1) | 0x1F<<10

	case ALDAXRW:
		return LDSTX(2, 0, 1, 0, 1) | 0x1F<<10

	case ALDAXRB:
		return LDSTX(0, 0, 1, 0, 1) | 0x1F<<10

	case ALDAXRH:
		return LDSTX(1, 0, 1, 0, 1) | 0x1F<<10

	case ALDXR:
		return LDSTX(3, 0, 1, 0, 0) | 0x1F<<10

	case ALDXRB:
		return LDSTX(0, 0, 1, 0, 0) | 0x1F<<10

	case ALDXRH:
		return LDSTX(1, 0, 1, 0, 0) | 0x1F<<10

	case ALDXRW:
		return LDSTX(2, 0, 1, 0, 0) | 0x1F<<10

	case ALDXP:
		return LDSTX(3, 0, 1, 1, 0)

	case ALDXPW:
		return LDSTX(2, 0, 1, 1, 0)
	}

	c.ctxt.Diag("bad opload %v\n%v", a, p)
	return 0
}

func (c *ctxt7) opstore(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case ASTLR:
		return LDSTX(3, 1, 0, 0, 1) | 0x1F<<10

	case ASTLRB:
		return LDSTX(0, 1, 0, 0, 1) | 0x1F<<10

	case ASTLRH:
		return LDSTX(1, 1, 0, 0, 1) | 0x1F<<10

	case ASTLRW:
		return LDSTX(2, 1, 0, 0, 1) | 0x1F<<10

	case ASTLXP:
		return LDSTX(3, 0, 0, 1, 1)

	case ASTLXPW:
		return LDSTX(2, 0, 0, 1, 1)

	case ASTLXR:
		return LDSTX(3, 0, 0, 0, 1) | 0x1F<<10

	case ASTLXRB:
		return LDSTX(0, 0, 0, 0, 1) | 0x1F<<10

	case ASTLXRH:
		return LDSTX(1, 0, 0, 0, 1) | 0x1F<<10

	case ASTLXRW:
		return LDSTX(2, 0, 0, 0, 1) | 0x1F<<10

	case ASTXR:
		return LDSTX(3, 0, 0, 0, 0) | 0x1F<<10

	case ASTXRB:
		return LDSTX(0, 0, 0, 0, 0) | 0x1F<<10

	case ASTXRH:
		return LDSTX(1, 0, 0, 0, 0) | 0x1F<<10

	case ASTXP:
		return LDSTX(3, 0, 0, 1, 0)

	case ASTXPW:
		return LDSTX(2, 0, 0, 1, 0)

	case ASTXRW:
		return LDSTX(2, 0, 0, 0, 0) | 0x1F<<10
	}

	c.ctxt.Diag("bad opstore %v\n%v", a, p)
	return 0
}

/*
 * load/store register (scaled 12-bit unsigned immediate) C3.3.13
 *	these produce 64-bit values (when there's an option)
 */
func (c *ctxt7) olsr12u(p *obj.Prog, o uint32, v int32, rn, rt int16) uint32 {
	if v < 0 || v >= (1<<12) {
		c.ctxt.Diag("offset out of range: %d\n%v", v, p)
	}
	o |= uint32(v&0xFFF) << 10
	o |= uint32(rn&31) << 5
	o |= uint32(rt & 31)
	o |= 1 << 24
	return o
}

/*
 * load/store register (unscaled 9-bit signed immediate) C3.3.12
 */
func (c *ctxt7) olsr9s(p *obj.Prog, o uint32, v int32, rn, rt int16) uint32 {
	if v < -256 || v > 255 {
		c.ctxt.Diag("offset out of range: %d\n%v", v, p)
	}
	o |= uint32((v & 0x1FF) << 12)
	o |= uint32(rn&31) << 5
	o |= uint32(rt & 31)
	return o
}

// store(immediate)
// scaled 12-bit unsigned immediate offset.
// unscaled 9-bit signed immediate offset.
// pre/post-indexed store.
// and the 12-bit and 9-bit are distinguished in olsr12u and oslr9s.
func (c *ctxt7) opstr(p *obj.Prog, a obj.As) uint32 {
	enc := c.opldr(p, a)
	switch p.As {
	case AFMOVQ:
		enc = enc &^ (1 << 22)
	default:
		enc = LD2STR(enc)
	}
	return enc
}

// load(immediate)
// scaled 12-bit unsigned immediate offset.
// unscaled 9-bit signed immediate offset.
// pre/post-indexed load.
// and the 12-bit and 9-bit are distinguished in olsr12u and oslr9s.
func (c *ctxt7) opldr(p *obj.Prog, a obj.As) uint32 {
	switch a {
	case AMOVD:
		return LDSTR(3, 0, 1) /* simm9<<12 | Rn<<5 | Rt */

	case AMOVW:
		return LDSTR(2, 0, 2)

	case AMOVWU:
		return LDSTR(2, 0, 1)

	case AMOVH:
		return LDSTR(1, 0, 2)

	case AMOVHU:
		return LDSTR(1, 0, 1)

	case AMOVB:
		return LDSTR(0, 0, 2)

	case AMOVBU:
		return LDSTR(0, 0, 1)

	case AFMOVS, AVMOVS:
		return LDSTR(2, 1, 1)

	case AFMOVD, AVMOVD:
		return LDSTR(3, 1, 1)

	case AFMOVQ, AVMOVQ:
		return LDSTR(0, 1, 3)
	}

	c.ctxt.Diag("bad opldr %v\n%v", a, p)
	return 0
}

// olsxrr attaches register operands to a load/store opcode supplied in o.
// The result either encodes a load of r from (r1+r2) or a store of r to (r1+r2).
func (c *ctxt7) olsxrr(p *obj.Prog, o int32, r int, r1 int, r2 int) uint32 {
	o |= int32(r1&31) << 5
	o |= int32(r2&31) << 16
	o |= int32(r & 31)
	return uint32(o)
}

// opldrr returns the ARM64 opcode encoding corresponding to the obj.As opcode
// for load instruction with register offset.
// The offset register can be (Rn)(Rm.UXTW<<2) or (Rn)(Rm<<2) or (Rn)(Rm).
func (c *ctxt7) opldrr(p *obj.Prog, a obj.As, extension bool) uint32 {
	OptionS := uint32(0x1a)
	if extension {
		OptionS = uint32(0) // option value and S value have been encoded into p.From.Offset.
	}
	switch a {
	case AMOVD:
		return OptionS<<10 | 0x3<<21 | 0x1f<<27
	case AMOVW:
		return OptionS<<10 | 0x5<<21 | 0x17<<27
	case AMOVWU:
		return OptionS<<10 | 0x3<<21 | 0x17<<27
	case AMOVH:
		return OptionS<<10 | 0x5<<21 | 0x0f<<27
	case AMOVHU:
		return OptionS<<10 | 0x3<<21 | 0x0f<<27
	case AMOVB:
		return OptionS<<10 | 0x5<<21 | 0x07<<27
	case AMOVBU:
		return OptionS<<10 | 0x3<<21 | 0x07<<27
	case AFMOVS:
		return OptionS<<10 | 0x3<<21 | 0x17<<27 | 1<<26
	case AFMOVD:
		return OptionS<<10 | 0x3<<21 | 0x1f<<27 | 1<<26
	}
	c.ctxt.Diag("bad opldrr %v\n%v", a, p)
	return 0
}

// opstrr returns the ARM64 opcode encoding corresponding to the obj.As opcode
// for store instruction with register offset.
// The offset register can be (Rn)(Rm.UXTW<<2) or (Rn)(Rm<<2) or (Rn)(Rm).
func (c *ctxt7) opstrr(p *obj.Prog, a obj.As, extension bool) uint32 {
	OptionS := uint32(0x1a)
	if extension {
		OptionS = uint32(0) // option value and S value have been encoded into p.To.Offset.
	}
	switch a {
	case AMOVD:
		return OptionS<<10 | 0x1<<21 | 0x1f<<27
	case AMOVW, AMOVWU:
		return OptionS<<10 | 0x1<<21 | 0x17<<27
	case AMOVH, AMOVHU:
		return OptionS<<10 | 0x1<<21 | 0x0f<<27
	case AMOVB, AMOVBU:
		return OptionS<<10 | 0x1<<21 | 0x07<<27
	case AFMOVS:
		return OptionS<<10 | 0x1<<21 | 0x17<<27 | 1<<26
	case AFMOVD:
		return OptionS<<10 | 0x1<<21 | 0x1f<<27 | 1<<26
	}
	c.ctxt.Diag("bad opstrr %v\n%v", a, p)
	return 0
}

func (c *ctxt7) oaddi(p *obj.Prog, a obj.As, v int32, rd, rn int16) uint32 {
	op := c.opirr(p, a)

	if (v & 0xFFF000) != 0 {
		if v&0xFFF != 0 {
			c.ctxt.Diag("%v misuses oaddi", p)
		}
		v >>= 12
		op |= 1 << 22
	}

	op |= (uint32(v&0xFFF) << 10) | (uint32(rn&31) << 5) | uint32(rd&31)

	return op
}

func (c *ctxt7) oaddi12(p *obj.Prog, v int32, rd, rn int16) uint32 {
	if v < -4095 || v > 4095 {
		c.ctxt.Diag("%v is not a 12 bit immediate: %v", v, p)
		return 0
	}
	a := AADD
	if v < 0 {
		a = ASUB
		v = -v
	}
	return c.oaddi(p, a, v, rd, rn)
}

/*
 * load a literal value into dr
 */
func (c *ctxt7) omovlit(as obj.As, p *obj.Prog, a *obj.Addr, dr int) uint32 {
	var o1 int32
	if p.Pool == nil { /* not in literal pool */
		c.aclass(a)
		c.ctxt.Logf("omovlit add %d (%#x)\n", c.instoffset, uint64(c.instoffset))

		/* TODO: could be clever, and use general constant builder */
		o1 = int32(c.opirr(p, AADD))

		v := int32(c.instoffset)
		if v != 0 && (v&0xFFF) == 0 {
			v >>= 12
			o1 |= 1 << 22 /* shift, by 12 */
		}

		o1 |= ((v & 0xFFF) << 10) | (REGZERO & 31 << 5) | int32(dr&31)
	} else {
		fp, w := 0, 0
		switch as {
		case AFMOVS, AVMOVS:
			fp = 1
			w = 0 /* 32-bit SIMD/FP */

		case AFMOVD, AVMOVD:
			fp = 1
			w = 1 /* 64-bit SIMD/FP */

		case AVMOVQ:
			fp = 1
			w = 2 /* 128-bit SIMD/FP */

		case AMOVD:
			if p.Pool.As == ADWORD {
				w = 1 /* 64-bit */
			} else if p.Pool.To.Offset < 0 {
				w = 2 /* 32-bit, sign-extended to 64-bit */
			} else if p.Pool.To.Offset >= 0 {
				w = 0 /* 32-bit, zero-extended to 64-bit */
			} else {
				c.ctxt.Diag("invalid operand %v in %v", a, p)
			}

		case AMOVBU, AMOVHU, AMOVWU:
			w = 0 /* 32-bit, zero-extended to 64-bit */

		case AMOVB, AMOVH, AMOVW:
			w = 2 /* 32-bit, sign-extended to 64-bit */

		default:
			c.ctxt.Diag("invalid operation %v in %v", as, p)
		}

		v := int32(c.brdist(p, 0, 19, 2))
		o1 = (int32(w) << 30) | (int32(fp) << 26) | (3 << 27)
		o1 |= (v & 0x7FFFF) << 5
		o1 |= int32(dr & 31)
	}

	return uint32(o1)
}

// load a constant (MOVCON or BITCON) in a into rt
func (c *ctxt7) omovconst(as obj.As, p *obj.Prog, a *obj.Addr, rt int) (o1 uint32) {
	if cls := int(a.Class); (cls == C_BITCON || cls == C_ABCON || cls == C_ABCON0) && rt != REGZERO {
		// or $bitcon, REGZERO, rt. rt can't be ZR.
		mode := 64
		var as1 obj.As
		switch as {
		case AMOVW:
			as1 = AORRW
			mode = 32
		case AMOVD:
			as1 = AORR
		}
		o1 = c.opirr(p, as1)
		o1 |= bitconEncode(uint64(a.Offset), mode) | uint32(REGZERO&31)<<5 | uint32(rt&31)
		return o1
	}

	if as == AMOVW {
		d := uint32(a.Offset)
		s := movcon(int64(d))
		if s < 0 || 16*s >= 32 {
			d = ^d
			s = movcon(int64(d))
			if s < 0 || 16*s >= 32 {
				c.ctxt.Diag("impossible 32-bit move wide: %#x\n%v", uint32(a.Offset), p)
			}
			o1 = c.opirr(p, AMOVNW)
		} else {
			o1 = c.opirr(p, AMOVZW)
		}
		o1 |= MOVCONST(int64(d), s, rt)
	}
	if as == AMOVD {
		d := a.Offset
		s := movcon(d)
		if s < 0 || 16*s >= 64 {
			d = ^d
			s = movcon(d)
			if s < 0 || 16*s >= 64 {
				c.ctxt.Diag("impossible 64-bit move wide: %#x\n%v", uint64(a.Offset), p)
			}
			o1 = c.opirr(p, AMOVN)
		} else {
			o1 = c.opirr(p, AMOVZ)
		}
		o1 |= MOVCONST(d, s, rt)
	}
	return o1
}

// load a 32-bit/64-bit large constant (LCON or VCON) in a.Offset into rt
// put the instruction sequence in os and return the number of instructions.
func (c *ctxt7) omovlconst(as obj.As, p *obj.Prog, a *obj.Addr, rt int, os []uint32) (num uint8) {
	switch as {
	case AMOVW:
		d := uint32(a.Offset)
		// use MOVZW and MOVKW to load a constant to rt
		os[0] = c.opirr(p, AMOVZW)
		os[0] |= MOVCONST(int64(d), 0, rt)
		os[1] = c.opirr(p, AMOVKW)
		os[1] |= MOVCONST(int64(d), 1, rt)
		return 2

	case AMOVD:
		d := a.Offset
		dn := ^d
		var immh [4]uint64
		var i int
		zeroCount := int(0)
		negCount := int(0)
		for i = 0; i < 4; i++ {
			immh[i] = uint64((d >> uint(i*16)) & 0xffff)
			if immh[i] == 0 {
				zeroCount++
			} else if immh[i] == 0xffff {
				negCount++
			}
		}

		if zeroCount == 4 || negCount == 4 {
			c.ctxt.Diag("the immediate should be MOVCON: %v", p)
		}
		switch {
		case zeroCount == 3:
			// one MOVZ
			for i = 0; i < 4; i++ {
				if immh[i] != 0 {
					os[0] = c.opirr(p, AMOVZ)
					os[0] |= MOVCONST(d, i, rt)
					break
				}
			}
			return 1

		case negCount == 3:
			// one MOVN
			for i = 0; i < 4; i++ {
				if immh[i] != 0xffff {
					os[0] = c.opirr(p, AMOVN)
					os[0] |= MOVCONST(dn, i, rt)
					break
				}
			}
			return 1

		case zeroCount == 2:
			// one MOVZ and one MOVK
			for i = 0; i < 4; i++ {
				if immh[i] != 0 {
					os[0] = c.opirr(p, AMOVZ)
					os[0] |= MOVCONST(d, i, rt)
					i++
					break
				}
			}
			for ; i < 4; i++ {
				if immh[i] != 0 {
					os[1] = c.opirr(p, AMOVK)
					os[1] |= MOVCONST(d, i, rt)
				}
			}
			return 2

		case negCount == 2:
			// one MOVN and one MOVK
			for i = 0; i < 4; i++ {
				if immh[i] != 0xffff {
					os[0] = c.opirr(p, AMOVN)
					os[0] |= MOVCONST(dn, i, rt)
					i++
					break
				}
			}
			for ; i < 4; i++ {
				if immh[i] != 0xffff {
					os[1] = c.opirr(p, AMOVK)
					os[1] |= MOVCONST(d, i, rt)
				}
			}
			return 2

		case zeroCount == 1:
			// one MOVZ and two MOVKs
			for i = 0; i < 4; i++ {
				if immh[i] != 0 {
					os[0] = c.opirr(p, AMOVZ)
					os[0] |= MOVCONST(d, i, rt)
					i++
					break
				}
			}

			for j := 1; i < 4; i++ {
				if immh[i] != 0 {
					os[j] = c.opirr(p, AMOVK)
					os[j] |= MOVCONST(d, i, rt)
					j++
				}
			}
			return 3

		case negCount == 1:
			// one MOVN and two MOVKs
			for i = 0; i < 4; i++ {
				if immh[i] != 0xffff {
					os[0] = c.opirr(p, AMOVN)
					os[0] |= MOVCONST(dn, i, rt)
					i++
					break
				}
			}

			for j := 1; i < 4; i++ {
				if immh[i] != 0xffff {
					os[j] = c.opirr(p, AMOVK)
					os[j] |= MOVCONST(d, i, rt)
					j++
				}
			}
			return 3

		default:
			// one MOVZ and 3 MOVKs
			os[0] = c.opirr(p, AMOVZ)
			os[0] |= MOVCONST(d, 0, rt)
			for i = 1; i < 4; i++ {
				os[i] = c.opirr(p, AMOVK)
				os[i] |= MOVCONST(d, i, rt)
			}
			return 4
		}
	default:
		return 0
	}
}

func (c *ctxt7) opbfm(p *obj.Prog, a obj.As, r, s int64, rf, rt int16) uint32 {
	var b uint32
	o := c.opirr(p, a)
	if (o & (1 << 31)) == 0 {
		b = 32
	} else {
		b = 64
	}
	if r < 0 || uint32(r) >= b {
		c.ctxt.Diag("illegal bit number\n%v", p)
	}
	o |= (uint32(r) & 0x3F) << 16
	if s < 0 || uint32(s) >= b {
		c.ctxt.Diag("illegal bit number\n%v", p)
	}
	o |= (uint32(s) & 0x3F) << 10
	o |= (uint32(rf&31) << 5) | uint32(rt&31)
	return o
}

func (c *ctxt7) opextr(p *obj.Prog, a obj.As, v int64, rn, rm, rt int16) uint32 {
	var b uint32
	o := c.opirr(p, a)
	if (o & (1 << 31)) != 0 {
		b = 63
	} else {
		b = 31
	}
	if v < 0 || uint32(v) > b {
		c.ctxt.Diag("illegal bit number\n%v", p)
	}
	o |= uint32(v) << 10
	o |= uint32(rn&31) << 5
	o |= uint32(rm&31) << 16
	o |= uint32(rt & 31)
	return o
}

/* generate instruction encoding for ldp and stp series */
func (c *ctxt7) opldpstp(p *obj.Prog, o *Optab, vo int32, rbase, rl, rh int16, ldp uint32) uint32 {
	wback := false
	if o.scond == C_XPOST || o.scond == C_XPRE {
		wback = true
	}
	switch p.As {
	case ALDP, ALDPW, ALDPSW:
		c.checkUnpredictable(p, true, wback, p.From.Reg, p.To.Reg, int16(p.To.Offset))
	case ASTP, ASTPW:
		if wback {
			c.checkUnpredictable(p, false, true, p.To.Reg, p.From.Reg, int16(p.From.Offset))
		}
	case AFLDPD, AFLDPQ, AFLDPS:
		c.checkUnpredictable(p, true, false, p.From.Reg, p.To.Reg, int16(p.To.Offset))
	}
	var ret uint32
	// check offset
	switch p.As {
	case AFLDPQ, AFSTPQ:
		if vo < -1024 || vo > 1008 || vo%16 != 0 {
			c.ctxt.Diag("invalid offset %v\n", p)
		}
		vo /= 16
		ret = 2<<30 | 1<<26
	case AFLDPD, AFSTPD:
		if vo < -512 || vo > 504 || vo%8 != 0 {
			c.ctxt.Diag("invalid offset %v\n", p)
		}
		vo /= 8
		ret = 1<<30 | 1<<26
	case AFLDPS, AFSTPS:
		if vo < -256 || vo > 252 || vo%4 != 0 {
			c.ctxt.Diag("invalid offset %v\n", p)
		}
		vo /= 4
		ret = 1 << 26
	case ALDP, ASTP:
		if vo < -512 || vo > 504 || vo%8 != 0 {
			c.ctxt.Diag("invalid offset %v\n", p)
		}
		vo /= 8
		ret = 2 << 30
	case ALDPW, ASTPW:
		if vo < -256 || vo > 252 || vo%4 != 0 {
			c.ctxt.Diag("invalid offset %v\n", p)
		}
		vo /= 4
		ret = 0
	case ALDPSW:
		if vo < -256 || vo > 252 || vo%4 != 0 {
			c.ctxt.Diag("invalid offset %v\n", p)
		}
		vo /= 4
		ret = 1 << 30
	default:
		c.ctxt.Diag("invalid instruction %v\n", p)
	}
	// check register pair
	switch p.As {
	case AFLDPQ, AFLDPD, AFLDPS, AFSTPQ, AFSTPD, AFSTPS:
		if rl < REG_F0 || REG_F31 < rl || rh < REG_F0 || REG_F31 < rh {
			c.ctxt.Diag("invalid register pair %v\n", p)
		}
	case ALDP, ALDPW, ALDPSW:
		if rl < REG_R0 || REG_R31 < rl || rh < REG_R0 || REG_R31 < rh {
			c.ctxt.Diag("invalid register pair %v\n", p)
		}
	case ASTP, ASTPW:
		if rl < REG_R0 || REG_R31 < rl || rh < REG_R0 || REG_R31 < rh {
			c.ctxt.Diag("invalid register pair %v\n", p)
		}
	}
	// other conditional flag bits
	switch o.scond {
	case C_XPOST:
		ret |= 1 << 23
	case C_XPRE:
		ret |= 3 << 23
	default:
		ret |= 2 << 23
	}
	ret |= 5<<27 | (ldp&1)<<22 | uint32(vo&0x7f)<<15 | uint32(rh&31)<<10 | uint32(rbase&31)<<5 | uint32(rl&31)
	return ret
}

func (c *ctxt7) maskOpvldvst(p *obj.Prog, o1 uint32) uint32 {
	if p.As == AVLD1 || p.As == AVST1 {
		return o1
	}

	o1 &^= 0xf000 // mask out "opcode" field (bit 12-15)
	switch p.As {
	case AVLD1R, AVLD2R:
		o1 |= 0xC << 12
	case AVLD3R, AVLD4R:
		o1 |= 0xE << 12
	case AVLD2, AVST2:
		o1 |= 8 << 12
	case AVLD3, AVST3:
		o1 |= 4 << 12
	case AVLD4, AVST4:
	default:
		c.ctxt.Diag("unsupported instruction:%v\n", p.As)
	}
	return o1
}

/*
 * size in log2(bytes)
 */
func movesize(a obj.As) int {
	switch a {
	case AFMOVQ:
		return 4

	case AMOVD, AFMOVD:
		return 3

	case AMOVW, AMOVWU, AFMOVS:
		return 2

	case AMOVH, AMOVHU:
		return 1

	case AMOVB, AMOVBU:
		return 0

	default:
		return -1
	}
}

// rm is the Rm register value, o is the extension, amount is the left shift value.
func roff(rm int16, o uint32, amount int16) uint32 {
	return uint32(rm&31)<<16 | o<<13 | uint32(amount)<<10
}

// encRegShiftOrExt returns the encoding of shifted/extended register, Rx<<n and Rx.UXTW<<n, etc.
func (c *ctxt7) encRegShiftOrExt(p *obj.Prog, a *obj.Addr, r int16) uint32 {
	var num, rm int16
	num = (r >> 5) & 7
	rm = r & 31
	switch {
	case REG_UXTB <= r && r < REG_UXTH:
		return roff(rm, 0, num)
	case REG_UXTH <= r && r < REG_UXTW:
		return roff(rm, 1, num)
	case REG_UXTW <= r && r < REG_UXTX:
		if a.Type == obj.TYPE_MEM {
			if num == 0 {
				// According to the arm64 specification, for instructions MOVB, MOVBU and FMOVB,
				// the extension amount must be 0, encoded in "S" as 0 if omitted, or as 1 if present.
				// But in Go, we don't distinguish between Rn.UXTW and Rn.UXTW<<0, so we encode it as
				// that does not present. This makes no difference to the function of the instruction.
				// This is also true for extensions LSL, SXTW and SXTX.
				return roff(rm, 2, 2)
			} else {
				return roff(rm, 2, 6)
			}
		} else {
			return roff(rm, 2, num)
		}
	case REG_UXTX <= r && r < REG_SXTB:
		return roff(rm, 3, num)
	case REG_SXTB <= r && r < REG_SXTH:
		return roff(rm, 4, num)
	case REG_SXTH <= r && r < REG_SXTW:
		return roff(rm, 5, num)
	case REG_SXTW <= r && r < REG_SXTX:
		if a.Type == obj.TYPE_MEM {
			if num == 0 {
				return roff(rm, 6, 2)
			} else {
				return roff(rm, 6, 6)
			}
		} else {
			return roff(rm, 6, num)
		}
	case REG_SXTX <= r && r < REG_SVE_VECTOR_EXT:
		if a.Type == obj.TYPE_MEM {
			if num == 0 {
				return roff(rm, 7, 2)
			} else {
				return roff(rm, 7, 6)
			}
		} else {
			return roff(rm, 7, num)
		}
	case REG_LSL <= r && r < REG_ARNG:
		if a.Type == obj.TYPE_MEM { // (R1)(R2<<1)
			if num == 0 {
				return roff(rm, 3, 2)
			} else {
				return roff(rm, 3, 6)
			}
		} else if isADDWop(p.As) {
			return roff(rm, 2, num)
		}
		return roff(rm, 3, num)
	default:
		c.ctxt.Diag("unsupported register extension type.")
	}

	return 0
}

// pack returns the encoding of the "Q" field and two arrangement specifiers.
func pack(q uint32, arngA, arngB uint8) uint32 {
	return uint32(q)<<16 | uint32(arngA)<<8 | uint32(arngB)
}
