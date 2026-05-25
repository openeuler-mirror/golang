// Copyright (c) Huawei Technologies Co., Ltd. 2021-2021. All rights reserved.

//go:build goexperiment.memmoveopt

#include "textflag.h"

// See memmove Go doc for important implementation constraints.

/* Assumptions:
 *
 * ARMv8-a, AArch64, unaligned accesses.
 *
 */

#define dstin	R0
#define src	R1
#define count	R2
#define dst	R3
#define srcend	R4
#define dstend	R5
#define tmp2	R6
#define tmp3	R7
#define tmp3w   R7
#define A_l	R6
#define A_lw	R6
#define A_h	R7
#define A_hw	R7
#define B_l	R8
#define B_lw	R8
#define B_h	R9
#define C_l	R10
#define C_h	R11
#define D_l	R12
#define D_h	R13
#define E_l	src
#define E_h	count
#define F_l	srcend
#define F_h	dst
#define G_l	count
#define G_h	dst
#define tmp1	R14

#define A_q	F0
#define B_q	F1
#define C_q	F2
#define D_q	F3
#define E_q	F4
#define F_q	F5
#define G_q	F6
#define H_q	F7
#define I_q	F16
#define J_q	F17

#define A_v	v0
#define B_v	v1
#define C_v	v2
#define D_v	v3
#define E_v	v4
#define F_v	v5
#define G_v	v6
#define H_v	v7
#define I_v	v16
#define J_v	v17

// Copies are split into 3 main cases: small copies of up to 32 bytes, medium
// copies of up to 128 bytes, and large copies. The overhead of the overlap
// check is negligible since it is only required for large copies.
//
// Large copies use a software pipelined loop processing 64 bytes per iteration.
// The destination pointer is 16-byte aligned to minimize unaligned accesses.
// The loop tail is handled by always copying 64 bytes from the end.

#define MEMCPY_PREFETCH_LDR 640

// func memmove(to, from unsafe.Pointer, n uintptr)
TEXT runtime·memmove<ABIInternal>(SB), NOSPLIT|NOFRAME, $0-24
#ifndef GOEXPERIMENT_regabiargs
	MOVD	to+0(FP), R0
	MOVD	from+8(FP), R1
	MOVD	n+16(FP), R2
#endif
	CBZ	R2, copy0

	ADD	R1, R2, R4 // R4 points just past the last source byte
	ADD	R0, R2, R5 // R5 points just past the last destination byte

	// Small copies: 1..16 bytes
	CMP	$16, R2
	BLE	copy16

	// Large copies
	CMP	$32, R2
	BHI	copy32_128

	// Small copies: 17..32 bytes.
	LDP	(R1), (R6, R7)
	LDP	-16(R4), (R12, R13)
	STP	(R6, R7), (R0)
	STP	(R12, R13), -16(R5)
	RET

// Small copies: 1..16 bytes.
	PCALIGN	$32
copy16:
	CMP	$8, R2
	BLT	copy7
	MOVD	(R1), R6
	MOVD	-8(R4), R7
	MOVD	R6, (R0)
	MOVD	R7, -8(R5)
	RET

	PCALIGN	$32
copy7:
	TBZ	$2, R2, copy3
	MOVWU	(R1), R6
	MOVWU	-4(R4), R7
	MOVW	R6, (R0)
	MOVW	R7, -4(R5)
	RET

	PCALIGN	$32
copy3:
	TBZ	$1, R2, copy1
	MOVHU	(R1), R6
	MOVHU	-2(R4), R7
	MOVH	R6, (R0)
	MOVH	R7, -2(R5)
	RET

	PCALIGN	$32
copy1:
	MOVBU	(R1), R6
	MOVB	R6, (R0)
copy0:
	RET

// Medium copies: 33..128 bytes.
	PCALIGN $32
copy32_128:
	CMP	$128, R2
	BHI	copy_long
	FLDPQ (src), (A_q, B_q) 		// ldp	q0, q1, [x1]
	FLDPQ -32(srcend), (C_q, D_q) 	// ldp	q2, q3, [x4, #-32]
	CMP	$64, R2
	BHI	copy128
	FSTPQ (A_q, B_q), (dstin) 		// stp	q0, q1, [x0]
	FSTPQ (C_q, D_q), -32(dstend) 	// stp	q2, q3, [x5, #-32]
	RET

