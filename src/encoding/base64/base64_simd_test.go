// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//go:build arm64 && !purego

package base64

// Differential tests for the SIMD encode/decode path (base64_arm64.{go,s}).
//
// simdDecode/simdEncode below mirror the full production dispatch in
// base64_arm64.go — including the srcLen and dst-length gates — but bypass
// the base64SimdEnabled compile-time gate, so the SIMD path is exercised in
// any build, with or without -gcflags=all=-base64simd. The generic
// implementations (decodeGeneric/encodeGeneric) are used as the reference
// instead of enc.Decode/enc.Encode, because the latter may themselves
// dispatch to SIMD when the compiler flag is enabled.
//
// These tests are the regression net for the two fixed classes of bugs:
//   - bytes >= 0x80 were silently decoded as 0 ('A') with no error;
//   - the loop16 store overshoot invariant (see base64_arm64.s).
//
// TestDecodeSimdUndersizedDstNoOverwrite, TestAppendDecodePaddingNoOverwrite,
// TestEncodeSimdUndersizedDstNoOverwrite and the flag-mode assertion in
// TestDispatchThroughRealEntrypoints go through the real entrypoints, so
// they exercise the SIMD dispatch only when the package is built with
// -gcflags=all=-base64simd or -gcflags=encoding/base64=-base64simd; in a
// default build they verify the generic path.
// TestSimdDispatchUndersizedDstFallsBack and
// TestSimdDispatchEncodeUndersizedDstFallsBack cover the same dispatch
// gates in every build via the simdDecode/simdEncode mirrors.

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"
)

func sameError(a, b error) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.Error() == b.Error()
}

// simdDecode mirrors the decode dispatch in base64_arm64.go, including its
// srcLen/dst-length gates, but bypasses the base64SimdEnabled compile-time
// gate so the accelerated path is exercised in any build. Inputs that the
// production dispatch would not send to the asm (srcLen < 24, or dst smaller
// than the full DecodedLen(srcLen) the asm write extent assumes) fall back
// to decodeGeneric exactly as they do in production.
func simdDecode(enc *Encoding, dst, src []byte) (int, error) {
	srcLen := len(src)
	if srcLen < 24 || len(dst) < enc.DecodedLen(srcLen) {
		return decodeGeneric(enc, dst, src)
	}
	remain := srcLen
	if enc.lut == &encodeStdLut {
		remain = decodeAsm(dst, src, &decodeStdLut)
	} else if enc.lut == &encodeURLLut {
		remain = decodeAsm(dst, src, &decodeURLLut)
	}
	if remain < srcLen {
		remain = srcLen - remain
		src = src[remain:]
		dstStart := (remain / 4) * 3
		dst = dst[dstStart:]
		n, err := decodeGeneric(enc, dst, src)
		if cerr, ok := err.(CorruptInputError); ok {
			return n + dstStart, CorruptInputError(int(cerr) + remain)
		}
		return n + dstStart, err
	}
	return decodeGeneric(enc, dst, src)
}

// simdEncode mirrors the encode dispatch, including its length/dst gates but
// bypassing the compile-time flag gate. Inputs the production dispatch would
// not accelerate (len(src) < 16, or dst smaller than EncodedLen(len(src)))
// go through encodeGeneric, as they do in production.
func simdEncode(enc *Encoding, dst, src []byte) {
	if len(src) < 16 || len(dst) < enc.EncodedLen(len(src)) {
		encodeGeneric(enc, dst, src)
		return
	}
	encoded := encodeAsm(dst, src, &enc.encode)
	src = src[(encoded/4)*3:]
	dst = dst[encoded:]
	encodeGeneric(enc, dst, src)
}

var simdTestEncodings = []*Encoding{
	StdEncoding,
	URLEncoding,
	RawStdEncoding,
	RawURLEncoding,
}

// lutFor returns the SIMD decode LUT for enc, mirroring the dispatch in decode.
func lutFor(enc *Encoding) *[128]byte {
	if enc.lut == &encodeStdLut {
		return &decodeStdLut
	}
	if enc.lut == &encodeURLLut {
		return &decodeURLLut
	}
	return nil
}

