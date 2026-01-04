// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file encapsulates some of the odd characteristics of the ARM64
// instruction set, to minimize its interaction with the core of the
// assembler.

package arch

import (
	"cmd/internal/obj"
	"cmd/internal/obj/arm64"
	"errors"
)

var arm64LS = map[string]uint8{
	"P": arm64.C_XPOST,
	"W": arm64.C_XPRE,
}

var arm64Jump = map[string]bool{
	"B":     true,
	"BL":    true,
	"BEQ":   true,
	"BNE":   true,
	"BCS":   true,
	"BHS":   true,
	"BCC":   true,
	"BLO":   true,
	"BMI":   true,
	"BPL":   true,
	"BVS":   true,
	"BVC":   true,
	"BHI":   true,
	"BLS":   true,
	"BGE":   true,
	"BLT":   true,
	"BGT":   true,
	"BLE":   true,
	"CALL":  true,
	"CBZ":   true,
	"CBZW":  true,
	"CBNZ":  true,
	"CBNZW": true,
	"JMP":   true,
	"TBNZ":  true,
	"TBZ":   true,

	// ADR isn't really a jump, but it takes a PC or label reference,
	// which needs to patched like a jump.
	"ADR":  true,
	"ADRP": true,
}

func jumpArm64(word string) bool {
	return arm64Jump[word]
}

var arm64SpecialOperand map[string]arm64.SpecialOperand

// GetARM64SpecialOperand returns the internal representation of a special operand.
func GetARM64SpecialOperand(name string) arm64.SpecialOperand {
	if arm64SpecialOperand == nil {
		// Generate the mapping automatically when the first time the function is called.
		arm64SpecialOperand = map[string]arm64.SpecialOperand{}
		for opd := arm64.SPOP_BEGIN; opd < arm64.SPOP_END; opd++ {
			arm64SpecialOperand[opd.String()] = opd
		}

		// Handle some special cases.
		specialMapping := map[string]arm64.SpecialOperand{
			// The internal representation of CS(CC) and HS(LO) are the same.
			"CS": arm64.SPOP_HS,
			"CC": arm64.SPOP_LO,
		}
		for s, opd := range specialMapping {
			arm64SpecialOperand[s] = opd
		}
	}
	if opd, ok := arm64SpecialOperand[name]; ok {
		return opd
	}
	return arm64.SPOP_END
}

func IsARM64ZRegister(reg int) bool {
	return reg >= arm64.REG_Z0 && reg <= arm64.REG_Z31
}

func IsARM64PRegister(reg int) bool {
	return reg >= arm64.REG_P0 && reg <= arm64.REG_P15
}

// IsARM64ADR reports whether the op (as defined by an arm64.A* constant) is
// one of the comparison instructions that require special handling.
func IsARM64ADR(op obj.As) bool {
	switch op {
	case arm64.AADR, arm64.AADRP:
		return true
	}
	return false
}

// IsARM64CMP reports whether the op (as defined by an arm64.A* constant) is
// one of the comparison instructions that require special handling.
func IsARM64CMP(op obj.As) bool {
	switch op {
	case arm64.ACMN, arm64.ACMP, arm64.ATST,
		arm64.ACMNW, arm64.ACMPW, arm64.ATSTW,
		arm64.AFCMPS, arm64.AFCMPD,
		arm64.AFCMPES, arm64.AFCMPED:
		return true
	}
	return false
}

// IsARM64STLXR reports whether the op (as defined by an arm64.A*
// constant) is one of the STLXR-like instructions that require special
// handling.
func IsARM64STLXR(op obj.As) bool {
	switch op {
	case arm64.ASTLXRB, arm64.ASTLXRH, arm64.ASTLXRW, arm64.ASTLXR,
		arm64.ASTXRB, arm64.ASTXRH, arm64.ASTXRW, arm64.ASTXR,
		arm64.ASTXP, arm64.ASTXPW, arm64.ASTLXP, arm64.ASTLXPW:
		return true
	}
	// LDADDx/SWPx/CASx atomic instructions
	return arm64.IsAtomicInstruction(op)
}

// IsARM64TBL reports whether the op (as defined by an arm64.A*
// constant) is one of the TBL-like instructions and one of its
// inputs does not fit into prog.Reg, so require special handling.
func IsARM64TBL(op obj.As) bool {
	switch op {
	case arm64.AVTBL, arm64.AVTBX, arm64.AVMOVQ:
		return true
	}
	return false
}

