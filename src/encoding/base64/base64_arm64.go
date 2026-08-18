// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//go:build arm64 && !purego

package base64

// decodeStdLut and decodeURLLut are the 128-byte SIMD decode lookup tables:
// byte value b maps to its 6-bit digit, or 255 (illegal) for ASCII characters
// that are not part of the alphabet. The tables cover only 0..127; bytes
// >= 0x80 are rejected by the asm high-bit check before any lookup happens.
var decodeStdLut = [128]byte{
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 62, 255, 255, 255, 63,
	52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 255, 255, 255, 255, 255, 255,
	255, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 255, 255, 255, 255, 255,
	255, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 255, 255, 255, 255, 255,
}

var decodeURLLut = [128]byte{
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 62, 255, 255,
	52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 255, 255, 255, 255, 255, 255,
	255, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 255, 255, 255, 255, 63,
	255, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 255, 255, 255, 255, 255,
}

//go:noescape
func encodeAsm(dst, src []byte, lut *[64]byte) int

//go:noescape
func decodeAsm(dst, src []byte, lut *[128]byte) int

// base64SimdEnabled is replaced by a compiler intrinsic when the package is
// compiled with the -base64simd flag. The intrinsic returns a compile-time
// constant, allowing DCE to eliminate the unused path.
//
// The flag only takes effect on packages actually compiled with it, so the
// feature must be enabled with -gcflags=all=-base64simd or
// -gcflags=encoding/base64=-base64simd. A bare -gcflags=-base64simd applies
// only to the packages named on the command line (typically the main
// package), leaving encoding/base64 unaccelerated without any warning.
func base64SimdEnabled() bool { return false }

// The 16-byte encode and 24-byte decode thresholds pair with the asm loop
// granularities (encode loop12: 12B in / 16B out; decode loop16: 16 chars in /
// 12B out, loop64: 64 chars in / 48B out). For decode in particular, srcLen
// >= 24 guarantees that after one loop16 iteration at least 8 input chars
// (6 output bytes) remain, which keeps the loop16 VST1 4-byte overshoot
// inside the dst capacity allocated by DecodedLen (see the asm comment at the
// loop16 store). Do not change these thresholds independently of the asm.
//
// The decode dispatch additionally requires len(dst) >= DecodedLen(srcLen):
// the overshoot-safety argument above assumes dst was sized for the FULL
// (padding-inclusive) DecodedLen(srcLen), but Decode's contract is to write
// at most len(dst) bytes and callers may pass a shorter dst. AppendDecode is
// exactly such a caller: it strips trailing padding and sizes dst for the
// unpadded decoded length, which is smaller than DecodedLen(srcLen) whenever
// padding is present. The asm path does not know len(dst) and always writes
// to the DecodedLen(srcLen) extent, so dispatching it on an undersized dst
// writes past len(dst) (silently corrupting the caller's buffer, and
// overflowing the heap when the allocation has no capacity slack). Falling
// back to decodeGeneric — which honors len(dst) — is always correct, so gate
// the SIMD path on dst being large enough for its write extent.
//
// The encode dispatch gates on len(dst) >= EncodedLen(len(src)) for the same
// reason: encodeAsm does not know len(dst) and writes exactly
// EncodedLen(consumed) bytes. Encode's contract requires dst to be at least
// EncodedLen(len(src)); a contract-violating caller gets the deterministic
// out-of-range panic of encodeGeneric instead of a silent out-of-bounds
// write, matching the decode side.
func encode(enc *Encoding, dst, src []byte) {
	if base64SimdEnabled() && len(src) >= 16 && len(dst) >= enc.EncodedLen(len(src)) {
		encoded := encodeAsm(dst, src, &enc.encode)
		src = src[(encoded/4)*3:]
		dst = dst[encoded:]
	}
	encodeGeneric(enc, dst, src)
}

func decode(enc *Encoding, dst, src []byte) (int, error) {
	srcLen := len(src)
	if base64SimdEnabled() && srcLen >= 24 && len(dst) >= enc.DecodedLen(srcLen) {
		remain := srcLen
		if enc.lut == &encodeStdLut {
			remain = decodeAsm(dst, src, &decodeStdLut)
		} else if enc.lut == &encodeURLLut {
			remain = decodeAsm(dst, src, &decodeURLLut)
		}

		if remain < srcLen {
			// decoded by SIMD
			remain = srcLen - remain // remain is decoded length now
			src = src[remain:]
			dstStart := (remain / 4) * 3
			dst = dst[dstStart:]
			n, err := decodeGeneric(enc, dst, src)
			if cerr, ok := err.(CorruptInputError); ok {
				return n + dstStart, CorruptInputError(int(cerr) + remain)
			}
			return n + dstStart, err
		}
	}
	return decodeGeneric(enc, dst, src)
}