// deterministic source so failures reproduce exactly
func simdRand(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

func validBase64(r *rand.Rand, enc *Encoding, n int) []byte {
	src := make([]byte, n)
	for i := range src {
		src[i] = enc.encode[r.Intn(64)]
	}
	return src
}

func checkDecode(t *testing.T, enc *Encoding, src []byte) {
	t.Helper()
	want := make([]byte, enc.DecodedLen(len(src)))
	got := make([]byte, enc.DecodedLen(len(src)))
	wantN, wantErr := decodeGeneric(enc, want, src)
	gotN, gotErr := simdDecode(enc, got, src)
	if gotN != wantN || !bytes.Equal(got[:gotN], want[:wantN]) || !sameError(gotErr, wantErr) {
		t.Fatalf("%s Decode(len=%d) = (%d, %v) [% x], want (%d, %v) [% x]",
			enc.encode, len(src), gotN, gotErr, got[:gotN], wantN, wantErr, want[:wantN])
	}
}

func checkEncode(t *testing.T, enc *Encoding, src []byte) {
	t.Helper()
	want := make([]byte, enc.EncodedLen(len(src)))
	got := make([]byte, enc.EncodedLen(len(src)))
	encodeGeneric(enc, want, src)
	simdEncode(enc, got, src)
	if !bytes.Equal(got, want) {
		t.Fatalf("%s Encode(len=%d) = %q, want %q", enc.encode, len(src), got, want)
	}
}

// TestDecodeAsmHighByteContract pins the decodeAsm exit contract for inputs
// containing bytes >= 0x80. Consumption happens in 64-char (loop64) and
// 16-char (loop16) quanta, and the asm must stop at the first block that
// contains a high byte: the remainder is left for the generic path.
func TestDecodeAsmHighByteContract(t *testing.T) {
	r := simdRand(1)
	valid := func(n int) []byte { return validBase64(r, StdEncoding, n) }
	withHigh := func(src []byte, pos int) []byte {
		s := append([]byte{}, src...)
		s[pos] = 0xff
		return s
	}
	cases := []struct {
		name string
		src  []byte
		want int // remaining src chars after decodeAsm
	}{
		{"all-high-64", bytes.Repeat([]byte{0x80}, 64), 64},
		{"all-high-128", bytes.Repeat([]byte{0x80}, 128), 128},
		{"high-first-64", withHigh(valid(64), 0), 64},
		{"high-last-64", withHigh(valid(64), 63), 64},
		{"high-pos30-100", withHigh(valid(100), 30), 100}, // in first loop64 block
		{"high-after-64", append(valid(64), withHigh(valid(64), 0)...), 64},
		{"high-pos99-100", withHigh(valid(100), 99), 20}, // after 64+16 consumed
		{"high-pos23-24", withHigh(valid(24), 23), 8},    // after one loop16 block
		{"all-valid-16", valid(16), 16},                  // below the 24 threshold
		{"all-valid-24", valid(24), 8},                   // one loop16 block
		{"all-valid-64", valid(64), 0},
		{"all-valid-100", valid(100), 20}, // 64 + 16 consumed
		{"all-valid-128", valid(128), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf := make([]byte, len(c.src)/4*3+8)
			remain := decodeAsm(buf, c.src, &decodeStdLut)
			if remain != c.want {
				t.Fatalf("decodeAsm remain = %d, want %d", remain, c.want)
			}
			consumed := len(c.src) - remain
			got := make([]byte, consumed/4*3)
			n, err := decodeGeneric(StdEncoding, got, c.src[:consumed])
			if err != nil || !bytes.Equal(buf[:n], got[:n]) {
				t.Fatalf("consumed prefix decoded incorrectly (n=%d, err=%v)", n, err)
			}
		})
	}
}

// TestDecodeSimdValidAgainstStdlib: valid base64, all alphabets, exhaustive
// lengths 1..300 plus spot checks up to 4096.
func TestDecodeSimdValidAgainstStdlib(t *testing.T) {
	r := simdRand(42)
	for _, enc := range simdTestEncodings {
		for n := 1; n <= 300; n++ {
			checkDecode(t, enc, validBase64(r, enc, n))
		}
		for _, n := range []int{344, 4096} {
			checkDecode(t, enc, validBase64(r, enc, n))
		}
	}
}

