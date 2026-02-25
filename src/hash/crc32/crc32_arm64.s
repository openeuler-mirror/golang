// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

// castagnoliUpdate updates the non-inverted crc with the given data.

// func castagnoliUpdate(crc uint32, p []byte) uint32
TEXT ·castagnoliUpdate(SB),NOSPLIT,$0-36
	MOVWU	crc+0(FP), R9  // CRC value
	MOVD	p+8(FP), R13  // data pointer
	MOVD	p_len+16(FP), R11  // len(p)

	CMP $1024, R11
	BLT update

	MOVD $0xa741c1bf, R1
	MOVD $0xe417f38a, R2
	MOVD $0xdd7e3b0c, R3
	MOVD $0x8f158014, R4
	MOVD $0xdaece73e, R5

	//将常量移动到向量寄存器
	VMOV R1, V5.D[0]
	VMOV R2, V6.D[0]
	VMOV R3, V7.D[0]
	VMOV R4, V8.D[0]
	VMOV R5, V9.D[0]

	JMP large_loop

large_loop:
	MOVD $0, R1
	MOVD $0, R2
	MOVD $0, R3
	MOVD $0, R4
	MOVD $0, R5
	MOVD.P  8(R13), R6
	CRC32CX R6, R9
    
	ADD $168, R13, R7
	ADD $336, R13, R8
	ADD $504, R13, R10
	ADD $672, R13, R12
	ADD $840, R13, R14
	
	//加载数据
	LDP.P 16(R13), (R15, R16)
	LDP.P 16(R7),  (R17, R19)
	LDP.P 16(R8),  (R20, R21)
	LDP.P 16(R10), (R22, R23)
	LDP.P 16(R12), (R24, R25)
	LDP.P 16(R14), (R26, R27)

	//主循环 - 4次重复
	MOVD $4, R0

loop_4x:
    CRC32CX R15, R9
	CRC32CX R17, R1
	CRC32CX R20, R2
	CRC32CX R22, R3
	CRC32CX R24, R4
	CRC32CX R26, R5

	LDP.P 16(R13), (R15, R22)
	LDP.P 16(R7),  (R17, R24)
	LDP.P 16(R8),  (R20, R26)

	CRC32CX R16, R9
	CRC32CX R19, R1
	CRC32CX R21, R2
	CRC32CX R23, R3
	CRC32CX R25, R4
	CRC32CX R27, R5

	LDP.P 16(R10), (R16, R23)
	LDP.P 16(R12), (R19, R25)
	LDP.P 16(R14), (R21, R27)

	CRC32CX R15, R9
	CRC32CX R17, R1
	CRC32CX R20, R2
	CRC32CX R16, R3
	CRC32CX R19, R4
	CRC32CX R21, R5

	LDP.P 16(R13), (R15, R16)
	LDP.P 16(R7),  (R17, R19)
	LDP.P 16(R8),  (R20, R21)

    CRC32CX R22, R9
	CRC32CX R24, R1
	CRC32CX R26, R2
	CRC32CX R23, R3
	CRC32CX R25, R4
	CRC32CX R27, R5

	LDP.P 16(R10), (R22, R23)
	LDP.P 16(R12), (R24, R25)
	LDP.P 16(R14), (R26, R27)

	SUB $1, R0
	CBNZ R0, loop_4x

	CRC32CX R15, R9
	CRC32CX R17, R1
	CRC32CX R20, R2
	CRC32CX R22, R3
	CRC32CX R24, R4
	CRC32CX R26, R5

	LDP.P 16(R13), (R15, R22)
	LDP.P 16(R7),  (R17, R24)
	LDP.P 16(R8),  (R20, R26)

	CRC32CX R16, R9
	CRC32CX R19, R1
	CRC32CX R21, R2
	CRC32CX R23, R3
	CRC32CX R25, R4
	CRC32CX R27, R5

	LDP.P 16(R10), (R16, R23)
	LDP.P 16(R12), (R19, R25)
	LDP.P 16(R14), (R21, R27)

	CRC32CX R15, R9
	CRC32CX R17, R1
	CRC32CX R20, R2
	CRC32CX R16, R3
	CRC32CX R19, R4
	CRC32CX R21, R5

	MOVD.P 8(R13), R15
	MOVD.P 8(R7),  R17
	MOVD.P 8(R8),  R20

	CRC32CX R22, R9
	CRC32CX R24, R1
	CRC32CX R26, R2
	CRC32CX R23, R3
	CRC32CX R25, R4
	CRC32CX R27, R5

	MOVD.P 8(R10), R22
	MOVD.P 8(R12), R24
	MOVD.P 8(R14), R26

	CRC32CX R15, R9
	CRC32CX R17, R1
	CRC32CX R20, R2
	CRC32CX R22, R3
	CRC32CX R24, R4
	CRC32CX R26, R5

	MOVD.P  8(R14), R26
	CRC32CX R26, R5

	MOVD    R14, R13

	VMOV R9, V0.D[0]
	VMOV R1, V1.D[0]
	VMOV R2, V2.D[0]
	VMOV R3, V3.D[0]
	VMOV R4, V4.D[0]

    VPMULL V0.D1, V5.D1, V0.Q1
	VPMULL V1.D1, V6.D1, V1.Q1
	VPMULL V2.D1, V7.D1, V2.Q1
	VPMULL V3.D1, V8.D1, V3.Q1
	VPMULL V4.D1, V9.D1, V4.Q1

	VMOV V0.D[0], R9
	VMOV V1.D[0], R1
	VMOV V2.D[0], R2
	VMOV V3.D[0], R3
	VMOV V4.D[0], R4

	CRC32CX R9, R0, R9
	CRC32CX R1, R0, R1
	CRC32CX R2, R0, R2
	CRC32CX R3, R0, R3
	CRC32CX R4, R0, R4

	EOR R1, R9, R9
	EOR R3, R2, R2
	EOR R5, R4, R4
	EOR R4, R2, R2
	EOR R2, R9, R9

	SUB  $1024, R11

	CMP $1024, R11
	BLT update

	JMP large_loop

