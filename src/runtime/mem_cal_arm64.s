// Copyright (c) 2026 Huawei Technologies Co., Ltd.
// memmory calculation for arm64 processor.

//go:build arm64 && (goexperiment.pagenum)

#include "textflag.h"

TEXT ·memMinSizeCalKp(SB), NOSPLIT, $0-24
    MOVD arr+0(FP), R0
    MOVD len+8(FP), R1

    ZAND $1, Z0.D, Z0.D
    ZAND $2, Z0.D, Z0.D
    MOVD $0, R2

loop:
    WHILELE R1, R2, P0.H
    BEQ done
    ZLD1H (R0)(R2<<1), P0/Z, [Z1.H]
    ZADD Z1.H, Z0.H, P0/M, Z0.H
    ZINCH R2
    B loop

done:
    PTRUE P0.H
    ZUADDV Z0.H, P0, V2.D2
    VMOV V2.D[0], R0
    MOVD $11541, R1
    UDIV R1, R0, R0
    MOVD R0, ret+16(FP)
    RET


TEXT ·checkMultPage(SB), NOSPLIT, $0-25
    MOVD    pages+0(FP), R0
    MOVD    idx+8(FP), R1
    MOVD    size+16(FP), R2

    MOVD    $0, R3
    ZDUP    R2, Z1.B

loop:
    WHILELE R3, R1, P0.B
    BEQ     ret_true

    ZLD1B   (R0)(R3), P0/Z, [Z0.B]
    ZCMPGT  Z1.B, Z0.B, P0/Z, P1.B
    PTEST   P1.B, P0
    BNE     ret_false

    ZINCB   R3
    B       loop

ret_true:
    MOVD    $1, R0
    MOVD    R0, ret+24(FP)
    RET

ret_false:
    MOVD    $0, R0
    MOVD    R0, ret+24(FP)
    RET


TEXT ·findMinSize(SB), NOSPLIT, $0-32
    MOVD    class+0(FP), R0
    MOVD    size+8(FP), R1
    MOVD    min+16(FP), R2

    MOVD    $0, R3
    ZDUP    R2, Z1.H

loop:
    WHILELT R1, R3, P0.H
    BEQ     not_found

    ZLD1H   (R0)(R3<<1), P0/Z, [Z0.H]
    ZCMPEQ  Z0.H, Z1.H, P0/Z, P1.H
    BNE     get_index

    ZINCH   R3
    B       loop

get_index:
    PFIRST  P1.B, P0, P1.B
    ZINDEX  $1, R3, Z2.H
    ZLASTB  Z2.H, P1, R0
    MOVD    R0, ret+24(FP)
    RET

not_found:
    MOVD    $-1, R0
    MOVD    R0, ret+24(FP)
    RET