// Copy 65..128 bytes.
	PCALIGN $32
copy128:
	FLDPQ 32(src), (E_q, F_q)  		// ldp	q4, q5, [x1, #32]
	CMP	$96, R2
	BLS	copy96
	FLDPQ -64(srcend), (G_q, H_q) 	// ldp	q6, q7, [x4, #-64]
	FSTPQ (G_q, H_q), -64(dstend) 	// stp	q6, q7, [x5, #-64]

copy96:
	FSTPQ (A_q, B_q), (dstin)    	// stp	q0, q1, [x0]
	FSTPQ (E_q, F_q), 32(dstin)  	// stp	q4, q5, [x0, #32]
	FSTPQ (C_q, D_q), -32(dstend) 	// stp	q2, q3, [x5, #-32]
	RET

// Copy more than 128 bytes.
	PCALIGN $32
copy_long:
	// Backward check
	SUB src, dstin, tmp1
	CBZ tmp1, copy0
	CMP count, tmp1
	BCC L_copy_long_backward

    CMP $2048, count
    BHI L_copy_prefetch
	AND	$15, dstin, tmp1
	BIC	$15, dstin, dst
	FMOVQ.P 16(src), A_q			// ldr A_q, [src], #16
	SUB	tmp1, src, src
	ADD	tmp1, count, count	/* Count is now 16 too large.  */
	FLDPQ.P	64(src), (B_q, C_q)		// ldp B_q, C_q, [src], #64
	FLDPQ	-32(src), (D_q, E_q)	// ldp D_q, E_q, [src, #-32]
	FMOVQ	A_q, (dstin)			// str A_q, [dstin]
	ADD	$16, dst, dst
	/* Already loaded 64+16 bytes. Check if at
	   least 64 more bytes left */
	SUBS	$(64+64+16), count, count /* Test and readjust count.  */
	BLS	L_last64

L_loop64:
	FSTPQ.P	(B_q, C_q), 64(dst)		// stp B_q, C_q, [dst], #64
	FLDPQ.P	64(src), (B_q, C_q)		// ldp B_q, C_q, [src], #64
	FSTPQ (D_q, E_q), -32(dst)		// stp D_q, E_q, [dst, #-32]
	FLDPQ -32(src), (D_q, E_q)		// ldp D_q, E_q, [src, #-32]
	SUBS	$64, count, count
	BHI	L_loop64

	/* Write the last full set of 64 bytes.  The remainder is at most 64
	   bytes, so it is safe to always copy 64 bytes from the end even if
	   there is just 1 byte left.  */
L_last64:
	FLDPQ	-64(srcend), (F_q, G_q)	// ldp F_q, G_q, [srcend, #-64]
	FSTPQ	(B_q, C_q), (dst)		// stp B_q, C_q, [dst]
	FLDPQ	-32(srcend), (B_q, C_q)	// ldp B_q, C_q, [srcend, #-32]
	FSTPQ	(D_q, E_q), 32(dst)		// stp D_q, E_q, [dst, #32]
	FSTPQ	(F_q, G_q), -64(dstend)	// stp F_q, G_q, [dstend, #-64]
	FSTPQ   (B_q, C_q), -32(dstend)	// stp B_q, C_q, [dstend, #-32]
	RET

L_copy_prefetch:
	FMOVQ.P 16(src), A_q	  		// ldr A_q, [src], #16
	AND	$15, src, tmp1
	BIC	$15, src, src
	FLDPQ.P	64(src), (B_q, C_q)		// ldp B_q, C_q, [src], #64
	SUB	tmp1, dstin, dst
	ADD	tmp1, count, count
	ADD	$16, dst, dst
	AND	$15, dst, tmp1
	FLDPQ	-32(src), (D_q, E_q)	// ldp D_q, E_q, [src, #-32]
	FMOVQ A_q, (dstin)	         	// str A_q, [dstin]

	/* Already loaded 64+16 bytes. Also included 64 bytes for next step. */
	SUB	$(64+64+16), count, count
	CBNZ	tmp1, L_dst_unaligned

	// bypass alignment NOPS with jump
	B L_loop64_prefetch
	PCALIGN $32
