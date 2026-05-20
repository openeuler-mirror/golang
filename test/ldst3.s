// asmcheck -d=aarch64ldst=all

    TEXT	NotMerge1(SB), $0-48
// arm64:`MOVD\t\(R0\), R1`
    MOVD 0(R0), R1
    RET
// arm64:`MOVD\t8\(R0\), R2`
    MOVD 8(R0), R2 // do not merge
    RET

    TEXT	NotMerge2(SB), $0-48
// arm64:`MOVD\t\(R0\), R1`
    MOVD 0(R0), R1
    CALL NotMerge1(SB)
// arm64:`MOVD\t8\(R0\), R2`
    MOVD 8(R0), R2 // do not merge
    RET

    TEXT	NotMerge3(SB), $0-48
// arm64:`MOVD\t\(R0\), R1`
    MOVD 0(R0), R1
    JMP NotMerge1(SB)
// arm64:`MOVD\t8\(R0\), R2`
    MOVD 8(R0), R2 // do not merge
    RET

    TEXT	NotMerge4(SB), $0-48
// arm64:`MOVD\t\(R0\), R1`
    MOVD 0(R0), R1
    MOVD.W R30, -16(R4)
// arm64:`MOVD\t8\(R0\), R2`
    MOVD 8(R0), R2 // do not merge
    RET


    TEXT	CanMerge1(SB), $0-48
// arm64:`STP\t\(R1, R2\), \(R0\)`
    MOVD R1, 0(R0)
    NOP
    NOP
    MOVD R2, 8(R0) // can merge with NOP

// arm64:`LDP\t8\(R4\), \(R6, R5\)`
    MOVD 16(R4), R5
    NOP
    NOP
    MOVD 8(R4), R6 // can merge with NOP
    RET

    TEXT	CanMerge2(SB), $0-48
// arm64:`STP\t\(R1, R2\), \(R0\)`
    MOVD R1, 0(R0)
    ADD R1, R2, R3
    MOVD R2, 8(R0) // can merge with ADD (not affect)

    MOVD 16(R4), R5
    ADD R1, R2, R6
// arm64:`MOVD\t8\(R4\), R6`
    MOVD 8(R4), R6 // not merge with ADD (affect dst)
    RET

    TEXT	CanMerge3(SB), $0-48
// arm64:`STP\t\(R2, R1\), -16\(R0\)`
    MOVD R1, -8(R0)
    MOVD R2, -16(R0) // can merge, negative offset
    RET

    TEXT	Distance1(SB), $0-48
// arm64:`LDP\t16\(R2\), \(R0, R1\)`
    MOVD 16(R2), R0
    MOVD R3, 0(R2)
    MOVD 24(R2), R1  // can merge, the load does not alias

// arm64:`MOVD\t16\(R4\), R5`
    MOVD 16(R4), R5
    MOVD R6, 8(R4)
// arm64:`MOVD\t24\(R4\), R7`
    MOVD 24(R4), R7 // Could merge with somewhat more specific check:
                    // currently the algorithm stops when it sees possible
                    // aliasing pair's memory location 8(R4) for the 16(R4).
                    // We could do better by only disallowing 8(R4) pair there and
                    // going ahead to look for possible pair 24(R4).

// arm64:`LDP\t88\(R4\), \(R5, R7\)`
    MOVD 88(R4), R5
    MOVD R6, 104(R4)
    MOVD 96(R4), R7 // can merge
    RET

    TEXT	Distance2(SB), $0-48
// arm64:`STP\t\(R0, R2\), 16\(R1\)`
    MOVD R0, 16(R1)
    MOVD 0(R1), R3
    MOVD R2, 24(R1) // can merge, the load does not alias

    MOVD R4, 16(R6)
    MOVD 8(R6), R7
// arm64:`MOVD\tR5, 24\(R6\)`
    MOVD R5, 24(R6) // Could merge with somewhat more specific check...

// arm64:`STP\t\(R4, R5\), 88\(R6\)`
    MOVD R4, 88(R6)
    MOVD 104(R6), R7
    MOVD R5, 96(R6) // can merge, the load does not alias
    RET

