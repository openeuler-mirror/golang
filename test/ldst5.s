// asmcheck -d=aarch64ldst=all

TEXT ·LdStGlobalGlobal(SB),$16
// arm64:`LDP\t<unlinkable>\.GV1\(SB\), \(R1, R2\)`
  MOVD ·GV1(SB), R1
// arm64:`STP\t\(R1, R2\), <unlinkable>\.GV2\(SB\)`
  MOVD R1, ·GV2(SB)
  MOVD ·GV1+8(SB), R2
  MOVD R2, ·GV2+8(SB)
  RET
