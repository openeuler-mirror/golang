// asmcheck -d=aarch64ldst=all

    TEXT	Tstp01(SB), $0-56
// arm64:`MOVD\tR1, \(R0\)`
    MOVD	R1, 0(R0)
// arm64:`STP\t\(R2, R3\), 20\(R0\)`
    STP	    (R2, R3), 20(R0)
// arm64:`STP\t\(R2, R3\), 8\(R0\)`
    MOVD	R2, 8(R0)
    MOVD	R3, 16(R0)
// arm64:`MOVD\tR4, 24\(R0\)`
    MOVD	R4, 24(R0)
    RET

    TEXT	Tstp02(SB), $0-56
// arm64:`STP\t\(R1, R2\), \(R0\)`
    MOVD	R1, 0(R0)
// arm64:`STP\t\(R0, R1\), 16\(R0\)`
    MOVD	R1, 24(R0)
    MOVD	R0, 16(R0)
// arm64:`STP\t\(R0, R2\), 24\(R0\)`
    MOVD	R2, 32(R0)
    MOVD	R2, 8(R0)
    MOVD	R0, 24(R0)
    RET

    TEXT	Tldp01(SB), $0-56
// arm64:`LDP\t32760\(R0\), \(R3, R1\)`
    MOVD	32768(R0), R1
    MOVD	R2, 32752(R0)
    MOVD	32760(R0), R3
    RET

    TEXT	Tldp02(SB), $0-56
// arm64:`MOVD\t32768\(R0\), R1`
    MOVD	32768(R0), R1
    MOVD	R2, 32752(R4)
    MOVD	32760(R0), R3
    RET

    TEXT	Tldp03(SB), $0-56
// arm64:`LDP\t32760\(R0\), \(R2, R1\)`
    MOVD	32768(R0), R1
// arm64:`MOVD\t32768\(R0\), R3`
    MOVD	32768(R0), R3
    MOVD	32760(R0), R2
    RET

    TEXT	Tldp04(SB), $0-56
// arm64:`MOVD\t32768\(R0\), R1`
    MOVD	32768(R0), R1
// arm64:`MOVD\t32768\(R0\), R2`
    MOVD	32768(R0), R2
// arm64:`MOVD\t32760\(R0\), R2`
    MOVD	32760(R0), R2
    RET

    TEXT	Tldp05(SB), $0-56
// arm64:`MOVD\t8\(R0\), R1`
    MOVD	8(R0), R1
// arm64:`MOVD\t\(R0\), R1`
    MOVD	0(R0), R1
    RET

    TEXT	Tldp06(SB), $0-56
    CBZ         R0, 1(PC)
// arm64:`LDP\t\(R0\), \(R2, R1\)`
    MOVD	8(R0), R1
    MOVD	0(R0), R2
    RET

    TEXT	Tldp07(SB), $0-56
    CBZ         R0, 2(PC)
// arm64:`MOVD\t8\(R0\), R1`
    MOVD	8(R0), R1
// arm64:`MOVD\t\(R0\), R2`
    MOVD	0(R0), R2
    RET