// IsARM64CASP reports whether the op (as defined by an arm64.A*
// constant) is one of the CASP-like instructions, and its 2nd
// destination is a register pair that require special handling.
func IsARM64CASP(op obj.As) bool {
	switch op {
	case arm64.ACASPD, arm64.ACASPW:
		return true
	}
	return false
}

// ARM64Suffix handles the special suffix for the ARM64.
// It returns a boolean to indicate success; failure means
// cond was unrecognized.
func ARM64Suffix(prog *obj.Prog, cond string) bool {
	if cond == "" {
		return true
	}
	bits, ok := parseARM64Suffix(cond)
	if !ok {
		return false
	}
	prog.Scond = bits
	return true
}

// parseARM64Suffix parses the suffix attached to an ARM64 instruction.
// The input is a single string consisting of period-separated condition
// codes, such as ".P.W". An initial period is ignored.
func parseARM64Suffix(cond string) (uint8, bool) {
	if cond == "" {
		return 0, true
	}
	return parseARMCondition(cond, arm64LS, nil)
}

func arm64RegisterNumber(name string, n int16) (int16, bool) {
	switch name {
	case "F":
		if 0 <= n && n <= 31 {
			return arm64.REG_F0 + n, true
		}
	case "P":
		if 0 <= n && n <= 15 {
			return arm64.REG_P0 + n, true
		}
	case "R":
		if 0 <= n && n <= 30 { // not 31
			return arm64.REG_R0 + n, true
		}
	case "V":
		if 0 <= n && n <= 31 {
			return arm64.REG_V0 + n, true
		}
	case "Z":
		if 0 <= n && n <= 31 {
			return arm64.REG_Z0 + n, true
		}
	}
	return 0, false
}

// ARM64RegisterShift constructs an ARM64 register with shift operation.
func ARM64RegisterShift(reg, op, count int16) (int64, error) {
	// the base register of shift operations must be general register.
	if reg > arm64.REG_R31 || reg < arm64.REG_R0 {
		return 0, errors.New("invalid register for shift operation")
	}
	return int64(reg&31)<<16 | int64(op)<<22 | int64(uint16(count)), nil
}