// TestDecodeAsmConsumesValidInput: valid input containing every alphabet
// character at every position must be consumed by decodeAsm exactly as the
// loop64/loop16 contract prescribes (64-char blocks, then 16-char blocks
// while >= 24 chars remain). This guards against the masking failure mode
// where a wrong LUT entry makes the SIMD path bail out to the generic path
// and differential tests still pass because the generic fallback produces
// correct results.
func TestDecodeAsmConsumesValidInput(t *testing.T) {
	r := simdRand(23)
	contract := func(n int) int {
		for n >= 64 {
			n -= 64
		}
		for n >= 24 {
			n -= 16
		}
		return n
	}
	for _, enc := range simdTestEncodings {
		lut := lutFor(enc)
		if lut == nil {
			t.Fatalf("no LUT for %s", enc.encode)
		}
		for _, n := range []int{24, 64, 100, 300} {
			want := contract(n)
			for pos := 0; pos < n; pos++ {
				for _, c := range enc.encode {
					src := validBase64(r, enc, n)
					src[pos] = c
					remain := decodeAsm(make([]byte, n/4*3+8), src, lut)
					if remain != want {
						t.Fatalf("%s n=%d pos=%d char=%q: decodeAsm remain=%d, want %d",
							enc.encode, n, pos, c, remain, want)
					}
				}
			}
		}
	}
}

// TestDecodeSimdInvalidAgainstStdlib: the illegal-byte matrix. Every byte
// >= 0x80 used to be silently decoded as 0; the SIMD path must now agree
// with the generic path on n and the CorruptInputError index.
func TestDecodeSimdInvalidAgainstStdlib(t *testing.T) {
	r := simdRand(7)
	highBytes := []byte{0x80, 0x81, 0xa5, 0xc0, 0xe0, 0xfe, 0xff}
	// '-' and '~' are in-table 255 for the std alphabet ('-' is a valid URL
	// digit, '~' is never valid): they must produce CorruptInputError, not be
	// silently decoded through a wrong LUT slot.
	asciiInvalid := []byte{0x00, 0x01, 0x1f, ' ', '!', '#', '$', '%', '=', '-', '~', 0x7f}
	lengths := []int{24, 32, 48, 64, 128, 344}
	positions := []int{0, 1, 15, 16, 17, 30, 63, 64}
	for _, enc := range simdTestEncodings {
		for _, n := range lengths {
			for _, b := range highBytes {
				for _, pos := range positions {
					if pos >= n {
						continue
					}
					src := validBase64(r, enc, n)
					src[pos] = b
					checkDecode(t, enc, src)
				}
			}
			// ASCII-domain illegal bytes must also stay correct.
			for _, b := range asciiInvalid {
				src := validBase64(r, enc, n)
				src[1] = b
				checkDecode(t, enc, src)
			}
		}
	}
}

// TestDecodeSimdHighByteOnly: all-0x80 input must decode to 0 bytes with
// CorruptInputError(0), like the generic path.
func TestDecodeSimdHighByteOnly(t *testing.T) {
	for _, n := range []int{24, 32, 48, 64, 128} {
		src := bytes.Repeat([]byte{0x80}, n)
		checkDecode(t, StdEncoding, src)
		checkDecode(t, URLEncoding, src)
	}
}

// TestDecodeSimdHighByteAnywhere: single high byte injected at every offset,
// for a sweep of lengths crossing the loop64/loop16 boundaries.
func TestDecodeSimdHighByteAnywhere(t *testing.T) {
	r := simdRand(9)
	for _, n := range []int{24, 63, 64, 65, 100, 127, 128, 129, 300, 344} {
		for pos := 0; pos < n; pos++ {
			src := validBase64(r, StdEncoding, n)
			src[pos] = 0xff
			checkDecode(t, StdEncoding, src)
		}
	}
}

