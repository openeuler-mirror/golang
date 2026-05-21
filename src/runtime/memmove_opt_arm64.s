// Copyright (c) 2025 Huawei Technologies Co., Ltd.
// Optimized memmove function for Huawei Kunpeng processor.

//go:build goexperiment.memmoveopt

#include "textflag.h"

// See memmove Go doc for important implementation constraints.

/* Assumptions:
 *
 * ARMv8-a, AArch64, unaligned accesses.
 *
 */

#define dst_in	R0
#define src	    R1
#define cnt	    R2

#define dst 	R3
#define src_end	R4
#define dst_end	R5
#define tmp1	R9

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
#define K_q	F18
#define L_q	F19
#define M_q F20
#define N_q F21
#define O_q F22
#define P_q F23

#define PREFETCH_OFFSET 640

// func memmove(to, from unsafe.Pointer, n uintptr)
TEXT runtime·memmove<ABIInternal>(SB), NOSPLIT|NOFRAME, $0-24
#ifndef GOEXPERIMENT_regabiargs
	MOVD	to+0(FP), R0
	MOVD	from+8(FP), R1
	MOVD	n+16(FP), R2
#endif
	CBZ	R2, copy0
// Small copies: 1..16 bytes
	CMP	$16, R2
	BLE	handle_LE_16
	CMP	$32, R2
	BHI	handle_33_128

// Small copies: 17..32 bytes.
	LDP	(R1), (R6, R7)
	ADD	R1, R2, R4
	LDP	-16(R4), (R12, R13)
	STP	(R6, R7), (R0)
	ADD	R0, R2, R5
	STP	(R12, R13), -16(R5)
	RET

// Small copies: 1..16 bytes.
handle_LE_16:
	ADD	R1, R2, R4
	ADD	R0, R2, R5
	CMP	$8, R2
	BLT	copy7
	MOVD	(R1), R6
	MOVD	-8(R4), R7
	MOVD	R6, (R0)
	MOVD	R7, -8(R5)
	RET

copy7:
	TBZ	$2, R2, copy3
	MOVWU	(R1), R6
	MOVWU	-4(R4), R7
	MOVW	R6, (R0)
	MOVW	R7, -4(R5)
	RET

copy3:
	TBZ	$1, R2, copy1
	MOVHU	(R1), R6
	MOVHU	-2(R4), R7
	MOVH	R6, (R0)
	MOVH	R7, -2(R5)
	RET

copy1:
	MOVBU	(R1), R6
	MOVB	R6, (R0)
copy0:
	RET

// Medium copies: 33..128 bytes.
handle_33_128:
	ADD	R1, R2, R4
	ADD	R0, R2, R5
	CMP	$128, R2
	BHI	Lhandle_128_more
	CMP $64, cnt
	BHI Lhandle_65_128

	LDP	(R1), (R6, R7)
	LDP	16(R1), (R8, R9)
	LDP	-32(R4), (R10, R11)
	LDP	-16(R4), (R12, R13)
	STP	(R6, R7), (R0)
	STP	(R8, R9), 16(R0)
	STP	(R10, R11), -32(R5)
	STP	(R12, R13), -16(R5)
	RET

Lhandle_65_128:
	SUB R1, R0, R9      
	CBZ R9, Lreturn
	CMP R9, R2
	BHI Lmove_65_128_comreg

Lcopy_64_128:
	FLDPQ (src), (A_q, B_q)
	FLDPQ 32(R1), (C_q, D_q)
	FLDPQ -32(R4), (E_q, F_q)
	FLDPQ -64(R4), (G_q, H_q)
	FSTPQ (A_q, B_q), (R0)
	FSTPQ (C_q, D_q), 32(R0)
	FSTPQ (E_q, F_q), -32(R5)
	FSTPQ (G_q, H_q), -64(R5)
	RET

// Large copies: 128 bytes more
Lhandle_128_more:
	CMP  $192, R2
	BHI Lis_cpy
    
