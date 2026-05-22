// asmcheck -d=aarch64ldst=all -I="../../../../pkg/include"

#include "funcdata.h"

    TEXT	SP1(SB), $0-48
// arm64:`STP\t\(R0, R1\), t0\+8\(SP\)`
    MOVD R0, t0+8(SP)
    MOVD R1, t1+16(SP) // can merge
    RET

    TEXT	SP2(SB), $0-48
// arm64:`MOVD\tR0, t0\+8\(SP\)`
    MOVD R0, t0+8(SP)
    SUB $16, RSP, RSP
// arm64:`MOVD\tR1, t1\+16\(SP\)`
    MOVD R1, t1+16(SP) // Should not be merged.
    RET

    TEXT	FP1(SB), $0-48
// arm64:`STP\t\(R0, R1\), t0\+8\(FP\)`
    MOVD R0, t0+8(FP)
    MOVD R1, t1+16(FP) // can merge
    RET

    TEXT	FP2(SB), $0-48
// arm64:`MOVD\tR0, t0\+8\(FP\)`
    MOVD R0, t0+8(FP)
    SUB $16, RSP, RSP
// arm64:`MOVD\tR1, t1\+16\(FP\)`
    MOVD R1, t1+16(FP) // Should not be merged.
    RET

    TEXT	FP3(SB), $0-48
// arm64:`STP\t\(R0, R1\), t0\+8\(FP\)`
    MOVD R0, t0+8(FP)
// arm64:`STP\t\(R2, R3\), s0\+8\(SP\)`
    MOVD R2, s0+8(SP)
    MOVD R3, s1+16(SP) // can merge
    MOVD R1, t1+16(FP) // We assume the FP and SP based references are disjoint, at least expect it for the compiler generated Progs.
    RET

    TEXT	PCDATA1(SB), $0-48
// arm64:`LDP\t\(R27\), \(R1, R2\)`
    MOVD 0(R27), R1
    MOVD R3, R4
    MOVD 8(R27), R2 // can merge and generate PCDATA for R27 usage
    RET

    TEXT	PCDATA2(SB), $0-48
// arm64:`MOVD\t\(R0\), R1`
    MOVD 0(R0), R1
    PCDATA $PCDATA_UnsafePoint, $-2
// arm64:`MOVD\t8\(R0\), R2`
    MOVD 8(R0), R2 // do not merge across PCDATA

// arm64:`MOVD\t\(R4\), R5`
    MOVD 0(R4), R5
    PCDATA $PCDATA_InlTreeIndex, $0
// arm64:`MOVD\t8\(R4\), R6`
    MOVD 8(R4), R6 // do not merge across PCDATA
    RET

DATA lpm(SB)/8, $0
DATA lpm+8(SB)/8, $0
DATA lpm+16(SB)/8, $0
DATA lpm+24(SB)/8, $0

    TEXT	FUNCDATA1(SB), $0-48
// arm64:`MOVD\t\(R0\), R1`
    MOVD 0(R0), R1
    FUNCDATA $FUNCDATA_LocalsPointerMaps, lpm(SB)
// arm64:`MOVD\t8\(R0\), R2`
    MOVD 8(R0), R2 // do not merge across funcdata

// arm64:`MOVD\t\(R4\), R5`
    MOVD 0(R4), R5
    FUNCDATA $FUNCDATA_ArgInfo, lpm(SB)
// arm64:`MOVD\t8\(R4\), R6`
    MOVD 8(R4), R6 // do not merge across funcdata
    RET

    TEXT	READDATA1(SB), $0-48
// arm64:`LDP\tlpm\(SB\), \(R1, R2\)`
    MOVD lpm(SB), R1
// arm64:`LDP\tlpm\+16\(SB\), \(R3, R4\)`
    MOVD lpm+24(SB), R4
    MOVD lpm+16(SB), R3
    MOVD lpm+8(SB), R2
    RET