// ARM64RegisterExtension constructs an ARM64 register with extension or arrangement.
func ARM64RegisterExtension(a *obj.Addr, ext string, reg, num int16, isAmount, isIndex bool) error {
	if reg <= arm64.REG_R31 && reg >= arm64.REG_R0 {
		if !isAmount {
			return errors.New("invalid register extension")
		}
		if num < 0 || num > 7 {
			return errors.New("index shift amount is out of range")
		}
		Rnum := int16(num)<<5 + int16(reg&31)
		switch ext {
		case "UXTB":
			if a.Type == obj.TYPE_MEM {
				return errors.New("invalid shift for the register offset addressing mode")
			}
			a.Reg = arm64.REG_UXTB + Rnum
		case "UXTH":
			if a.Type == obj.TYPE_MEM {
				return errors.New("invalid shift for the register offset addressing mode")
			}
			a.Reg = arm64.REG_UXTH + Rnum
		case "UXTW":
			// effective address of memory is a base register value and an offset register value.
			if a.Type == obj.TYPE_MEM {
				a.Index = arm64.REG_UXTW + Rnum
			} else {
				a.Reg = arm64.REG_UXTW + Rnum
			}
		case "UXTX":
			if a.Type == obj.TYPE_MEM {
				return errors.New("invalid shift for the register offset addressing mode")
			}
			a.Reg = arm64.REG_UXTX + Rnum
		case "SXTB":
			if a.Type == obj.TYPE_MEM {
				return errors.New("invalid shift for the register offset addressing mode")
			}
			a.Reg = arm64.REG_SXTB + Rnum
		case "SXTH":
			if a.Type == obj.TYPE_MEM {
				return errors.New("invalid shift for the register offset addressing mode")
			}
			a.Reg = arm64.REG_SXTH + Rnum
		case "SXTW":
			if a.Type == obj.TYPE_MEM {
				a.Index = arm64.REG_SXTW + Rnum
			} else {
				a.Reg = arm64.REG_SXTW + Rnum
			}
		case "SXTX":
			if a.Type == obj.TYPE_MEM {
				a.Index = arm64.REG_SXTX + Rnum
			} else {
				a.Reg = arm64.REG_SXTX + Rnum
			}
		case "LSL":
			a.Index = arm64.REG_LSL + Rnum
		default:
			return errors.New("unsupported general register extension type: " + ext)

		}
	} else if reg <= arm64.REG_V31 && reg >= arm64.REG_V0 {
		switch ext {
		case "B8":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_8B & 15) << 5)
		case "B16":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_16B & 15) << 5)
		case "H4":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_4H & 15) << 5)
		case "H8":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_8H & 15) << 5)
		case "S2":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_2S & 15) << 5)
		case "S4":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_4S & 15) << 5)
		case "D1":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_1D & 15) << 5)
		case "D2":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_2D & 15) << 5)
		case "Q1":
			if isIndex {
				return errors.New("invalid register extension")
			}
			a.Reg = arm64.REG_ARNG + (reg & 31) + ((arm64.ARNG_1Q & 15) << 5)
		case "B":
			if !isIndex {
				return nil
			}
			a.Reg = arm64.REG_ELEM + (reg & 31) + ((arm64.ARNG_B & 15) << 5)
			a.Index = num
		case "H":
			if !isIndex {
				return nil
			}
			a.Reg = arm64.REG_ELEM + (reg & 31) + ((arm64.ARNG_H & 15) << 5)
			a.Index = num
		case "S":
			if !isIndex {
				return nil
			}
			a.Reg = arm64.REG_ELEM + (reg & 31) + ((arm64.ARNG_S & 15) << 5)
			a.Index = num
		case "D":
			if !isIndex {
				return nil
			}
			a.Reg = arm64.REG_ELEM + (reg & 31) + ((arm64.ARNG_D & 15) << 5)
			a.Index = num
		default:
			return errors.New("unsupported simd register extension type: " + ext)
		}
	} else if arm64.REG_SVE_VECTOR <= reg && reg < arm64.REG_SVE_VECTOR_INDEX {
		switch ext {
		case "LSL":
			if a.Type == obj.TYPE_MEM {
				a.Index = arm64.REG_SVE_VECTOR_LSL + (reg & 0x1ff) + num<<9
			} else {
				a.Reg = arm64.REG_SVE_VECTOR_LSL + (reg & 0x1ff) + num<<9
			}
		case "UXTW":
			if a.Type == obj.TYPE_MEM {
				a.Index = arm64.REG_SVE_VECTOR_UXTW + (reg & 0x1ff) + num<<9
			} else {
				a.Reg = arm64.REG_SVE_VECTOR_UXTW + (reg & 0x1ff) + num<<9
			}
		case "SXTW":
			if a.Type == obj.TYPE_MEM {
				a.Index = arm64.REG_SVE_VECTOR_SXTW + (reg & 0x1ff) + num<<9
			} else {
				a.Reg = arm64.REG_SVE_VECTOR_SXTW + (reg & 0x1ff) + num<<9
			}
		default:
			return errors.New("unsupported SVE register extension type: " + ext)
		}
	} else if arm64.REG_Z0 <= reg && reg <= arm64.REG_Z31 {
		switch ext {
		case "B":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_VECTOR_INDEX + (reg & 31) + (arm64.ARNG_B << 5)
			a.Index = num
		case "H":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_VECTOR_INDEX + (reg & 31) + (arm64.ARNG_H << 5)
			a.Index = num
		case "S":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_VECTOR_INDEX + (reg & 31) + (arm64.ARNG_S << 5)
			a.Index = num
		case "D":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_VECTOR_INDEX + (reg & 31) + (arm64.ARNG_D << 5)
			a.Index = num
		case "Q":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_VECTOR_INDEX + (reg & 31) + (arm64.ARNG_Q << 5)
			a.Index = num
		default:
			return errors.New("unsupported SVE register extension type: " + ext)
		}
	} else if arm64.REG_P0 <= reg && reg <= arm64.REG_P15 {
		switch ext {
		case "B":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_PREDICATE + (reg & 15) + (arm64.ARNG_B << 5)
			a.Index = num
		case "H":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_PREDICATE + (reg & 15) + (arm64.ARNG_H << 5)
			a.Index = num
		case "S":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_PREDICATE + (reg & 15) + (arm64.ARNG_S << 5)
			a.Index = num
		case "D":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_PREDICATE + (reg & 15) + (arm64.ARNG_D << 5)
			a.Index = num
		case "Q":
			if !isIndex {
				return errors.New("invalid SVE register extension")
			}
			a.Reg = arm64.REG_SVE_PREDICATE + (reg & 15) + (arm64.ARNG_Q << 5)
			a.Index = num
		default:
			return errors.New("unsupported SVE register extension type: " + ext)
		}
	} else {
		return errors.New("invalid register and extension combination")
	}
	return nil
}