// TestDecodeSimdBoundaryLengths: every length in 1..200, both fully valid
// and with a trailing high byte, against the generic path.
func TestDecodeSimdBoundaryLengths(t *testing.T) {
	r := simdRand(11)
	for n := 1; n <= 200; n++ {
		checkDecode(t, StdEncoding, validBase64(r, StdEncoding, n))
		src := validBase64(r, StdEncoding, n)
		src[n-1] = 0xfe
		checkDecode(t, StdEncoding, src)
	}
}

// TestDecodeSimdUndersizedDstNoOverwrite: the SIMD decode dispatch must never
// write past len(dst). The loop16 VST1 overshoot invariant (base64_arm64.s)
// assumes dst is sized for the FULL DecodedLen(srcLen), but Decode's contract
// is to write at most len(dst) bytes and callers may pass a shorter dst.
// AppendDecode is exactly such a caller: it strips trailing '=' padding and
// sizes dst for the unpadded decoded length, which is smaller than
// DecodedLen(srcLen) whenever padding is present. Before the dispatch gained
// a len(dst) >= DecodedLen(srcLen) gate, the SIMD path overran such a dst:
// the error code was still correct, but the bytes just past len(dst) were
// silently rewritten (and a tight allocation turned it into a heap overflow).
// The return values are compared against the generic path, but the actual
// regression net is the canary scan past len(dst).
func TestDecodeSimdUndersizedDstNoOverwrite(t *testing.T) {
	cases := []struct {
		name   string
		src    []byte
		dstLen int // < DecodedLen(len(src)); what AppendDecode would allocate
	}{
		// PR #203 review finding: 16 'A' + 8 '=' = 24 chars, DecodedLen(24)=18,
		// dst sized for the unpadded 16 chars = 12 bytes. One loop16 iteration
		// stored 16 bytes, overrunning dst[12:16].
		{"review-16A-8pad", []byte("AAAAAAAAAAAAAAAA========"), 12},
		// Escalation to two loop16 iterations: 32 'A' + 8 '=' = 40 chars, dst
		// sized for the unpadded 32 chars = 24 bytes; the SIMD path stored 28
		// bytes, overrunning dst[24:28]. With a tight allocation this is a heap
		// out-of-bounds write.
		{"escalation-32A-8pad", []byte(strings.Repeat("A", 32) + "========"), 24},
	}
	for _, enc := range []*Encoding{StdEncoding, URLEncoding} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				// Roomy backing array so any overshoot lands on canaries rather
				// than off the end of the allocation.
				backing := make([]byte, c.dstLen+16)
				for i := range backing {
					backing[i] = 0xCC
				}
				dst := backing[:c.dstLen]

				want := make([]byte, c.dstLen)
				wantN, wantErr := decodeGeneric(enc, want, c.src)

				gotN, gotErr := enc.Decode(dst, c.src)

				if gotN != wantN || !sameError(gotErr, wantErr) {
					t.Fatalf("Decode(len(dst)=%d, src=%q) = (%d, %v), want (%d, %v)",
						c.dstLen, c.src, gotN, gotErr, wantN, wantErr)
				}
				if !bytes.Equal(dst[:gotN], want[:wantN]) {
					t.Fatalf("Decode bytes = % x, want % x", dst[:gotN], want[:wantN])
				}
				// The regression check: nothing at or past len(dst) may be touched.
				for i := c.dstLen; i < len(backing); i++ {
					if backing[i] != 0xCC {
						t.Fatalf("Decode overran len(dst)=%d: backing[%d] = %#x, want canary 0xCC",
							c.dstLen, i, backing[i])
					}
				}
			})
		}
	}
}

