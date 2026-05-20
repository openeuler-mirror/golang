// asmcheck -d=aarch64ldst=all

TEXT ·ldstStackGlobal(SB),$16
// arm64:`LDP\t<unlinkable>.ldstSlice\(SB\), \(R0, R1\)`
  MOVD    ·ldstSlice(SB), R0
  MOVD    R0, -8(RSP)
  MOVD    ·ldstSlice+8(SB), R1
  MOVD    R1, -24(RSP)
  MOVD    ·ldstSlice+16(SB), R2
  MOVD    R2, -16(RSP)
  RET
