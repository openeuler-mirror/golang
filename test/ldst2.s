// asmcheck -d=aarch64ldst=all

    TEXT	Load1(SB), $0-48
// arm64:`LDP\t16\(R0\), \(R5, R3\)`
    MOVD 16(R0), R5
    MOVD 24(R0), R3
// arm64:`MOVD\t24\(R0\), R6`
    MOVD 24(R0), R6 // can merge when prev read src
    RET

    TEXT	Load2(SB), $0-48
// arm64:`LDP\t16\(R0\), \(R5, R6\)`
    MOVD 16(R0), R5
// arm64:`MOVD\tR0, R2`
    MOVD R0, R2
    MOVD 24(R0), R6 // can merge when prev read src ptr
    RET

    TEXT	Load3(SB), $0-48
    MOVD 16(R0), R5
    MOVD R1, 24(R0)
// arm64:`MOVD\t24\(R0\), R6`
    MOVD 24(R0), R6 // do not merge when prev write src
    RET

    TEXT	Load4(SB), $0-48
    MOVD 16(R0), R5
    SUB  R1, R2, R0
// arm64:`MOVD\t24\(R0\), R6`
    MOVD 24(R0), R6 // do not merge when prev write src ptr
    RET

    TEXT	Load5(SB), $0-48
    MOVD 16(R0), R5
    MOVD R6, R1
// arm64:`MOVD\t24\(R0\), R6`
    MOVD 24(R0), R6 // do not merge when prev read dst

    MOVD 56(R0), R5
    MOVD 88(R6), R1
// arm64:`MOVD\t64\(R0\), R6`
    MOVD 64(R0), R6 // do not merge when prev read dst ptr
    RET

    TEXT	Load6(SB), $0-48
    MOVD 16(R0), R5
    MOVD 0(R1), R6
// arm64:`MOVD\t24\(R0\), R6`
    MOVD 24(R0), R6 // do not merge when prev write dst
    RET

    TEXT	Load7(SB), $0-48
    MOVD 0(R0), R0
// arm64:`MOVD\t8\(R0\), R1`
    MOVD 8(R0), R1 // do not merge when prev change src ptr
    RET

    TEXT	Load8(SB), $0-48
    MOVD 0(R0), R1
// arm64:`MOVD\t8\(R0\), R0`
    MOVD 8(R0), R0  // do not merge when this write src ptr
    RET

    TEXT	Store1(SB), $0-48
// arm64:`STP\t\(R0, R1\), 16\(R2\)`
    MOVD R0, 16(R2)
    MOVD R0, R5
    MOVD R1, R6
    MOVD R1, 24(R2)  // can merge when prev read src
    RET

    TEXT	Store2(SB), $0-48
    MOVD R0, 16(R2)
    MOVD R5, R1
// arm64:`MOVD\tR1, 24\(R2\)`
    MOVD R1, 24(R2) // could merge and put stp here, but currently merging only to first instruction
    RET

    TEXT	Store3(SB), $0-48
    MOVD R0, 16(R2)
    MOVD 24(R2), R6
// arm64:`MOVD\tR1, 24\(R2\)`
    MOVD R1, 24(R2) // do not merge when prev read dst
    RET

    TEXT	Store4(SB), $0-48
// arm64:`STP\t\(R0, R1\), 16\(R2\)`
    MOVD R0, 16(R2)
    MOVD R2, R6
    MOVD R1, 24(R2) // can merge when prev read dst ptr
    RET

    TEXT	Store5(SB), $0-48
// arm64:`STP\t\(R0, R5\), 16\(R2\)`
    MOVD R0, 16(R2)
    MOVD R5, 24(R2) // dead store, but that's another pass' problem
    MOVD R1, 24(R2) // do not merge when prev write dst
    RET

    TEXT	Store5_1(SB), $0-48
// arm64:`MOVD\tR0, 16\(R2\)`
    MOVD R0, 16(R2)
// arm64:`MOVW\tR5, 28\(R2\)`
    MOVW R5, 28(R2) // dead store, but that's another pass' problem
// arm64:`MOVD\tR1, 24\(R2\)`
    MOVD R1, 24(R2) // do not merge when prev write dst
    RET

    TEXT	Store6(SB), $0-48
    MOVD R0, 16(R2)
    MOVD R5, R2
// arm64:`MOVD\tR1, 24\(R2\)`
    MOVD R1, 24(R2) // do not merge when prev write dst ptr
    RET

    TEXT	Store7(SB), $0-48
// arm64:`STP\t\(R0, R1\), \(R0\)`
    MOVD R0, 0(R0)
    MOVD R1, 8(R0) // can merge when prev read src
    RET

    TEXT	Store8(SB), $0-48
// arm64:`STP\t\(R1, R0\), \(R0\)`
    MOVD R1, 0(R0)
    MOVD R0, 8(R0) // can merge when this read src
    RET

    TEXT	Store9(SB), $0-48
// arm64:`MOVD\tR0, 16\(R2\)`
    MOVD R0, 16(R2)
// arm64:`MOVD\tR3, 48\(R1\)`
    MOVD R3, 48(R1)
// arm64:`MOVD\tR1, 24\(R2\)`
    MOVD R1, 24(R2) // not sure if 48(R1) may alias 24(R2)

// arm64:`MOVD\tR4, 16\(R6\)`
    MOVD R4, 16(R6)
// arm64:`MOVD\tR7, 48\(R4\)`
    MOVD R7, 48(R4)
// arm64:`MOVD\tR5, 24\(R6\)`
    MOVD R5, 24(R6) // not sure if 48(R4) may alias 24(R6)
    RET