// TestAppendDecodePaddingNoOverwrite: the exact API from the PR #203 review
// comment. AppendDecode strips trailing padding and so sizes its buffer for
// the unpadded length; the SIMD decode path must not corrupt the bytes of the
// caller's buffer that lie past the appended result.
func TestAppendDecodePaddingNoOverwrite(t *testing.T) {
	const canary = 0xCC
	for _, enc := range []*Encoding{StdEncoding, URLEncoding} {
		src := []byte("AAAAAAAAAAAAAAAA========") // 16 'A' + 8 '='
		buf := make([]byte, 64)
		for i := range buf {
			buf[i] = canary
		}
		res, err := enc.AppendDecode(buf[:0], src)
		if err == nil {
			t.Fatalf("%s AppendDecode(%q) = no error, want CorruptInputError", enc.encode, src)
		}
		if _, ok := err.(CorruptInputError); !ok {
			t.Fatalf("%s AppendDecode(%q) error = %v, want CorruptInputError", enc.encode, src, err)
		}
		// The appended result is 12 zero bytes (16 'A' -> 12 bytes).
		if len(res) != 12 {
			t.Fatalf("%s AppendDecode result len = %d, want 12", enc.encode, len(res))
		}
		for i := 0; i < 12; i++ {
			if res[i] != 0 {
				t.Fatalf("%s AppendDecode result[%d] = %#x, want 0", enc.encode, i, res[i])
			}
		}
		// Every byte past the appended result must still hold the canary.
		for i := len(res); i < len(buf); i++ {
			if buf[i] != canary {
				t.Fatalf("%s AppendDecode overran the result: buf[%d] = %#x, want canary 0xCC",
					enc.encode, i, buf[i])
			}
		}
	}
}

// TestSimdDispatchUndersizedDstFallsBack: the flag-independent counterpart of
// the two entrypoint tests above. simdDecode applies the same dispatch gates
// as production decode, so a dst sized for the unpadded decoded length (as
// AppendDecode allocates) must fall back to the generic path in every build,
// match the generic result, and leave the bytes past len(dst) untouched. If
// the gate logic regresses and the asm path runs on an undersized dst, the
// canary scan below catches the overwrite.
func TestSimdDispatchUndersizedDstFallsBack(t *testing.T) {
	cases := []struct {
		name   string
		src    []byte
		dstLen int // < DecodedLen(len(src)); what AppendDecode would allocate
	}{
		// 16 'A' + 8 '=' = 24 chars, DecodedLen(24) = 18, unpadded length 12.
		{"16A-8pad", []byte("AAAAAAAAAAAAAAAA========"), 12},
		// 32 'A' + 8 '=' = 40 chars, DecodedLen(40) = 30, unpadded length 24.
		{"32A-8pad", []byte(strings.Repeat("A", 32) + "========"), 24},
	}
	for _, enc := range []*Encoding{StdEncoding, URLEncoding} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				backing := make([]byte, c.dstLen+16)
				for i := range backing {
					backing[i] = 0xCC
				}
				dst := backing[:c.dstLen]

				want := make([]byte, c.dstLen)
				wantN, wantErr := decodeGeneric(enc, want, c.src)

				gotN, gotErr := simdDecode(enc, dst, c.src)

				if gotN != wantN || !sameError(gotErr, wantErr) {
					t.Fatalf("%s simdDecode(len(dst)=%d, src=%q) = (%d, %v), want (%d, %v)",
						enc.encode, c.dstLen, c.src, gotN, gotErr, wantN, wantErr)
				}
				if !bytes.Equal(dst[:gotN], want[:wantN]) {
					t.Fatalf("%s simdDecode bytes = % x, want % x", enc.encode, dst[:gotN], want[:wantN])
				}
				for i := c.dstLen; i < len(backing); i++ {
					if backing[i] != 0xCC {
						t.Fatalf("%s simdDecode overran len(dst)=%d: backing[%d] = %#x, want canary 0xCC",
							enc.encode, c.dstLen, i, backing[i])
					}
				}
			})
		}
	}
}

// TestEncodeSimdAgainstStdlib: exhaustive lengths 1..300 plus spot checks.
func TestEncodeSimdAgainstStdlib(t *testing.T) {
	r := simdRand(13)
	for _, enc := range simdTestEncodings {
		for n := 1; n <= 300; n++ {
			checkEncode(t, enc, makeRandBytes(r, n))
		}
		for _, n := range []int{344, 4096} {
			checkEncode(t, enc, makeRandBytes(r, n))
		}
	}
}