Lmove_128_192:
	FLDPQ (R1), (A_q, B_q)
	FLDPQ 32(R1), (C_q, D_q)
	FLDPQ -128(R4), (E_q, F_q)
	FLDPQ -96(R4), (G_q, H_q)
	FLDPQ -64(R4), (I_q, J_q)
	FLDPQ -32(R4), (K_q, L_q)
	FSTPQ (A_q, B_q), (R0)
	FSTPQ (C_q, D_q), 32(R0)
	FSTPQ (E_q, F_q), -128(R5)
	FSTPQ (G_q, H_q), -96(R5)
	FSTPQ (I_q, J_q), -64(R5)
	FSTPQ (K_q, L_q), -32(R5)
	RET

Lis_cpy:
	SUB R1, R0, R9      
	CBZ R9, Lreturn
	CMP R9, R2
	BLS Lcopy_192_more

	PCALIGN	$32
Lmove_192_more:
	CMP $256, cnt
	BLS Lmove_long_backward_comreg

	FMOVQ	-16(src_end), E_q
	AND	$15, src_end, tmp1
	SUB	tmp1, src_end, src_end
	SUB	tmp1, cnt, cnt
	FLDPQ	-32(src_end), (A_q, B_q)
	FMOVQ	E_q, -16(dst_end)
	FLDPQ.W	-64(src_end), (C_q, D_q)
	SUB	tmp1, dst_end, dst_end
	/* Already loaded 32+32 bytes. Check if at
	   least 64 more bytes left */
	SUBS	$(32+32+64), cnt, cnt
	BLS	L_copy64_from_start

L_loop64_backward:
	FSTPQ	(A_q, B_q), -32(dst_end)	// stp A_q, B_q, [dst_end, #-32]
	FLDPQ	-32(src_end), (A_q, B_q)	// ldp A_q, B_q, [src_end, #-32]
	FSTPQ.W	(C_q, D_q), -64(dst_end)	// stp C_q, D_q, [dst_end, #-64]!
	FLDPQ.W	-64(src_end), (C_q, D_q)	// ldp C_q, D_q, [src_end, #-64]!
	SUBS	$64, cnt, cnt
	BHI	L_loop64_backward

L_copy64_from_start:
	FLDPQ	32(src), (E_q, F_q)		// ldp E_q, F_q, [src, #32]
	FSTPQ	(A_q, B_q), -32(dst_end)	// stp A_q, B_q, [dst_end, #-32]
	FLDPQ	(src), (A_q, B_q)		// ldp A_q, B_q, [src]
	FSTPQ	(C_q, D_q), -64(dst_end)	// stp C_q, D_q, [dst_end, #-64]
	FSTPQ	(E_q, F_q), 32(dst_in)	// stp E_q, F_q, [dst_in, #32]
	FSTPQ	(A_q, B_q), (dst_in)		// stp A_q, B_q, [dst_in]
Lreturn:
	RET

Lcopy_192_more:
	FLDPQ (R1), (E_q, F_q)
	FLDPQ 32(R1), (G_q, H_q)
	ADD $0, R0, R3       
	AND $63, R1, R9
	CBZ R9, Lalready_align_64
	BIC $63, R1, R1 
	SUB R9, R0, R3 
	ADD R2, R9, R2  

Lalready_align_64:
	FLDPQ 64(R1), (A_q, B_q)
	FLDPQ 96(R1), (C_q, D_q)

	FLDPQ -128(R4), (I_q, J_q)
	FLDPQ -96(R4), (K_q, L_q)
	FLDPQ -64(R4), (M_q, N_q)
	FLDPQ -32(R4), (O_q, P_q)

	FSTPQ (E_q, F_q), (R0)
	FSTPQ (G_q, H_q), 32(R0)
	SUBS  $(128+64+64), R2, R2
	BLS Ltail128_align_64