L_loop64_prefetch:
	PRFM	MEMCPY_PREFETCH_LDR(src), PLDL1STRM
	FSTPQ.P	(B_q, C_q), 64(dst)		// stp B_q, C_q, [dst], #64
	FLDPQ.P	64(src), (B_q, C_q)		// ldp B_q, C_q, [src], #64
	FSTPQ (D_q, E_q), -32(dst)		// stp D_q, E_q, [dst, #-32]
	FLDPQ -32(src), (D_q, E_q)		// ldp D_q, E_q, [src, #-32]
	SUB	$64, count, count
	CMP	$MEMCPY_PREFETCH_LDR, count
	BHI	L_loop64_prefetch
	B L_loop64

	PCALIGN	$32
L_copy_long_backward:
	FMOVQ	-16(srcend), E_q		// ldr E_q, [srcend, #-16]
	AND	$15, srcend, tmp1
	SUB	tmp1, srcend, srcend
	SUB	tmp1, count, count
	FLDPQ	-32(srcend), (A_q, B_q)	// ldp A_q, B_q, [srcend, #-32]
	FMOVQ	E_q, -16(dstend)		// str E_q, [dstend, #-16]
	FLDPQ.W	-64(srcend), (C_q, D_q)	// ldp C_q, D_q, [srcend, #-64]!
	SUB	tmp1, dstend, dstend
	/* Already loaded 32+32 bytes. Check if at
	   least 64 more bytes left */
	SUBS	$(32+32+64), count, count
	BLS	L_copy64_from_start

L_loop64_backward:
	FSTPQ	(A_q, B_q), -32(dstend)	// stp A_q, B_q, [dstend, #-32]
	FLDPQ	-32(srcend), (A_q, B_q)	// ldp A_q, B_q, [srcend, #-32]
	FSTPQ.W	(C_q, D_q), -64(dstend)	// stp C_q, D_q, [dstend, #-64]!
	FLDPQ.W	-64(srcend), (C_q, D_q)	// ldp C_q, D_q, [srcend, #-64]!
	SUBS	$64, count, count
	BHI	L_loop64_backward

L_copy64_from_start:
	FLDPQ	32(src), (E_q, F_q)		// ldp E_q, F_q, [src, #32]
	FSTPQ	(A_q, B_q), -32(dstend)	// stp A_q, B_q, [dstend, #-32]
	FLDPQ	(src), (A_q, B_q)		// ldp A_q, B_q, [src]
	FSTPQ	(C_q, D_q), -64(dstend)	// stp C_q, D_q, [dstend, #-64]
	FSTPQ	(E_q, F_q), 32(dstin)	// stp E_q, F_q, [dstin, #32]
	FSTPQ	(A_q, B_q), (dstin)		// stp A_q, B_q, [dstin]
	RET

	PCALIGN $32
L_dst_unaligned:
	/* For the unaligned store case the code loads two
	   aligned chunks and then merges them using ext
	   instruction. This can be up to 30% faster than
	   the the simple unaligned store access.

	   Current state: tmp1 = dst % 16; B_q, C_q, D_q, E_q
	   contains data yet to be stored. src and dst points
	   to next-to-be-processed data. A_q contains
	   data already stored before, count = bytes left to
	   be load decremented by 64.

	   The control is passed here if at least 64 bytes left
	   to be loaded. The code does two aligned loads and then
	   extracts (16-tmp1) bytes from the first register and
	   tmp1 bytes from the next register forming the value
	   for the aligned store.

	   As ext instruction can only have it's index encoded
	   as immediate. 15 code chunks process each possible
	   index value. Computed goto is used to reach the
	   required code. */

	/* Store the 16 bytes to dst and align dst for further
	   operations, several bytes will be stored at this
	   address once more */

	FLDPQ.P 32(src), (F_q, G_q)    	// ldp	F_q, G_q, [src], #32
	FSTPQ.P (B_q, C_q), 32(dst)	 	// stp	B_q, C_q, [dst], #32
	BIC	$15, dst, dst
	SUB	$32, count, count

	/* Align hot loop inside macro */
	// bypass alignment NOPS with jump
	B	L_dst_unaligned_tramp
	PCALIGN $32
/* To make the loop in each chunk 16-bytes aligned.
   Loop begins with fourth instructions in EXT_CHUNK macro,
   but since we don't want to execute NOP instructions on the first
   iteration of the loop, we will blase the aligment here.
   The number of instructions from here to the begining of the loop is 8
   (5 in trampoline + 3 VEXT). */