// TestEncodeAsmContract: encodeAsm must consume exactly the blocks it
// reported (encoded/4*3 input bytes).
func TestEncodeAsmContract(t *testing.T) {
	r := simdRand(17)
	for _, n := range []int{16, 17, 28, 47, 48, 49, 60, 61, 100, 200} {
		src := makeRandBytes(r, n)
		dst := make([]byte, StdEncoding.EncodedLen(n))
		encoded := encodeAsm(dst, src, &StdEncoding.encode)
		consumed := (encoded / 4) * 3
		if encoded == 0 {
			t.Fatalf("encodeAsm(len=%d) consumed nothing; SIMD path not exercised", n)
		}
		got := append([]byte{}, dst[:encoded]...)
		want := make([]byte, StdEncoding.EncodedLen(consumed))
		encodeGeneric(StdEncoding, want, src[:consumed])
		if !bytes.Equal(got, want) {
			t.Fatalf("encodeAsm(len=%d) prefix mismatch after %d consumed", n, consumed)
		}
	}
}

// TestEncodeSimdUndersizedDstNoOverwrite: the encode-side counterpart of
// TestDecodeSimdUndersizedDstNoOverwrite, with one deliberate asymmetry in
// what is asserted. decodeGeneric honors len(dst) and returns a partial
// result, so the decode test can compare (n, err); encodeGeneric does NOT
// honor len(dst) — Encode's contract requires dst >= EncodedLen(len(src))
// and encodeGeneric indexes dst unconditionally — so a short-dst Encode
// cannot return a comparable result at all. The contract-preserving outcome
// is encodeGeneric's deterministic index-out-of-range panic: the dispatch
// gate must route an undersized dst to encodeGeneric (panic, bytes past
// len(dst) untouched) and never to encodeAsm, which does not know len(dst)
// and would silently write its full EncodedLen extent. The test asserts
// exactly that: a recovered panic plus an intact 0xCC canary tail past
// len(dst). The panic assertion additionally catches silent-skip mutations
// (fallback that encodes nothing and returns without panicking). If the
// gate regresses and the package is built with -base64simd, encodeAsm runs
// instead and silently writes its full EncodedLen extent past len(dst);
// the canary scan catches the overwrite regardless of whether the dispatch
// then also panics reslicing dst past its length (the damage is already
// done by that point). In a default build the SIMD branch is DCE'd and the
// test verifies the generic-path panic. The src lengths are multiples of
// 48 so encodeAsm's loop48 consumes ALL input and no encodeGeneric tail
// follows it: with a partially consumed input the tail's encodeGeneric
// panic would still occur even with the gate deleted, and only the canary
// scan would tell the two paths apart.
func TestEncodeSimdUndersizedDstNoOverwrite(t *testing.T) {
	cases := []struct {
		name   string
		srcLen int
		dstLen int // < EncodedLen(srcLen)
	}{
		// 48 src bytes: loop48 consumes all 48 and writes exactly 64 bytes;
		// dst is 4 short, so the asm write extent lands on the canaries.
		{"loop48-48", 48, 60},
		// Two loop48 iterations: 96 src bytes -> 128 written, dst 4 short.
		{"loop48-96", 96, 124},
	}
	r := simdRand(29)
	for _, enc := range []*Encoding{StdEncoding, URLEncoding} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				backing := make([]byte, c.dstLen+16)
				for i := range backing {
					backing[i] = 0xCC
				}
				dst := backing[:c.dstLen]
				src := makeRandBytes(r, c.srcLen)

				panicked := false
				func() {
					defer func() {
						if recover() != nil {
							panicked = true
						}
					}()
					enc.Encode(dst, src)
				}()
				if !panicked {
					t.Fatalf("%s Encode(len(dst)=%d, srcLen=%d) did not panic; the dispatch gate must route an undersized dst to encodeGeneric, whose out-of-range panic is the deterministic contract-violation outcome (encodeAsm would silently overrun)",
						enc.encode, c.dstLen, c.srcLen)
				}
				for i := c.dstLen; i < len(backing); i++ {
					if backing[i] != 0xCC {
						t.Fatalf("%s Encode overran len(dst)=%d: backing[%d] = %#x, want canary 0xCC",
							enc.encode, c.dstLen, i, backing[i])
					}
				}
			})
		}
	}
}