// ARM64RegisterArrangement constructs an ARM64 vector register arrangement.
func ARM64RegisterArrangement(reg int16, name, arng string) (int64, error) {
	var curQ, curSize uint16
	if name[0] != 'V' {
		return 0, errors.New("expect V0 through V31; found: " + name)
	}
	if reg < 0 {
		return 0, errors.New("invalid register number: " + name)
	}
	switch arng {
	case "B8":
		curSize = 0
		curQ = 0
	case "B16":
		curSize = 0
		curQ = 1
	case "H4":
		curSize = 1
		curQ = 0
	case "H8":
		curSize = 1
		curQ = 1
	case "S2":
		curSize = 2
		curQ = 0
	case "S4":
		curSize = 2
		curQ = 1
	case "D1":
		curSize = 3
		curQ = 0
	case "D2":
		curSize = 3
		curQ = 1
	default:
		return 0, errors.New("invalid arrangement in ARM64 register list")
	}
	return (int64(curQ) & 1 << 30) | (int64(curSize&3) << 10), nil
}

// ARM64RegisterListOffset generates offset encoding according to AArch64 specification.
func ARM64RegisterListOffset(a *obj.Addr, name string, firstReg, regCnt int, specifier int64) error {
	offset := int64(firstReg)
	switch name[0] {
	case 'Z': // [Zt1.B, Zt2.B]
		if regCnt < 1 || regCnt > 4 {
			return errors.New("invalid register number in ARM64 register list")
		}
		// |11 - 9|  8  -  5  |  4 - 0 |
		// regCnt | specifier | firstReg
		offset |= int64(regCnt)<<9 | (specifier&15)<<5
		// For assembler printing, the information of scalable vector register with datasize is recorded in a.Reg.
		a.Offset = offset
		a.Reg = arm64.REG_SVE_VECTOR + int16(firstReg&31) + int16(specifier&15)<<5
		return nil
	case 'P': // [Pt1.B, Pt2.B]
		if regCnt < 1 || regCnt > 4 {
			return errors.New("invalid register number in ARM64 register list")
		}
		// |11 - 9|  8  -  5  |  4 - 0 |
		// regCnt | specifier | firstReg
		offset |= int64(regCnt)<<9 | (specifier&15)<<5
		// For assembler printing, the information of scalable vector register with datasize is recorded in a.Reg.
		a.Offset = offset
		a.Reg = arm64.REG_SVE_PREDICATE + int16(firstReg&15) + int16(specifier&15)<<5
		return nil
	case 'V': // [Vt1.B8, Vt2.B8]
		switch regCnt {
		case 1:
			offset |= 0x7 << 12
		case 2:
			offset |= 0xa << 12
		case 3:
			offset |= 0x6 << 12
		case 4:
			offset |= 0x2 << 12
		default:
			return errors.New("invalid register numbers in ARM64 register list")
		}
		offset |= specifier
		// arm64 uses the 60th bit to differentiate from other archs
		// For more details, refer to: obj/arm64/list7.go
		offset |= 1 << 60
		a.Offset = offset
		return nil
	default:
		return errors.New("invalid ARM64 register list")
	}
}

// ARM64RegisterDatasize parses an ARM64 scalable vector register data size.
func ARM64RegisterDatasize(datasize string) (int64, error) {
	var size int64
	switch datasize {
	case "B":
		size = arm64.ARNG_B
	case "H":
		size = arm64.ARNG_H
	case "S":
		size = arm64.ARNG_S
	case "D":
		size = arm64.ARNG_D
	case "Q":
		size = arm64.ARNG_Q
	default:
		return 0, errors.New("unsupported data size: " + datasize)
	}
	return size, nil
}

// ARM64RegisterScalable parses an ARM64 scalable vector/predicate registers with data size (e.g. Zn.B or Pn.B).
func ARM64RegisterScalable(ext, name string, reg int16) (int16, error) {
	var res int16
	size, err := ARM64RegisterDatasize(ext)
	if err != nil {
		return res, err
	}
	switch name[0] {
	case 'Z':
		res = arm64.REG_SVE_VECTOR + reg&31 + int16(size&15)<<5
	case 'P':
		res = arm64.REG_SVE_PREDICATE + reg&31 + int16(size&15)<<5
	default:
		return res, errors.New("invalid scalable vector/predicate registers")
	}
	return res, nil
}

// ARM64GoverningPredicateRegister parses an ARM64 governing scalable predicate register (e.g. Pg/M or Pg/Z).
func ARM64GoverningPredicateRegister(a *obj.Addr, reg int16, predication string) error {
	switch predication {
	case "M":
		a.Reg = arm64.REG_SVE_PREDICATE_M + (reg & 15)
	case "Z":
		a.Reg = arm64.REG_SVE_PREDICATE_Z + (reg & 15)
	default:
		return errors.New("invalid predication")
	}
	return nil
}