L_dst_unaligned_tramp:
	/* tmp1 contains index which is used to calculate jump offset: (4byte*16instr = 64)
	   there is no index 0 so we need to substract 64 explicitely */ 
	LSL $4, tmp1, tmp3				// tmp3 == tmp1 * 16
	ADR L_dst_unaligned_table, tmp2
	SUB $64, tmp2, tmp2 			// tmp2 == L_dst_unaligned_table - 64
	ADD tmp3.SXTH<<2, tmp2, tmp2	
	JMP (tmp2)						// tmp2 == L_dst_unaligned_table + (index - 1) * 64

#define EXT_CHUNK(shft)                                                                     \
	VEXT $(16-shft), V3.B16, V2.B16, V0.B16; /*ext     A_v.16b, C_v.16b, D_v.16b, 16-shft*/ \
	VEXT $(16-shft), V4.B16, V3.B16, V1.B16; /*ext     B_v.16b, D_v.16b, E_v.16b, 16-shft*/ \
	VEXT $(16-shft), V5.B16, V4.B16, V7.B16; /*ext     H_v.16b, E_v.16b, F_v.16b, 16-shft*/ \
	/* loop start here: */                                                                  \
	FSTPQ.P (A_q, B_q), 64(dst);             /*stp     A_q, B_q, [dst], #64*/               \
	PRFM    MEMCPY_PREFETCH_LDR(src), PLDL1STRM;                                            \
	FLDPQ.P 64(src), (C_q, D_q);             /*ldp     C_q, D_q, [src], #64*/               \
	VEXT $(16-shft), V6.B16, V5.B16, V16.B16;/*ext     I_v.16b, F_v.16b, G_v.16b, 16-shft*/ \
	FSTPQ (H_q, I_q), -32(dst);              /*stp     H_q, I_q, [dst, -32]*/               \
	VEXT $(16-shft), V2.B16, V6.B16, V0.B16; /*ext     A_v.16b, G_v.16b, C_v.16b, 16-shft*/ \
	VEXT $(16-shft), V3.B16, V2.B16, V1.B16; /*ext     B_v.16b, C_v.16b, D_v.16b, 16-shft*/ \
	FLDPQ -32(src), (F_q, G_q);              /*ldp     F_q, G_q, [src, -32]*/               \
	VEXT $(16-shft), V5.B16, V3.B16, V7.B16; /*ext     H_v.16b, D_v.16b, F_v.16b, 16-shft*/ \
	SUBS    $64, count, count;                                                              \
	BGE -10(PC);                                                                            \
	VEXT $(16-shft), V6.B16, V5.B16, V16.B16;/*ext     I_v.16b, F_v.16b, G_v.16b, 16-shft*/ \
	JMP	L_dst_unaligned_tail;

L_dst_unaligned_table:
	EXT_CHUNK(1)
	EXT_CHUNK(2)
	EXT_CHUNK(3)
	EXT_CHUNK(4)
	EXT_CHUNK(5)
	EXT_CHUNK(6)
	EXT_CHUNK(7)
	EXT_CHUNK(8)
	EXT_CHUNK(9)
	EXT_CHUNK(10)
	EXT_CHUNK(11)
	EXT_CHUNK(12)
	EXT_CHUNK(13)
	EXT_CHUNK(14)
	EXT_CHUNK(15)

	PCALIGN $32
L_dst_unaligned_tail:
	FLDPQ -64(srcend), (C_q, D_q)// ldp	C_q, D_q, [srcend, -64]
	FLDPQ -32(srcend), (E_q, F_q)// ldp	E_q, F_q, [srcend, -32]：
	FSTPQ.P (A_q, B_q), 32(dst)  // stp	A_q, B_q, [dst], #32
	FSTPQ.P (H_q, I_q), 16(dst)  // stp	H_q, I_q, [dst], #16
	WORD $0x3cae6866             // str	G_q, [dst, tmp1]
	FSTPQ (C_q, D_q), -64(dstend)// stp	C_q, D_q, [dstend, -64]
	FSTPQ (E_q, F_q), -32(dstend)// stp	E_q, F_q, [dstend, -32]
	RET