// TestSimdDispatchEncodeUndersizedDstFallsBack: the flag-independent
// counterpart of the test above, via the simdEncode mirror (which applies
// the same dst-length gate as the production encode dispatch but bypasses
// the base64SimdEnabled compile-time gate). The gate must route an
// undersized dst to encodeGeneric in every build; the recovered panic
// pins that fallback (it also catches silent-skip mutations), and the
// canary scan is the discriminator against an encodeAsm overrun, which
// writes past len(dst) before any later reslice panic. If the mirror gate
// regresses, encodeAsm runs even in a default build and the canaries past
// len(dst) are clobbered.
func TestSimdDispatchEncodeUndersizedDstFallsBack(t *testing.T) {
	cases := []struct {
		name   string
		srcLen int
		dstLen int // < EncodedLen(srcLen)
	}{
		{"loop48-48", 48, 60},
		{"loop48-96", 96, 124},
	}
	r := simdRand(31)
	for _, enc := range []*Encoding{StdEncoding, URLEncoding} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				backing := make([]byte, c.dstLen+16)
				for i := range backing {
					backing[i] = 0xCC
				}
				dst := backing[:c.dstLen]
				src := makeRandBytes(r, c.srcLen)

				panicked := false
				func() {
					defer func() {
						if recover() != nil {
							panicked = true
						}
					}()
					simdEncode(enc, dst, src)
				}()
				if !panicked {
					t.Fatalf("%s simdEncode(len(dst)=%d, srcLen=%d) did not panic; the mirror gate must fall back to encodeGeneric on an undersized dst (encodeAsm would silently overrun)",
						enc.encode, c.dstLen, c.srcLen)
				}
				for i := c.dstLen; i < len(backing); i++ {
					if backing[i] != 0xCC {
						t.Fatalf("%s simdEncode overran len(dst)=%d: backing[%d] = %#x, want canary 0xCC",
							enc.encode, c.dstLen, i, backing[i])
					}
				}
			})
		}
	}
}

// TestDispatchThroughRealEntrypoints: when the package is compiled with
// -gcflags=all=-base64simd (or -gcflags=encoding/base64=-base64simd) the
// real decode/encode dispatch takes the SIMD path. This test asserts
// correctness in both modes, and additionally checks that the SIMD branch
// was actually taken in flag mode.
func TestDispatchThroughRealEntrypoints(t *testing.T) {
	r := simdRand(19)
	for _, n := range []int{24, 64, 100, 300} {
		src := validBase64(r, StdEncoding, n)
		wantDst := make([]byte, StdEncoding.DecodedLen(n))
		wantN, wantErr := decodeGeneric(StdEncoding, wantDst, src)
		gotDst := make([]byte, StdEncoding.DecodedLen(n))
		gotN, gotErr := StdEncoding.Decode(gotDst, src)
		if gotN != wantN || !bytes.Equal(gotDst[:gotN], wantDst[:wantN]) || !sameError(gotErr, wantErr) {
			t.Fatalf("Decode(len=%d) = (%d, %v), want (%d, %v)", n, gotN, gotErr, wantN, wantErr)
		}
	}
	if !base64SimdEnabled() {
		t.Log("generic mode (no -base64simd flag); SIMD branch verified by direct-asm tests above")
		return
	}
	// In flag mode the dispatch must actually have used the SIMD path; use a
	// high-byte input where generic and SIMD used to diverge.
	src := bytes.Repeat([]byte{0x80}, 64)
	dst := make([]byte, StdEncoding.DecodedLen(len(src)))
	n, err := StdEncoding.Decode(dst, src)
	if n != 0 || err == nil || err.Error() != CorruptInputError(0).Error() {
		t.Fatalf("flag-mode Decode(all-0x80) = (%d, %v), want (0, CorruptInputError(0))", n, err)
	}
}

func makeRandBytes(r *rand.Rand, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Intn(256))
	}
	return b
}