#ifdef GOARM64_RPRFM
	CMP $2048, cnt
	BLS Lloop128_align_64
	ADD $PREFETCH_OFFSET, src, R10
	SUB $PREFETCH_OFFSET, cnt, R11
	RPRFM (R10), R11, PLDSTRM
#endif

Lloop128_align_64:
	FLDPQ 128(R1), (E_q, F_q)
	FSTPQ (A_q, B_q), 64(R3)
	FLDPQ 160(R1), (G_q, H_q)
	FSTPQ (C_q, D_q), 96(R3)
	FLDPQ 192(R1), (A_q, B_q)
	FSTPQ (E_q, F_q), 128(R3)
	FLDPQ 224(R1), (C_q, D_q)
	FSTPQ (G_q, H_q), 160(R3)
	ADD $128, R1, R1   
	ADD $128, R3, R3   
	SUBS $128, R2, R2
	BHS Lloop128_align_64

Ltail128_align_64:
	FSTPQ (A_q, B_q), 64(R3)
	FSTPQ (C_q, D_q), 96(R3)

	FSTPQ (I_q, J_q), -128(R5)
	FSTPQ (K_q, L_q), -96(R5)
	FSTPQ (M_q, N_q), -64(R5)
	FSTPQ (O_q, P_q), -32(R5)
	RET
	
	// backward move 65..128 bytes.
Lmove_65_128_comreg:
	LDP	(R1), (R6, R7)
	LDP	16(R1), (R8, R9)
	LDP	32(R1), (R14, R15)
	LDP	48(R1), (R16, R17)
	LDP	-32(R4), (R10, R11)
	LDP	-16(R4), (R12, R13)
	CMP	$96, R2
	BLS	Lmove96_comreg
	LDP	-64(R4), (R2, R3)
	LDP	-48(R4), (R1, R4)
	STP	(R2, R3), -64(R5)
	STP	(R1, R4), -48(R5)

Lmove96_comreg:
	STP	(R6, R7), (R0)
	STP	(R8, R9), 16(R0)
	STP	(R14, R15), 32(R0)
	STP	(R16, R17), 48(R0)
	STP	(R10, R11), -32(R5)
	STP	(R12, R13), -16(R5)
	RET

Lmove_long_backward_comreg:
	LDP	-16(R4), (R12, R13)
	AND	$15, R8, R14
	SUB	R14, R4, R4
	SUB	R14, R2, R2
	LDP	-16(R4), (R6, R7)
	STP	(R12, R13), -16(R5)
	LDP	-32(R4), (R8, R9)
	LDP	-48(R4), (R10, R11)
	LDP.W	-64(R4), (R12, R13)
	SUB	R14, R5, R5
	SUBS	$128, R2, R2
	BLS	Lmove_64_from_start_comreg

Lmove_loop64_backward_comreg:
	STP	(R6, R7), -16(R5)
	LDP	-16(R4), (R6, R7)
	STP	(R8, R9), -32(R5)
	LDP	-32(R4), (R8, R9)
	STP	(R10, R11), -48(R5)
	LDP	-48(R4), (R10, R11)
	STP.W	(R12, R13), -64(R5)
	LDP.W	-64(R4), (R12, R13)
	SUBS	$64, R2, R2
	BHI	Lmove_loop64_backward_comreg

	// Write the last iteration and copy 64 bytes from the start.
Lmove_64_from_start_comreg:
	LDP	48(R1), (R2, R3)
	STP	(R6, R7), -16(R5)
	LDP	32(R1), (R6, R7)
	STP	(R8, R9), -32(R5)
	LDP	16(R1), (R8, R9)
	STP	(R10, R11), -48(R5)
	LDP	(R1), (R10, R11)
	STP	(R12, R13), -64(R5)
	STP	(R2, R3), 48(R0)
	STP	(R6, R7), 32(R0)
	STP	(R8, R9), 16(R0)
	STP	(R10, R11), (R0)
	RET