update:
	CMP	$16, R11
	BLT	less_than_16
	LDP.P	16(R13), (R8, R10)
	CRC32CX	R8, R9
	CRC32CX	R10, R9
	SUB	$16, R11

	JMP	update

less_than_16:
	TBZ	$3, R11, less_than_8

	MOVD.P	8(R13), R10
	CRC32CX	R10, R9

less_than_8:
	TBZ	$2, R11, less_than_4

	MOVWU.P	4(R13), R10
	CRC32CW	R10, R9

less_than_4:
	TBZ	$1, R11, less_than_2

	MOVHU.P	2(R13), R10
	CRC32CH	R10, R9

less_than_2:
	TBZ	$0, R11, done

	MOVBU	(R13), R10
	CRC32CB	R10, R9

done:
	MOVWU	R9, ret+32(FP)
	RET

// ieeeUpdate updates the non-inverted crc with the given data.

// func ieeeUpdate(crc uint32, p []byte) uint32
TEXT ·ieeeUpdate(SB),NOSPLIT,$0-36
	MOVWU	crc+0(FP), R9  // CRC value
	MOVD	p+8(FP), R13  // data pointer
	MOVD	p_len+16(FP), R11  // len(p)

update:
	CMP	$16, R11
	BLT	less_than_16
	LDP.P	16(R13), (R8, R10)
	CRC32X	R8, R9
	CRC32X	R10, R9
	SUB	$16, R11

	JMP	update

less_than_16:
	TBZ $3, R11, less_than_8

	MOVD.P	8(R13), R10
	CRC32X	R10, R9

less_than_8:
	TBZ	$2, R11, less_than_4

	MOVWU.P	4(R13), R10
	CRC32W	R10, R9

less_than_4:
	TBZ	$1, R11, less_than_2

	MOVHU.P	2(R13), R10
	CRC32H	R10, R9

less_than_2:
	TBZ	$0, R11, done

	MOVBU	(R13), R10
	CRC32B	R10, R9

done:
	MOVWU	R9, ret+32(FP)
	RET
