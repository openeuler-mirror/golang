// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parsing of Moduledata executables (Linux, FreeBSD, and so on).

// Package objfile provides moduledata parsing and output functionality for objdump.
// It enables reading Go binary metadata including types, interface vtables (itabs),
// function tables, and runtime structures. The moduledata is read using safe
// pointer translation to handle addresses that may not be directly accessible.

package objfile

import (
	"bufio"
	"cmd/internal/sys"
	"fmt"
	"internal/abi"
	"internal/bytealg"
	"internal/goarch"
	"internal/goos"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"unsafe"
)

// Source: src/runtime/malloc.go
const (
	_64bit       = 1 << (^uintptr(0) >> 63) / 2
	heapAddrBits = (_64bit*(1-goarch.IsWasm)*(1-goos.IsIos*goarch.IsArm64))*48 + (1-_64bit+goarch.IsWasm)*(32-(goarch.IsMips+goarch.IsMipsle)) + 40*goos.IsIos*goarch.IsArm64
	maxAlloc     = (1 << heapAddrBits) - (1-_64bit)*1
)

// Source: src/runtime/symtab.go
var firstmoduledata moduledata

// Copy of structure from src/runtime/symtab.go
type FuncInfo struct {
	*_func
	datap *moduledata
}

// Local variables
var arch *sys.Arch
var MD *Moduledata
var gofuncBuf []byte
var findfunctabBuf []byte
var typesBuf []byte
var typesBase uintptr
var GOOS = runtime.GOOS
var GOARCH = runtime.GOARCH

var printFuncData = false

func SetPrintFuncData(val bool) {
	printFuncData = val
}

var PCDATA_IND = []uint32{abi.PCDATA_UnsafePoint, abi.PCDATA_StackMapIndex, abi.PCDATA_InlTreeIndex, abi.PCDATA_ArgLiveIndex}
var PCDATA_TO_STR = []string{"PCDATA_UnsafePoint", "PCDATA_StackMapIndex", "PCDATA_InlTreeIndex", "PCDATA_ArgLiveIndex"}
var FUNCDATA_IND = []uint8{abi.FUNCDATA_ArgsPointerMaps, abi.FUNCDATA_LocalsPointerMaps, abi.FUNCDATA_StackObjects, abi.FUNCDATA_InlTree, abi.FUNCDATA_OpenCodedDeferInfo, abi.FUNCDATA_ArgInfo, abi.FUNCDATA_ArgLiveInfo, abi.FUNCDATA_WrapInfo}
var FUNCDATA_TO_STR = []string{"FUNCDATA_ArgsPointerMaps", "FUNCDATA_LocalsPointerMaps", "FUNCDATA_StackObjects", "FUNCDATA_InlTree", "FUNCDATA_OpenCodedDeferInfo", "FUNCDATA_ArgInfo", "FUNCDATA_ArgLiveInfo", "FUNCDATA_WrapInfo"}

const minfunc = 16
const FuncTabBucketSize = 256 * minfunc // 4096 bytes
const SUBBUCKETS = 16
const SUBBUCKETSIZE = FuncTabBucketSize / SUBBUCKETS // 256 bytes

// Copy from src/runtime/type.go
type interfacetype = abi.InterfaceType

// type itab is defined in moduledata_types.go (sourced from runtime/runtime2.go)
// This alias is for compatibility - in Go 1.22 itab is defined in moduledata_types.go

// Type is an alias for abi.Type used in objfile for parsing moduledata
// Source: src/internal/abi/type.go
// rtype is a wrapper that allows us to define additional methods.
type rtype struct {
	*abi.Type
}

// From src/runtime/type.go
func toRType(t *_type) *rtype {
	return &rtype{(*abi.Type)(t)}
}

func (t rtype) string() string {
	n := t.nameOff(t.Str)
	if n.Bytes == nil {
		return ""
	}
	s := n.Name()
	if t.TFlag&abi.TFlagExtraStar != 0 {
		return s[1:]
	}
	return s
}

func (t rtype) name() string {
	if t.TFlag&abi.TFlagNamed == 0 {
		return ""
	}
	s := t.string()
	i := len(s) - 1
	sqBrackets := 0
	for i >= 0 && (s[i] != '.' || sqBrackets != 0) {
		switch s[i] {
		case ']':
			sqBrackets++
		case '[':
			sqBrackets--
		}
		i--
	}
	return s[i+1:]
}

// From src/runtime/type.go
func (t rtype) nameOff(off nameOff) name {
	// based on resolveNameOff(unsafe.Pointer(t.Type), off)
	return name{Bytes: (*byte)(unsafe.Pointer(firstmoduledata.types + uintptr(off)))}
}

// translateToBuffer translates a pointer from the types section to a buffer-accessible address.
// It validates that the pointer offset is within bounds of the types buffer.
func translateToBuffer(originalPtr unsafe.Pointer) (unsafe.Pointer, error) {
	if originalPtr == nil {
		return nil, nil
	}
	offset := uintptr(originalPtr) - typesBase
	if int(offset) < 0 || int(offset) > len(typesBuf) {
		return nil, fmt.Errorf("pointer offset 0x%x out of typesBuf bounds [0, 0x%x)", offset, len(typesBuf))
	}
	return add(unsafe.Pointer(firstmoduledata.types), offset), nil
}

// translateSliceData translates a slice pointer to buffer-accessible memory.
// Returns the data pointer, length, and any error.
func translateSliceData(slicePtr unsafe.Pointer, elemSize uintptr) (uintptr, int, error) {
	sh := (*reflect.SliceHeader)(slicePtr)
	if sh.Len == 0 {
		return 0, 0, nil
	}
	offset := sh.Data - typesBase
	totalSize := uintptr(sh.Len) * elemSize
	if int(offset) < 0 || int(offset)+int(totalSize) > len(typesBuf) {
		return 0, sh.Len, fmt.Errorf("slice data out of bounds")
	}
	return uintptr(add(unsafe.Pointer(firstmoduledata.types), offset)), sh.Len, nil
}

func readVarint(data uintptr) int {
	v := 0
	for i := 0; i < 10; i++ {
		b := *(*byte)(unsafe.Pointer(data + uintptr(i)))
		v += int(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return v
		}
	}
	return 0
}

func translateFieldName(name abi.Name) string {
	if name.Bytes == nil {
		return ""
	}
	namePtr, err := translateToBuffer(unsafe.Pointer(name.Bytes))
	if err != nil {
		return "<error>"
	}
	ptr := uintptr(namePtr)
	offset := 1
	_, nameLen := readVarintWithOffset(ptr, 1)
	offset = 1 + varintSize(ptr, 1) + nameLen
	if nameLen > 0 && nameLen < 2048 {
		return readString(ptr+uintptr(varintSize(ptr, 1)+1), nameLen)
	}
	_, tagLen := readVarintWithOffset(ptr, offset)
	offset += varintSize(ptr, offset) + tagLen
	if tagLen > 0 && tagLen < 2048 {
		return readString(ptr+uintptr(offset), tagLen)
	}
	return ""
}

func varintSize(data uintptr, start int) int {
	for i := start; i < start+10; i++ {
		if *(*byte)(unsafe.Pointer(data + uintptr(i)))&0x80 == 0 {
			return i - start + 1
		}
	}
	return 10
}

func readVarintWithOffset(data uintptr, offset int) (int, int) {
	v := 0
	for i := 0; ; i++ {
		b := *(*byte)(unsafe.Pointer(data + uintptr(offset+i)))
		v += int(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return i + 1, v
		}
	}
}

func readString(data uintptr, n int) string {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = *(*byte)(unsafe.Pointer(data + uintptr(i)))
	}
	return string(b)
}

func translateName(n abi.Name) (abi.Name, error) {
	if n.Bytes == nil {
		return n, nil
	}
	ptr, err := translateToBuffer(unsafe.Pointer(n.Bytes))
	if err != nil {
		return abi.Name{}, err
	}
	return abi.Name{Bytes: (*byte)(ptr)}, nil
}

// Local module structures
type Moduledata struct {
	md *moduledata
	ph *pcHeader
}

type FuncSymbol struct {
	Name string
	Addr uint64
	Size uint64
}

func (e *Entry) Moduledata(checkVersion bool) (*Moduledata, error) {
	goarch := e.raw.goarch()
	var targetArch *sys.Arch
	for _, a := range sys.Archs {
		if a.Name == goarch {
			targetArch = a
			break
		}
	}
	// Cross-architecture parsing is intentionally unsupported for now:
	// moduledata and pcHeader are reinterpreted using the host layout, so
	// e.g. a 64-bit tool would read shifted fields from a 32-bit target.
	if targetArch == nil || targetArch.Name != runtime.GOARCH {
		return nil, fmt.Errorf("cross-architecture moduledata parsing is not supported: binary is %s, tool is %s", goarch, runtime.GOARCH)
	}
	arch = targetArch

	if checkVersion == true {
		binVer, err := e.raw.buildVersion()
		if err != nil {
			return nil, err
		}
		// Check that tool & input binary uses same compiler verison
		// only goX.YY pattern should be matched, so we check only
		// first 6 bytes
		if binVer != runtime.Version()[:6] {
			return nil, fmt.Errorf("Binary/Tool compiler versions mismatch: %s vs %s", binVer, runtime.Version()[:6])
		}
	}

	mdraw, err := e.raw.firstmoduledata()
	if err != nil {
		return nil, err
	}
	md := (*moduledata)(unsafe.Pointer(&mdraw[0]))
	phraw, err := e.raw.pcHeader(uint64(uintptr(unsafe.Pointer((*md).pcHeader))))
	if err != nil {
		return nil, err
	}
	ph := (*pcHeader)(unsafe.Pointer(&phraw[0]))

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&(*md).pclntable))
	pclntableBuf, err := e.raw.tableBufAt(uint64(sh.Data), uint64(sh.Len))
	if err != nil {
		return nil, err
	}
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.pclntable))
	sh.Data = (uintptr)(unsafe.Pointer(&pclntableBuf[0]))

	sh = (*reflect.SliceHeader)(unsafe.Pointer(&(*md).pctab))
	pctabBuf, err := e.raw.tableBufAt(uint64(sh.Data), uint64(sh.Len))
	if err != nil {
		return nil, err
	}
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.pctab))
	sh.Data = (uintptr)(unsafe.Pointer(&pctabBuf[0]))

	sh = (*reflect.SliceHeader)(unsafe.Pointer(&(*md).funcnametab))
	funcnametabBuf, err := e.raw.tableBufAt(uint64(sh.Data), uint64(sh.Len))
	if err != nil {
		return nil, err
	}
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.funcnametab))
	sh.Data = (uintptr)(unsafe.Pointer(&funcnametabBuf[0]))

	textsectSize := uint64(unsafe.Sizeof(*(*textsect)(nil)))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&(*md).textsectmap))
	textsectmapBuf, err := e.raw.tableBufAt(uint64(sh.Data), uint64(sh.Len)*textsectSize)
	if err != nil {
		return nil, err
	}
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.textsectmap))
	sh.Data = (uintptr)(unsafe.Pointer(&textsectmapBuf[0]))

	functabSize := uint64(unsafe.Sizeof(*(*functab)(nil)))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&(*md).ftab))
	ftabBuf, err := e.raw.tableBufAt(uint64(sh.Data), uint64(sh.Len)*functabSize)
	if err != nil {
		return nil, err
	}
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.ftab))
	sh.Data = (uintptr)(unsafe.Pointer(&ftabBuf[0]))

	// TODO: Currently moduledata structure contain only start of gofunc table, but it
	// doesn't contain information about it's size. go:func.* symbol doesn't have size
	// in ELF eigher, so we count on fact that next table defined in moduledata
	// will be gcdata, so use it for size calculation.
	gofuncBuf, err = e.raw.tableBufAt(uint64((*md).gofunc), uint64((*md).gcdata-(*md).gofunc))
	if err != nil {
		return nil, err
	}

	calculatedSize := uint64(((*md).maxpc - (*md).minpc + FuncTabBucketSize - 1) / FuncTabBucketSize * unsafe.Sizeof(findfuncbucket{}))
	calculatedBuckets := int(calculatedSize / uint64(unsafe.Sizeof(findfuncbucket{})))
	bucketSize := uint64(unsafe.Sizeof(findfuncbucket{}))

	symbolSize, symbolErr := e.raw.symbolSize("runtime.findfunctab")
	if symbolErr == nil && symbolSize > 0 {
		symbolBuckets := int((symbolSize + bucketSize - 1) / bucketSize)
		if symbolBuckets != calculatedBuckets {
			fmt.Fprintf(os.Stderr, "objdump: WARNING: runtime.findfunctab symbol size (%d bytes, %d buckets) != calculated (%d bytes, %d buckets)\n",
				symbolSize, symbolBuckets, calculatedSize, calculatedBuckets)
		}
	}

	findfunctabSize := calculatedSize
	findfunctabBuf, err = e.raw.tableBufAt(uint64((*md).findfunctab), findfunctabSize)
	if err != nil {
		findfunctabBuf = nil
	}

	typesBuf, err = e.raw.tableBufAt(uint64((*md).types), uint64((*md).etypes-(*md).types))
	if err != nil {
		return nil, err
	}

	typelinkSize := uint64(unsafe.Sizeof(*(*int32)(nil)))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&(*md).typelinks))
	typelinksBuf, err := e.raw.tableBufAt(uint64(sh.Data), uint64(sh.Len)*typelinkSize)
	if err != nil {
		return nil, err
	}

	itablinkSize := uint64(unsafe.Sizeof(*(**itab)(nil)))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&(*md).itablinks))
	itablinksBuf, err := e.raw.tableBufAt(uint64(sh.Data), uint64(sh.Len)*itablinkSize)
	if err != nil {
		return nil, err
	}

	MD = &Moduledata{
		md: md,
		ph: ph,
	}

	firstmoduledata = *md

	// Fix slices pointers to Data after original moduledata was saved to firstmoduledata & Moduledata.md
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.pclntable))
	sh.Data = (uintptr)(unsafe.Pointer(&pclntableBuf[0]))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.funcnametab))
	sh.Data = (uintptr)(unsafe.Pointer(&funcnametabBuf[0]))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.pctab))
	sh.Data = (uintptr)(unsafe.Pointer(&pctabBuf[0]))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.ftab))
	sh.Data = (uintptr)(unsafe.Pointer(&ftabBuf[0]))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.textsectmap))
	sh.Data = (uintptr)(unsafe.Pointer(&textsectmapBuf[0]))
	firstmoduledata.gofunc = (uintptr)(unsafe.Pointer(&gofuncBuf[0]))
	firstmoduledata.findfunctab = (uintptr)(unsafe.Pointer(&findfunctabBuf[0]))

	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.typelinks))
	sh.Data = (uintptr)(unsafe.Pointer(&typelinksBuf[0]))
	sh = (*reflect.SliceHeader)(unsafe.Pointer(&firstmoduledata.itablinks))
	sh.Data = (uintptr)(unsafe.Pointer(&itablinksBuf[0]))
	typesBase = firstmoduledata.types
	firstmoduledata.types = (uintptr)(unsafe.Pointer(&typesBuf[0]))

	// Override next moduledata (support only first so far)
	// TODO: Support multiple moduledata
	firstmoduledata.next = nil

	return MD, nil
}

// Verification issue types and collection
type VerificationIssue struct {
	Level  string
	Func   string
	Table  string
	PC     uint64
	Offset uint64
	Msg    string
}

func (v VerificationIssue) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s]", v.Level))
	if v.Func != "" {
		sb.WriteString(fmt.Sprintf(" %s", v.Func))
	}
	if v.Table != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", v.Table))
	}
	if v.PC != 0 {
		sb.WriteString(fmt.Sprintf(" @0x%x", v.PC))
	}
	sb.WriteString(fmt.Sprintf(": %s", v.Msg))
	return sb.String()
}

type VerificationResult struct {
	Issues   []VerificationIssue
	HasError bool
}

func newVerificationResult() *VerificationResult {
	return &VerificationResult{
		Issues: make([]VerificationIssue, 0),
	}
}

func (r *VerificationResult) add(level string, funcName, table string, pc uint64, msg string) {
	issue := VerificationIssue{
		Level: level,
		Func:  funcName,
		Table: table,
		PC:    pc,
		Msg:   msg,
	}
	r.Issues = append(r.Issues, issue)
	if level == "ERROR" {
		r.HasError = true
	}
}

func (r *VerificationResult) addIssue(issue VerificationIssue) {
	r.Issues = append(r.Issues, issue)
	if issue.Level == "ERROR" {
		r.HasError = true
	}
}

func (r *VerificationResult) String() string {
	if len(r.Issues) == 0 {
		return "moduledata verification: no issues found"
	}
	var sb strings.Builder
	// Count non-DEBUG issues (errors/warnings)
	errorOrWarningCount := 0
	for _, issue := range r.Issues {
		if issue.Level != "DEBUG" {
			errorOrWarningCount++
		}
	}
	// Print all issues if there are errors/warnings, otherwise show "no issues found"
	// DEBUG-only output (like "skipped") is informational but not a problem
	if errorOrWarningCount > 0 {
		sb.WriteString("moduledata verification issues:\n")
		for _, issue := range r.Issues {
			sb.WriteString("  ")
			sb.WriteString(issue.String())
			sb.WriteString("\n")
		}
		return sb.String()
	}
	// Only DEBUG messages (informational) - treat as "no issues"
	return "moduledata verification: no issues found"
}

func (r *VerificationResult) ExitCode() int {
	if r.HasError {
		return 1
	}
	return 0
}

// verifyModuledata performs core verification of moduledata structure
// Mirrors runtime.moduledataverify1() from src/runtime/symtab.go
func verifyModuledata(md *moduledata, ph *pcHeader, result *VerificationResult) {
	// Check for nil critical structures (would crash objdump)
	if md == nil {
		result.add("ERROR", "", "", 0, "moduledata is nil")
		return
	}
	if ph == nil {
		result.add("ERROR", "", "", 0, "pcHeader is nil")
		return
	}

	// Check ftab is not empty
	if len(md.ftab) == 0 {
		result.add("ERROR", "", "", 0, "ftab is empty")
		return
	}

	// Check minpc/maxpc validity
	if md.minpc >= md.maxpc {
		result.add("ERROR", "", "", 0, fmt.Sprintf("invalid minpc/maxpc: minpc=0x%x, maxpc=0x%x", md.minpc, md.maxpc))
	}

	// pcHeader magic and format validation
	// Must match 0xFFFFFFF1 for Go 1.12+
	if ph.magic != 0xFFFFFFF1 {
		result.add("ERROR", "", "", 0, fmt.Sprintf("pcHeader: invalid magic=0x%x (expected 0xFFFFFFF1)", ph.magic))
	}
	if ph.pad1 != 0 {
		result.add("DEBUG", "", "", 0, fmt.Sprintf("pcHeader: pad1=%d (expected 0)", ph.pad1))
	}
	if ph.pad2 != 0 {
		result.add("DEBUG", "", "", 0, fmt.Sprintf("pcHeader: pad2=%d (expected 0)", ph.pad2))
	}

	// minLC must match PCQuantum for the architecture
	expectedMinLC := uint8(arch.MinLC)
	if ph.minLC != expectedMinLC {
		result.add("ERROR", "", "", 0, fmt.Sprintf("pcHeader: minLC=%d (expected %d for %s)", ph.minLC, expectedMinLC, GOARCH))
	}

	// ptrSize must match goarch.PtrSize
	expectedPtrSize := uint8(goarch.PtrSize)
	if ph.ptrSize != expectedPtrSize {
		result.add("ERROR", "", "", 0, fmt.Sprintf("pcHeader: ptrSize=%d (expected %d)", ph.ptrSize, expectedPtrSize))
	}

	// textStart must match moduledata.text
	if ph.textStart != md.text {
		result.add("ERROR", "", "", 0, fmt.Sprintf("pcHeader: textStart=0x%x != moduledata.text=0x%x", ph.textStart, md.text))
	}

	// ftab sorting validation
	// Each function's entryoff must be less than or equal to the next function's entryoff
	nftab := len(md.ftab)
	if nftab < 2 {
		result.add("DEBUG", "", "", 0, "ftab has fewer than 2 entries, skipping sort check")
	} else {
		for i := 0; i < nftab-1; i++ {
			if md.ftab[i].entryoff > md.ftab[i+1].entryoff {
				result.add("ERROR", "", "", 0,
					fmt.Sprintf("ftab not sorted at index %d: entryoff=0x%x > 0x%x",
						i, md.ftab[i].entryoff, md.ftab[i+1].entryoff))
			}
		}
	}

	// minpc/maxpc computed from ftab
	// ftab[0].entryoff should correspond to minpc
	// ftab[nftab-1].entryoff should correspond to maxpc
	if nftab >= 1 {
		minComputed := md.textAddr(md.ftab[0].entryoff)
		maxComputed := md.textAddr(md.ftab[nftab-1].entryoff)
		if md.minpc != minComputed {
			result.add("DEBUG", "", "", 0,
				fmt.Sprintf("minpc=0x%x != computed 0x%x from ftab[0]", md.minpc, minComputed))
		}
		if md.maxpc != maxComputed {
			result.add("DEBUG", "", "", 0,
				fmt.Sprintf("maxpc=0x%x != computed 0x%x from ftab[last]", md.maxpc, maxComputed))
		}
	}
}

// verifyPCDataCoverage checks that pcsp and PCDATA tables cover the entire function.
// Uses ELF symbol boundaries (Addr, Size) as the expected end address.
// Varint-based range calculation (same as display) via step() function.
func verifyPCDataCoverage(md *moduledata, symbols []FuncSymbol, result *VerificationResult, filter *regexp.Regexp) {
	nftab := len(md.ftab)
	if nftab < 2 {
		return
	}
	nfuncs := nftab - 1

	// Build symbol map for fast lookup by address
	// Some functions may have multiple ftab entries with the same name at different addresses,
	// so address-based matching is more reliable than name-based matching.
	symbolMapByAddr := make(map[uint64]FuncSymbol)
	for _, sym := range symbols {
		symbolMapByAddr[sym.Addr] = sym
	}

	// Iterate through all functions
	for i := 0; i < nfuncs; i++ {
		// Get function bounds from ftab
		entryOff := md.ftab[i].entryoff
		entryPC := md.textAddr(entryOff)

		// Create FuncInfo for this function
		if md.ftab[i].funcoff >= uint32(len(md.pclntable)) {
			continue
		}
		f := FuncInfo{(*_func)(unsafe.Pointer(&md.pclntable[md.ftab[i].funcoff])), md}
		funcName := funcNameFromOffset(f._func.nameOff, md)

		// Check if function name matches filter (if filter provided)
		if filter != nil && !filter.MatchString(funcName) {
			continue
		}

		// Look up ELF symbol by entry address
		sym, hasSymbol := symbolMapByAddr[uint64(entryPC)]
		if !hasSymbol {
			// No matching ELF symbol - skip verification for this function
			result.add("DEBUG", funcName, "symbol", uint64(entryPC),
				"no matching ELF symbol, skipped")
			continue
		}

		// Expected end address from ELF symbol
		expectedEnd := uintptr(sym.Addr + sym.Size)

		// Verify pcsp coverage (if present)
		if f._func.pcsp != 0 {
			verifySinglePCDataCoverage(f, funcName, "pcsp", entryPC, expectedEnd, result)
		}

		// Verify each PCDATA table
		for tableIdx := 0; tableIdx < int(f._func.npcdata); tableIdx++ {
			// Map tableIdx to PCDATA index name
			if tableIdx >= len(PCDATA_TO_STR) {
				break
			}
			tableName := PCDATA_TO_STR[tableIdx]
			off := pcdatastart(f, uint32(tableIdx))
			if off != 0 {
				verifySinglePCDataTableCoverage(f, funcName, tableName, off, entryPC, expectedEnd, result)
			}
		}
	}
}

// pctabAt returns datap.pctab[off:], or an error if off points outside the
// pctab. The offset comes from the binary being inspected and must not be
// used to index the buffer without validation.
func pctabAt(datap *moduledata, off uint32) ([]byte, error) {
	if uint64(off) >= uint64(len(datap.pctab)) {
		return nil, fmt.Errorf("pctab offset %d out of range [0, %d)", off, len(datap.pctab))
	}
	return datap.pctab[off:], nil
}

// readvarintBounded reads, removes, and returns a varint from *p. Unlike
// readvarint it returns an error instead of panicking when the table ends
// in the middle of a varint or the varint is too long.
func readvarintBounded(p *[]byte) (uint32, error) {
	var v, shift uint32
	s := *p
	for shift = 0; ; shift += 7 {
		if len(s) == 0 {
			return 0, fmt.Errorf("truncated varint")
		}
		if shift > 28 {
			return 0, fmt.Errorf("varint too long")
		}
		b := s[0]
		s = s[1:]
		v |= (uint32(b) & 0x7F) << shift
		if b&0x80 == 0 {
			break
		}
	}
	*p = s
	return v, nil
}

// stepBounded advances to the next pc, value pair in the encoded table,
// like step, but returns an error instead of panicking when the table ends
// in the middle of a varint. pc and val are only updated when the whole
// entry was decoded successfully.
func stepBounded(p *[]byte, pc *uint64, val *int32, first bool, arch *sys.Arch) (bool, error) {
	uvdelta, err := readvarintBounded(p)
	if err != nil {
		return false, err
	}
	if uvdelta == 0 && !first {
		return false, nil
	}
	if uvdelta&1 != 0 {
		uvdelta = ^(uvdelta >> 1)
	} else {
		uvdelta >>= 1
	}
	vdelta := int32(uvdelta)
	pcdelta, err := readvarintBounded(p)
	if err != nil {
		return false, err
	}
	*pc += uint64(pcdelta * uint32(arch.MinLC))
	*val += vdelta
	return true, nil
}

// verifySinglePCDataCoverage verifies a single pcdata table covers [entryPC, expectedEnd)
// Uses varint-based range calculation (same as display) via step() function.
func verifySinglePCDataCoverage(f FuncInfo, funcName, tableName string, entryPC, expectedEnd uintptr, result *VerificationResult) {
	// pcsp offset is directly in _func.pcsp
	off := f._func.pcsp
	if off == 0 {
		return
	}

	p, err := pctabAt(f.datap, off)
	if err != nil {
		result.add("ERROR", funcName, tableName, uint64(entryPC), err.Error())
		return
	}
	pc := uint64(entryPC)
	maxPC := uint64(entryPC)
	val := int32(-1)

	for {
		prevPC := pc
		ok, err := stepBounded(&p, &pc, &val, prevPC == uint64(entryPC), arch)
		if err != nil {
			result.add("ERROR", funcName, tableName, uint64(entryPC),
				fmt.Sprintf("invalid pctab entry at offset %d: %v", off, err))
			return
		}
		if !ok {
			break
		}
		maxPC = pc
	}

	if maxPC < uint64(expectedEnd) {
		result.add("DEBUG", funcName, tableName, uint64(entryPC),
			fmt.Sprintf("coverage gap: covered [0x%x, 0x%x), expected [0x%x, 0x%x)",
				entryPC, maxPC, entryPC, expectedEnd))
	}
}

// verifySinglePCDataTableCoverage verifies a PCDATA table covers [entryPC, expectedEnd)
// Uses varint-based range calculation (same as display) via step() function.
func verifySinglePCDataTableCoverage(f FuncInfo, funcName, tableName string, tableOff uint32, entryPC, expectedEnd uintptr, result *VerificationResult) {
	p, err := pctabAt(f.datap, tableOff)
	if err != nil {
		result.add("ERROR", funcName, tableName, uint64(entryPC), err.Error())
		return
	}
	pc := uint64(entryPC)
	maxPC := uint64(entryPC)
	val := int32(-1)

	for {
		prevPC := pc
		ok, err := stepBounded(&p, &pc, &val, prevPC == uint64(entryPC), arch)
		if err != nil {
			result.add("ERROR", funcName, tableName, uint64(entryPC),
				fmt.Sprintf("invalid pctab entry at offset %d: %v", tableOff, err))
			return
		}
		if !ok {
			break
		}
		maxPC = pc
	}

	if maxPC < uint64(expectedEnd) {
		result.add("DEBUG", funcName, tableName, uint64(entryPC),
			fmt.Sprintf("coverage gap: covered [0x%x, 0x%x), expected [0x%x, 0x%x)",
				entryPC, maxPC, entryPC, expectedEnd))
	}
}

// verifyWrapperIntegrity verifies wrapper function integrity
func verifyWrapperIntegrity(md *moduledata, result *VerificationResult) {
	nftab := len(md.ftab)
	if nftab < 2 {
		return
	}
	nfuncs := nftab - 1

	for i := 0; i < nfuncs; i++ {
		if md.ftab[i].funcoff >= uint32(len(md.pclntable)) {
			continue
		}
		f := FuncInfo{(*_func)(unsafe.Pointer(&md.pclntable[md.ftab[i].funcoff])), md}
		funcName := funcNameFromOffset(f._func.nameOff, md)

		// Check if this is a wrapper function
		if f._func.funcID != abi.FuncIDWrapper {
			continue
		}

		// Check FUNCDATA_WrapInfo presence
		// Only wrappers with captured variables (go/defer) have FUNCDATA_WrapInfo
		// ABI wrappers, interface methods, runtime helpers don't have it - these are all valid
		if uintptr(f._func.nfuncdata) <= uintptr(abi.FUNCDATA_WrapInfo) {
			continue
		}

		// Read wrapped function PC from FUNCDATA_WrapInfo
		funcdataBase := uintptr(unsafe.Pointer(&f.nfuncdata)) + unsafe.Sizeof(f.nfuncdata) + uintptr(f._func.npcdata)*4
		wrapInfoPtr := funcdataBase + uintptr(abi.FUNCDATA_WrapInfo)*4
		wrappedOff := *(*uint32)(unsafe.Pointer(wrapInfoPtr))
		if wrappedOff == ^uint32(0) {
			result.add("DEBUG", funcName, "wrapper", 0, "FUNCDATA_WrapInfo offset is nil")
			continue
		}

		if uintptr(wrappedOff)+4 > uintptr(len(gofuncBuf)) {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedOff),
				fmt.Sprintf("FUNCDATA_WrapInfo offset 0x%x out of go:func range [0, 0x%x)", wrappedOff, len(gofuncBuf)))
			continue
		}
		wrappedTextOff := *(*uint32)(unsafe.Pointer(md.gofunc + uintptr(wrappedOff)))
		if uintptr(wrappedTextOff) >= uintptr(md.etext-md.text) {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedTextOff),
				fmt.Sprintf("wrapped function text offset 0x%x out of range [0, 0x%x)", wrappedTextOff, md.etext-md.text))
			continue
		}
		wrappedPC := md.textAddr(wrappedTextOff)

		// Verify wrapped PC is within bounds (ERROR - would crash)
		if wrappedPC < md.minpc || wrappedPC >= md.maxpc {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedPC),
				fmt.Sprintf("wrapped PC 0x%x out of bounds [0x%x, 0x%x)",
					wrappedPC, md.minpc, md.maxpc))
		}

		// Verify wrapped function exists in ftab (findfunc)
		// First check findfunctab bounds to avoid crash on truncated buffer
		if err := checkFindfunctabBounds(wrappedPC); err != nil {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedPC),
				fmt.Sprintf("findfunc bounds check failed: %s", err.Error()))
		} else {
			wf := findfunc(wrappedPC)
			if !wf.Valid() {
				result.add("ERROR", funcName, "wrapper", uint64(wrappedPC),
					"findfunc returned invalid FuncInfo (wrapped function not found in ftab)")
			}
		}

		// Note: ELF symbol check would require access to Entry's symbol table
		// This is done separately in objdump main if needed
	}
}

// verifyFindfunctab validates the findfunctab bucket table structure and consistency
// This table provides fast PC-to-function lookup with ~0.5% overhead
func verifyFindfunctab(result *VerificationResult) {
	md := &firstmoduledata

	nftab := len(md.ftab) - 1
	if nftab <= 0 {
		return // Already checked in verifyModuledata
	}

	// Use actual buffer size for bucket count
	// The buffer size is determined by the linker formula, but actual symbol size may differ
	// for externally linked binaries. Using actual buffer size avoids reading garbage.
	bucketSize := int(unsafe.Sizeof(findfuncbucket{}))
	nbuckets := (len(findfunctabBuf) + bucketSize - 1) / bucketSize

	// Calculate total subbuckets from text range and subbuckets in last (partial) bucket
	// These match the linker's formulas in pcln.go:848-850
	n := int((md.maxpc - md.minpc + SUBBUCKETSIZE - 1) / SUBBUCKETSIZE)
	lastBucketSubbuckets := n - (nbuckets-1)*SUBBUCKETS
	if lastBucketSubbuckets < 0 {
		lastBucketSubbuckets = 0
	}
	if lastBucketSubbuckets > SUBBUCKETS {
		lastBucketSubbuckets = SUBBUCKETS
	}

	if nbuckets <= 0 {
		result.add("DEBUG", "", "findfunctab", 0,
			"findfunctab buffer too small or empty")
		return
	}

	// Validate buffer size against expected size from text segment (ceiling)
	expectedBuckets := int((md.maxpc - md.minpc + FuncTabBucketSize - 1) / FuncTabBucketSize)
	if nbuckets < expectedBuckets {
		result.add("ERROR", "", "findfunctab", uint64(md.minpc+uintptr(nbuckets)*FuncTabBucketSize),
			fmt.Sprintf("findfunctab buffer truncated: expected %d buckets for pc range [0x%x, 0x%x), got %d",
				expectedBuckets, md.minpc, md.maxpc, nbuckets))
	}

	prevIdx := uint32(0)
	for b := 0; b < nbuckets; b++ {
		ffb := (*findfuncbucket)(add(unsafe.Pointer(md.findfunctab), uintptr(b)*unsafe.Sizeof(findfuncbucket{})))

		// Check idx is valid
		if ffb.idx >= uint32(nftab) {
			result.add("ERROR", "", "findfunctab", uint64(md.minpc+uintptr(b)*FuncTabBucketSize),
				fmt.Sprintf("bucket[%d] idx=%d exceeds ftab bounds (len=%d)",
					b, ffb.idx, nftab))
		}

		// Check per-bucket function limit (linker constraint: idx + sub < 256)
		subbucketsToCheck := SUBBUCKETS
		if b == nbuckets-1 {
			subbucketsToCheck = lastBucketSubbuckets
		}
		for i := 0; i < subbucketsToCheck; i++ {
			totalIdx := uint32(ffb.idx) + uint32(ffb.subbuckets[i])
			if totalIdx >= uint32(nftab) {
				result.add("ERROR", "", "findfunctab", uint64(md.minpc+uintptr(b)*FuncTabBucketSize),
					fmt.Sprintf("bucket[%d] subbucket[%d]: idx=%d + sub=%d = %d exceeds ftab bounds (len=%d)",
						b, i, ffb.idx, ffb.subbuckets[i], totalIdx, nftab))
			}
			// Note: subbuckets[i] stores (idx - base) as uint8, always < 256
			// This is enforced by the linker and the uint8 storage type
		}

		// Check ftab[idx].funcoff is valid
		if ffb.idx < uint32(nftab) {
			funcoff := md.ftab[ffb.idx].funcoff
			maxFuncoff := uint32(len(md.pclntable))
			if funcoff >= maxFuncoff {
				result.add("ERROR", "", "findfunctab", uint64(md.minpc+uintptr(b)*FuncTabBucketSize),
					fmt.Sprintf("bucket[%d] ftab[%d].funcoff=%d exceeds pclntable size (%d)",
						b, ffb.idx, funcoff, maxFuncoff))
			}
		}

		// Check ftab[idx].entryoff <= ftab[idx+1].entryoff (monotonicity)
		if ffb.idx > 0 && ffb.idx < uint32(nftab) {
			if md.ftab[ffb.idx].entryoff > md.ftab[ffb.idx+1].entryoff {
				result.add("ERROR", "", "findfunctab", uint64(md.minpc+uintptr(b)*FuncTabBucketSize),
					fmt.Sprintf("bucket[%d] ftab[%d].entryoff=0x%x > ftab[%d].entryoff=0x%x (not monotonic)",
						b, ffb.idx, md.ftab[ffb.idx].entryoff, ffb.idx+1, md.ftab[ffb.idx+1].entryoff))
			}
		}

		// Check idx monotonicity (must be non-decreasing for correct findfunc operation)
		if b > 0 && ffb.idx < prevIdx {
			result.add("ERROR", "", "findfunctab", uint64(md.minpc+uintptr(b)*FuncTabBucketSize),
				fmt.Sprintf("bucket[%d] idx=%d decreased from bucket[%d] idx=%d (not monotonically non-decreasing)",
					b, ffb.idx, b-1, prevIdx))
		}
		// Detect idx reset to 0 after reaching non-zero (sign of incomplete/truncated data)
		if prevIdx > 0 && ffb.idx == 0 {
			result.add("ERROR", "", "findfunctab", uint64(md.minpc+uintptr(b)*FuncTabBucketSize),
				fmt.Sprintf("bucket[%d] idx reset to 0 after reaching %d (incomplete findfunctab data)",
					b, prevIdx))
		}
		prevIdx = ffb.idx
	}

	// Verify scan loop termination: max possible idx should be < len(ftab)-1
	// This prevents potential infinite loops in findfunc
	maxPossibleIdx := uint32(0)
	for b := 0; b < nbuckets; b++ {
		ffb := (*findfuncbucket)(add(unsafe.Pointer(md.findfunctab), uintptr(b)*unsafe.Sizeof(findfuncbucket{})))
		subbucketsToCheck := SUBBUCKETS
		if b == nbuckets-1 {
			subbucketsToCheck = lastBucketSubbuckets
		}
		for i := 0; i < subbucketsToCheck; i++ {
			idx := uint32(ffb.idx) + uint32(ffb.subbuckets[i])
			if idx > maxPossibleIdx {
				maxPossibleIdx = idx
			}
		}
	}
	// The scan loop accesses ftab[idx+1], so idx must be < len(ftab)-1
	if maxPossibleIdx >= uint32(nftab) {
		result.add("ERROR", "", "findfunctab", 0,
			fmt.Sprintf("max possible idx=%d >= ftab length-1=%d (scan loop may access out of bounds)",
				maxPossibleIdx, nftab))
	}

	// Sample PC lookup verification
	samplePCs := []uintptr{
		md.minpc,
		md.minpc + FuncTabBucketSize/2,
		md.minpc + FuncTabBucketSize - 1,
		md.maxpc - 1,
	}

	for _, pc := range samplePCs {
		if pc < md.minpc || pc >= md.maxpc {
			continue
		}

		// Check findfunctab bounds before calling findfunc
		if err := checkFindfunctabBounds(pc); err != nil {
			result.add("ERROR", "", "findfunctab", uint64(pc),
				fmt.Sprintf("findfunc bounds check failed: %s", err.Error()))
			continue
		}

		fi := findfunc(pc)
		if !fi.Valid() {
			result.add("ERROR", "", "findfunctab", uint64(pc),
				fmt.Sprintf("findfunc(0x%x) returned invalid FuncInfo", pc))
			continue
		}

		// Verify returned function contains this PC
		pcOff, ok := fi.datap.textOff(pc)
		if !ok {
			result.add("ERROR", "", "findfunctab", uint64(pc),
				fmt.Sprintf("findfunc(0x%x): cannot compute textOff", pc))
			continue
		}

		// Find the ftab index for this function
		var fiIdx uint32
		for i := uint32(0); i < uint32(nftab); i++ {
			if fi._func == (*_func)(unsafe.Pointer(&fi.datap.pclntable[fi.datap.ftab[i].funcoff])) {
				fiIdx = i
				break
			}
		}

		// Check: ftab[fiIdx].entryoff <= pcOff < ftab[fiIdx+1].entryoff
		if fiIdx < uint32(nftab) {
			entryoff := fi.datap.ftab[fiIdx].entryoff
			var endOff uint32
			if fiIdx+1 < uint32(nftab) {
				endOff = fi.datap.ftab[fiIdx+1].entryoff
			} else {
				// Last function - use maxpc as end
				endOff = uint32(md.maxpc - md.text)
			}

			if pcOff < entryoff || pcOff >= endOff {
				result.add("ERROR", "", "findfunctab", uint64(pc),
					fmt.Sprintf("findfunc(0x%x) returned function but PC not in [0x%x, 0x%x)",
						pc, fi.datap.textAddr(entryoff), fi.datap.textAddr(endOff)))
			}
		}
	}
}

// verifyTypeBounds checks typelinks and itablinks are within types section
func verifyTypeBounds(result *VerificationResult) {
	md := &firstmoduledata
	if md.types >= md.etypes {
		return // Already checked in verifyModuledata
	}

	typesLen := int(md.etypes - md.types)

	// Check typelinks
	for i := 0; i < len(md.typelinks); i++ {
		offset := int(md.typelinks[i])
		if offset < 0 || offset >= typesLen {
			result.add("DEBUG", "", "typelinks", 0,
				fmt.Sprintf("TYPE[%d]: offset %d out of bounds [0, 0x%x)",
					i, offset, typesLen))
		}
	}

	// Check itablinks
	for i := 0; i < len(md.itablinks); i++ {
		if md.itablinks[i] == nil {
			continue
		}
		itPtr := uintptr(unsafe.Pointer(md.itablinks[i]))
		offset := int(itPtr - typesBase)
		if offset < 0 || offset >= typesLen {
			result.add("DEBUG", "", "itablinks", 0,
				fmt.Sprintf("ITAB[%d]: offset 0x%x out of bounds [0, 0x%x)",
					i, offset, typesLen))
		}
	}
}

// funcNameFromOffset retrieves function name from funcnametab
func funcNameFromOffset(nameOff int32, md *moduledata) string {
	if nameOff < 0 || int(nameOff) >= len(md.funcnametab) {
		return "<invalid name offset>"
	}
	// Find null terminator
	start := int(nameOff)
	end := start
	for end < len(md.funcnametab) && md.funcnametab[end] != 0 {
		end++
	}
	return string(md.funcnametab[start:end])
}

// VerifyModuledata runs all verification phases and returns collected issues.
// symbols provides ELF symbol information (Addr, Size) for accurate function boundaries.
func (p *Moduledata) VerifyModuledata(symbols []FuncSymbol) *VerificationResult {
	result := newVerificationResult()

	// Use firstmoduledata global which has been properly initialized with buffer pointers
	md := &firstmoduledata

	// Core verification (pcHeader, ftab, minpc/maxpc)
	verifyModuledata(md, p.ph, result)
	if result.HasError {
		return result
	}

	// PCDATA/PCSP coverage verification using symbol boundaries
	verifyPCDataCoverage(md, symbols, result, nil)
	if result.HasError {
		return result
	}

	// Wrapper function verification
	verifyWrapperIntegrity(md, result)
	if result.HasError {
		return result
	}

	// typelinks/itablinks bounds
	verifyTypeBounds(result)

	// findfunctab validation
	verifyFindfunctab(result)

	return result
}

// VerifyAllFindfunctabBuckets performs comprehensive verification of every
// bucket and subbucket in the findfunctab table. For each (bucket, subbucket)
// pair, it:
// 1. Calculates the expected PC address (center of subbucket range)
// 2. Calls findfunc() to resolve the PC to a function
// 3. Verifies findfunc returns a function that contains the PC within its bounds
func (p *Moduledata) VerifyAllFindfunctabBuckets() *VerificationResult {
	result := newVerificationResult()
	md := &firstmoduledata

	nftab := len(md.ftab) - 1
	if nftab <= 0 {
		return result
	}

	bucketSize := int(unsafe.Sizeof(findfuncbucket{}))
	nbuckets := (len(findfunctabBuf) + bucketSize - 1) / bucketSize

	// Calculate total subbuckets from text range and subbuckets in last (partial) bucket
	n := int((md.maxpc - md.minpc + SUBBUCKETSIZE - 1) / SUBBUCKETSIZE)
	lastBucketSubbuckets := n - (nbuckets-1)*SUBBUCKETS
	if lastBucketSubbuckets < 0 {
		lastBucketSubbuckets = 0
	}
	if lastBucketSubbuckets > SUBBUCKETS {
		lastBucketSubbuckets = SUBBUCKETS
	}

	if nbuckets <= 0 {
		result.add("DEBUG", "", "findfunctab", 0, "empty findfunctab")
		return result
	}

	// Check buffer size matches expected for PC range (ceiling)
	expectedBuckets := int((md.maxpc - md.minpc + FuncTabBucketSize - 1) / FuncTabBucketSize)
	if nbuckets < expectedBuckets {
		result.add("ERROR", "", "findfunctab", uint64(md.findfunctab),
			fmt.Sprintf("findfunctab buffer truncated: expected %d buckets for pc range [0x%x, 0x%x), got %d",
				expectedBuckets, md.minpc, md.maxpc, nbuckets))
		result.HasError = true
		return result
	}

	// Verify every bucket and subbucket
	checked := 0
	errCount := 0
	for b := 0; b < nbuckets; b++ {
		ffb := (*findfuncbucket)(add(unsafe.Pointer(md.findfunctab), uintptr(b)*unsafe.Sizeof(findfuncbucket{})))
		bucketBasePC := md.minpc + uintptr(b)*FuncTabBucketSize

		subbucketsToCheck := SUBBUCKETS
		if b == nbuckets-1 {
			subbucketsToCheck = lastBucketSubbuckets
		}
		for i := 0; i < subbucketsToCheck; i++ {
			// Calculate PC this subbucket represents (center of subbucket range)
			subbucketPC := bucketBasePC + uintptr(i)*(FuncTabBucketSize/16) + (FuncTabBucketSize/16)/2

			// Skip PCs outside text range
			if subbucketPC < md.text || subbucketPC >= md.maxpc {
				continue
			}
			checked++

			// Get pcOff for this PC
			pcOff, ok := md.textOff(subbucketPC)
			if !ok {
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d]: textOff(0x%x) failed", b, i, subbucketPC))
				errCount++
				continue
			}

			// Expected idx from bucket table
			expectedIdx := uint32(ffb.idx) + uint32(ffb.subbuckets[i])
			if expectedIdx >= uint32(nftab) {
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d]: idx=%d >= nftab=%d", b, i, expectedIdx, nftab))
				errCount++
				continue
			}

			// Call findfunc() to get the function
			// Check findfunctab bounds first to avoid crash on truncated buffer
			if err := checkFindfunctabBounds(subbucketPC); err != nil {
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d]: findfunc bounds check failed: %s", b, i, err.Error()))
				errCount++
				continue
			}

			fi := findfunc(subbucketPC)
			if !fi.Valid() {
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d]: findfunc(0x%x) returned invalid FuncInfo", b, i, subbucketPC))
				errCount++
				continue
			}

			// Find the ftab index for this function
			fiIdx := uint32(0)
			found := false
			for j := uint32(0); j < uint32(nftab); j++ {
				if fi._func == (*_func)(unsafe.Pointer(&fi.datap.pclntable[fi.datap.ftab[j].funcoff])) {
					fiIdx = j
					found = true
					break
				}
			}
			if !found {
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d]: findfunc returned unknown function", b, i))
				errCount++
				continue
			}

			// Check ftab entryoff bounds: entryoff <= pcOff < next entryoff
			entryoff := fi.datap.ftab[fiIdx].entryoff
			var endOff uint32
			if fiIdx+1 < uint32(nftab) {
				endOff = fi.datap.ftab[fiIdx+1].entryoff
			} else {
				endOff = uint32(md.maxpc - md.text)
			}

			if pcOff < entryoff {
				funcName := fi.datap.funcName(fi._func.nameOff)
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d] (idx=%d): pc 0x%x below function %s entry 0x%x",
						b, i, expectedIdx, pcOff, funcName, entryoff))
				errCount++
			}
			if pcOff >= endOff {
				funcName := fi.datap.funcName(fi._func.nameOff)
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d] (idx=%d): pc 0x%x above function %s end 0x%x",
						b, i, expectedIdx, pcOff, funcName, endOff))
				errCount++
			}

			// Check idx consistency: findfunc may advance idx forward via scan loop
			// so fiIdx >= expectedIdx is valid, but fiIdx < expectedIdx indicates inconsistency
			if fiIdx < expectedIdx {
				funcName := fi.datap.funcName(fi._func.nameOff)
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d]: idx=%d but findfunc returned ftab[%d] (%s) (idx regression)",
						b, i, expectedIdx, fiIdx, funcName))
				errCount++
			}

			// Check that ftab[expectedIdx] points to the same function as findfunc
			// This catches inconsistencies where bucket table and ftab disagree
			expectedFunc := (*_func)(unsafe.Pointer(&fi.datap.pclntable[fi.datap.ftab[expectedIdx].funcoff]))
			if fi._func != expectedFunc {
				funcName := fi.datap.funcName(fi._func.nameOff)
				expectedName := fi.datap.funcName(expectedFunc.nameOff)
				result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
					fmt.Sprintf("bucket[%d].subbucket[%d]: ftab[%d] points to %s, but findfunc(0x%x) returned %s",
						b, i, expectedIdx, expectedName, subbucketPC, funcName))
				errCount++
			}

			// Check entryoff is in expected subbucket range (only when function starts here)
			// Function starts here if: i==0 OR subbuckets[i] != subbuckets[i-1]
			// This handles cross-bucket functions automatically (they have equal subbuckets)
			funcStartsHere := (i == 0 || ffb.subbuckets[i] != ffb.subbuckets[i-1])
			if funcStartsHere {
				funcEntryoff := fi.datap.ftab[expectedIdx].entryoff
				expectedMin := uint32(md.text - md.minpc + bucketBasePC - md.minpc + uintptr(i)*(FuncTabBucketSize/16))
				expectedMax := expectedMin + uint32(FuncTabBucketSize/16)
				if funcEntryoff < expectedMin || funcEntryoff >= expectedMax {
					funcName := fi.datap.funcName(fi._func.nameOff)
					result.add("ERROR", "", "findfunctab", uint64(subbucketPC),
						fmt.Sprintf("bucket[%d].subbucket[%d] (%s): ftab[%d].entryoff=0x%x not in expected range [0x%x, 0x%x)",
							b, i, funcName, expectedIdx, funcEntryoff, expectedMin, expectedMax))
					errCount++
				}
			}
		}
	}

	return result
}

// VerifyModuledataForFuncs verifies moduledata for functions matching filter
// and continues to disassembly after verification.
// symbols provides ELF symbol information (Addr, Size) for accurate function boundaries.
func (p *Moduledata) VerifyModuledataForFuncs(symbols []FuncSymbol, filter *regexp.Regexp) *VerificationResult {
	result := newVerificationResult()

	// Use firstmoduledata global which has been properly initialized with buffer pointers
	md := &firstmoduledata

	// Core verification (pcHeader, ftab, minpc/maxpc)
	// Always verify core (Option A: stop early on errors)
	verifyModuledata(md, p.ph, result)
	if result.HasError {
		return result
	}

	// PCDATA/PCSP coverage for matching functions only (using symbol boundaries)
	verifyPCDataCoverage(md, symbols, result, filter)

	// Wrapper integrity for matching functions only
	verifyWrapperIntegrityForFuncs(md, result, filter)

	// NOTE: typelinks/itablinks are SKIPPED with -s
	// They are not function-specific and verification is the same as -d alone

	return result
}

// verifyWrapperIntegrityForFuncs verifies wrapper integrity for functions matching filter only
func verifyWrapperIntegrityForFuncs(md *moduledata, result *VerificationResult, filter *regexp.Regexp) {
	nftab := len(md.ftab)
	if nftab < 2 {
		return
	}

	// Iterate through all functions looking for wrappers
	for i := 0; i < nftab-1; i++ {
		if md.ftab[i].funcoff >= uint32(len(md.pclntable)) {
			continue
		}

		f := FuncInfo{(*_func)(unsafe.Pointer(&md.pclntable[md.ftab[i].funcoff])), md}
		funcName := funcNameFromOffset(f._func.nameOff, md)

		// Check if function name matches filter
		if !filter.MatchString(funcName) {
			continue
		}

		// Check if this is a wrapper function
		if f._func.funcID != abi.FuncIDWrapper {
			continue
		}

		// Only wrappers with captured variables (go/defer) have FUNCDATA_WrapInfo
		// ABI wrappers, interface methods, runtime helpers don't have it - these are all valid
		if uintptr(f._func.nfuncdata) <= uintptr(abi.FUNCDATA_WrapInfo) {
			continue
		}

		// Read wrapped function PC from FUNCDATA_WrapInfo
		funcdataBase := uintptr(unsafe.Pointer(&f.nfuncdata)) + unsafe.Sizeof(f.nfuncdata) + uintptr(f._func.npcdata)*4
		wrapInfoPtr := funcdataBase + uintptr(abi.FUNCDATA_WrapInfo)*4
		wrappedOff := *(*uint32)(unsafe.Pointer(wrapInfoPtr))
		if wrappedOff == ^uint32(0) {
			result.add("DEBUG", funcName, "wrapper", 0, "FUNCDATA_WrapInfo offset is nil")
			continue
		}

		if uintptr(wrappedOff)+4 > uintptr(len(gofuncBuf)) {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedOff),
				fmt.Sprintf("FUNCDATA_WrapInfo offset 0x%x out of go:func range [0, 0x%x)", wrappedOff, len(gofuncBuf)))
			continue
		}
		wrappedTextOff := *(*uint32)(unsafe.Pointer(md.gofunc + uintptr(wrappedOff)))
		if uintptr(wrappedTextOff) >= uintptr(md.etext-md.text) {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedTextOff),
				fmt.Sprintf("wrapped function text offset 0x%x out of range [0, 0x%x)", wrappedTextOff, md.etext-md.text))
			continue
		}
		wrappedPC := md.textAddr(wrappedTextOff)

		// Verify wrapped PC is within bounds (ERROR - would crash)
		if wrappedPC < md.minpc || wrappedPC >= md.maxpc {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedPC),
				fmt.Sprintf("wrapped PC 0x%x out of bounds [0x%x, 0x%x)",
					wrappedPC, md.minpc, md.maxpc))
		}

		// Verify wrapped function exists in ftab (findfunc)
		// First check findfunctab bounds to avoid crash on truncated buffer
		if err := checkFindfunctabBounds(wrappedPC); err != nil {
			result.add("ERROR", funcName, "wrapper", uint64(wrappedPC),
				fmt.Sprintf("findfunc bounds check failed: %s", err.Error()))
		} else {
			wf := findfunc(wrappedPC)
			if !wf.Valid() {
				result.add("ERROR", funcName, "wrapper", uint64(wrappedPC),
					"findfunc returned invalid FuncInfo (wrapped function not found in ftab)")
			}
		}
	}
}

// Output helper functions
func (p pcHeader) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

// Copy from src/runtime/string.go
func findnull(s *byte) int {
	if s == nil {
		return 0
	}

	// Avoid IndexByteString on Plan 9 because it uses SSE instructions
	// on x86 machines, and those are classified as floating point instructions,
	// which are illegal in a note handler.
	if GOOS == "plan9" {
		p := (*[maxAlloc/2 - 1]byte)(unsafe.Pointer(s))
		l := 0
		for p[l] != 0 {
			l++
		}
		return l
	}

	// pageSize is the unit we scan at a time looking for NULL.
	// It must be the minimum page size for any architecture Go
	// runs on. It's okay (just a minor performance loss) if the
	// actual system page size is larger than this value.
	const pageSize = 4096

	offset := 0
	ptr := unsafe.Pointer(s)
	// IndexByteString uses wide reads, so we need to be careful
	// with page boundaries. Call IndexByteString on
	// [ptr, endOfPage) interval.
	safeLen := int(pageSize - uintptr(ptr)%pageSize)

	for {
		t := *(*string)(unsafe.Pointer(&stringStruct{ptr, safeLen}))
		// Check one page at a time.
		if i := bytealg.IndexByteString(t, 0); i != -1 {
			return offset + i
		}
		// Move to next page
		ptr = unsafe.Pointer(uintptr(ptr) + uintptr(safeLen))
		offset += safeLen
		safeLen = pageSize
	}
}

// Copy from src/runtime/string.go
func gostringnocopy(str *byte) string {
	ss := stringStruct{str: unsafe.Pointer(str), len: findnull(str)}
	s := *(*string)(unsafe.Pointer(&ss))
	return s
}

// Copy from src/runtime/symtab.go
func (md *moduledata) funcName(nameOff int32) string {
	if nameOff == 0 {
		return ""
	}
	return gostringnocopy(&md.funcnametab[nameOff])
}

func (p moduledata) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		switch value.Kind() {
		case reflect.Ptr:
			sb.WriteString(fmt.Sprintf("%s:0x%x", field.Name, uint64(value.Pointer())))
		case reflect.Slice:
			sb.WriteString(fmt.Sprintf("%s:{Data:0x%x Len:%d Cap:%d}", field.Name, uintptr(value.UnsafePointer()), value.Len(), value.Cap()))
		case reflect.Struct:
			method := value.MethodByName("String")
			if method.IsValid() {
				sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
			}
		case reflect.Map:
			sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value)) //TODO: find better output
		default:
			sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		}
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (p bitvector) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	// We do not use reflect for printing structure fields here, because
	// we need print byteslice with variable size (p.n)
	sb.WriteString(fmt.Sprintf("n:%d,", p.n))
	bytedata := unsafe.Slice((*byte)(unsafe.Pointer(p.bytedata)), p.n)
	sb.WriteString(fmt.Sprintf("bytedata:%#v", bytedata))
	sb.WriteString("}")
	return sb.String()
}

func (p functab) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{\n", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		sb.WriteString(fmt.Sprintf("\t%s:%#v\n", field.Name, value))
	}
	sb.WriteString("}")
	return sb.String()
}

func (p _func) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if field.Name == "_" {
			sb.WriteString(fmt.Sprintf("%s", field.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		}
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (p inlinedCall) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if field.Name == "_" {
			sb.WriteString(fmt.Sprintf("%s", field.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		}
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (p stackmap) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	// We do not use reflect for printing structure fields here, because
	// we need print byteslice with variable size (p.n)
	sb.WriteString(fmt.Sprintf("n:%d,nbit:%d,", p.n, p.nbit))
	bytedata := unsafe.Slice((*byte)(unsafe.Pointer(&p.bytedata[0])), p.n*((p.nbit+7)>>3))
	sb.WriteString(fmt.Sprintf("bytedata:%#v", bytedata))
	sb.WriteString("}")
	return sb.String()
}

func (p stackObjectRecord) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

// Original function src/runtime/mbitmap.go:addb
func addb(p *byte, n uintptr) *byte {
	return (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + n))
}

// Original function src/runtime/stubs.go:noescape
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

// Original function src/runtime/stubs.go:add
func add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

// Original function src/runtime/symtab.go:stackmapdata
func stackmapdata(stkmap *stackmap, n int32) bitvector {
	return bitvector{stkmap.nbit, addb(&stkmap.bytedata[0], uintptr(n*((stkmap.nbit+7)>>3)))}
}

// Original function src/runtime/symtab.go
func findmoduledatap(pc uintptr) *moduledata {
	for datap := &firstmoduledata; datap != nil; datap = datap.next {
		if datap.minpc <= pc && pc < datap.maxpc {
			return datap
		}
	}
	return nil
}

func throw(s string) {
	panic(s)
}

// Original function src/runtime/symtab.go
func (md *moduledata) textAddr(off32 uint32) uintptr {
	off := uintptr(off32)
	res := md.text + off
	if len(md.textsectmap) > 1 {
		for i, sect := range md.textsectmap {
			// For the last section, include the end address (etext), as it is included in the functab.
			if off >= sect.vaddr && off < sect.end || (i == len(md.textsectmap)-1 && off == sect.end) {
				res = sect.baseaddr + off - sect.vaddr
				break
			}
		}
		if res > md.etext && GOARCH != "wasm" {
			throw("runtime: text offset out of range")
		}
	}
	return res
}

// Original function src/runtime/symtab.go
func (md *moduledata) textOff(pc uintptr) (uint32, bool) {
	res := uint32(pc - md.text)
	if len(md.textsectmap) > 1 {
		for i, sect := range md.textsectmap {
			if sect.baseaddr > pc {
				// pc is not in any section.
				return 0, false
			}
			end := sect.baseaddr + (sect.end - sect.vaddr)
			// For the last section, include the end address (etext), as it is included in the functab.
			if i == len(md.textsectmap)-1 {
				end++
			}
			if pc < end {
				res = uint32(pc - sect.baseaddr + sect.vaddr)
				break
			}
		}
	}
	return res, true
}

// Original function src/runtime/symtab.go
// entry returns the entry PC for f.
func (f FuncInfo) Valid() bool {
	return f._func != nil
}

func (f FuncInfo) entry() uintptr {
	return f.datap.textAddr(f.entryOff)
}

// Original function src/runtime/symtab.go
func findfunc(pc uintptr) FuncInfo {
	datap := findmoduledatap(pc)
	if datap == nil {
		return FuncInfo{}
	}
	const nsub = uintptr(len(findfuncbucket{}.subbuckets))

	pcOff, ok := datap.textOff(pc)
	if !ok {
		return FuncInfo{}
	}

	x := uintptr(pcOff) + datap.text - datap.minpc // TODO: are datap.text and datap.minpc always equal?
	b := x / FuncTabBucketSize
	i := x % FuncTabBucketSize / (FuncTabBucketSize / nsub)

	// Bounds check: ensure bucket index is within findfunctab buffer
	bucketSize := int(unsafe.Sizeof(findfuncbucket{}))
	maxBuckets := len(findfunctabBuf) / bucketSize
	if int(b) >= maxBuckets {
		expectedBuckets := int((datap.maxpc - datap.minpc + FuncTabBucketSize - 1) / FuncTabBucketSize)
		fmt.Fprintf(os.Stderr, "objdump: ERROR: findfunc(0x%x): bucket index %d exceeds findfunctab bounds "+
			"(have %d buckets for range [0x%x, 0x%x), need %d buckets)\n",
			pc, b, maxBuckets, datap.minpc, datap.maxpc, expectedBuckets)
		return FuncInfo{}
	}

	ffb := (*findfuncbucket)(add(unsafe.Pointer(datap.findfunctab), b*unsafe.Sizeof(findfuncbucket{})))
	idx := ffb.idx + uint32(ffb.subbuckets[i])

	// Bounds check: ensure idx is within ftab bounds (catches garbage from truncated buffer)
	if idx >= uint32(len(datap.ftab)) {
		fmt.Fprintf(os.Stderr, "objdump: ERROR: findfunc(0x%x): idx %d exceeds ftab bounds (len=%d), "+
			"bucket index %d may have garbage data from truncated findfunctab buffer\n",
			pc, idx, len(datap.ftab), b)
		return FuncInfo{}
	}

	// Find the ftab entry.
	for idx < uint32(len(datap.ftab))-1 && datap.ftab[idx+1].entryoff <= pcOff {
		idx++
	}

	funcoff := datap.ftab[idx].funcoff
	return FuncInfo{(*_func)(unsafe.Pointer(&datap.pclntable[funcoff])), datap}
}

// checkFindfunctabBounds validates if a PC is safe to use with findfunc().
// Returns nil if safe, or an error with debugging info if findfunctab is truncated.
func checkFindfunctabBounds(pc uintptr) error {
	md := &firstmoduledata
	if pc < md.minpc || pc >= md.maxpc {
		return nil
	}
	pcOff, ok := md.textOff(pc)
	if !ok {
		return nil
	}
	x := uintptr(pcOff) + md.text - md.minpc
	bucketIdx := x / FuncTabBucketSize
	bucketSize := int(unsafe.Sizeof(findfuncbucket{}))
	maxBuckets := len(findfunctabBuf) / bucketSize
	expectedBuckets := int((md.maxpc - md.minpc + FuncTabBucketSize - 1) / FuncTabBucketSize)
	if bucketIdx >= uintptr(maxBuckets) {
		return fmt.Errorf("PC 0x%x maps to bucket index %d but findfunctab has only %d buckets "+
			"(expected %d buckets for range [0x%x, 0x%x))",
			pc, bucketIdx, maxBuckets, expectedBuckets, md.minpc, md.maxpc)
	}
	return nil
}

// Original function src/runtime/panic.go:readvarintUnsafe
func readvarintUnsafe(fd unsafe.Pointer) (uint32, unsafe.Pointer) {
	var r uint32
	var shift int
	for {
		b := *(*uint8)(fd)
		fd = add(fd, unsafe.Sizeof(b))
		if b < 128 {
			return r + uint32(b)<<shift, fd
		}
		r += uint32(b&0x7F) << (shift & 31)
		shift += 7
		if shift > 28 {
			panic("Bad varint")
		}
	}
}

func (f FuncInfo) String() string {
	return f.stringInternal(true)
}

func (f FuncInfo) StringBrief() string {
	return f.stringInternal(false)
}

func (f FuncInfo) stringInternal(printDetails bool) string {
	var sb strings.Builder
	datap := f.datap
	sb.WriteString(fmt.Sprintf("%s\n", f._func.String()))
	sb.WriteString(fmt.Sprintf("Name:%s\n", datap.funcName(f._func.nameOff)))

	if !printDetails || !printFuncData {
		return sb.String()
	}

	maxInlTreeIndex := int32(0)
	maxArgLiveIndex := int32(0)

	if f._func.pcsp > 0 {
		sb.WriteString("PCSP:\n")
		pp, err := pctabAt(datap, f._func.pcsp)
		if err != nil {
			sb.WriteString(fmt.Sprintf("\tERROR: %v\n", err))
		} else {
			pc := uint64(f.entry())
			val := int32(-1)
			for {
				var ok bool
				prev_pc := pc
				ok, err = stepBounded(&pp, &pc, &val, pc == 0, arch)
				if err != nil {
					sb.WriteString(fmt.Sprintf("\tERROR: invalid pctab entry at offset %d: %v\n", f._func.pcsp, err))
					break
				}
				if !ok {
					break
				}
				sb.WriteString(fmt.Sprintf("\t0x%x-0x%x:%d\n", prev_pc, pc-1, val))
				if val&int32(arch.PtrSize-1) != 0 {
					sb.WriteString(fmt.Sprintf("\tERROR: invalid spdelta %d at 0x%x\n", val, prev_pc))
					break
				}
			}
		}
	}

	for _, ind := range PCDATA_IND {
		if ind >= f._func.npcdata {
			break
		}
		off := pcdatastart(f, ind)
		// Based on src/runtime/symtab.go:pcvalue
		sb.WriteString(fmt.Sprintf("%s,0x%x:\n", PCDATA_TO_STR[ind], off))
		if off == 0 {
			continue
		}
		p, err := pctabAt(datap, off)
		if err != nil {
			sb.WriteString(fmt.Sprintf("\tERROR: %v\n", err))
			continue
		}

		val := int32(-1)
		pc := uint64(f.entry())
		entry_pc := pc
		for {
			var ok bool
			prev_pc := pc
			ok, err = stepBounded(&p, &pc, &val, pc == entry_pc, arch)
			if err != nil {
				sb.WriteString(fmt.Sprintf("\tERROR: invalid pctab entry at offset %d: %v\n", off, err))
				break
			}
			if !ok {
				break
			}
			sb.WriteString(fmt.Sprintf("\t0x%x-0x%x:%d\n", prev_pc, pc-1, val))
			switch ind {
			case abi.PCDATA_InlTreeIndex:
				{
					if val > maxInlTreeIndex {
						maxInlTreeIndex = val
					}
				}
			case abi.PCDATA_ArgLiveIndex:
				{
					if val > maxArgLiveIndex {
						maxArgLiveIndex = val
					}
				}
			default:
			}
		}
	}

	inlinedCallSize := uint64(unsafe.Sizeof(*(*inlinedCall)(nil)))
	for _, ind := range FUNCDATA_IND {
		base := funcdata(f, ind)
		if base == nil {
			off := 0
			sb.WriteString(fmt.Sprintf("%s,0x%x\n", FUNCDATA_TO_STR[ind], off))
			continue
		} else {
			off := uint32(uintptr(base) - f.datap.gofunc)
			sb.WriteString(fmt.Sprintf("%s,0x%x:\n", FUNCDATA_TO_STR[ind], off))
		}

		switch ind {
		case abi.FUNCDATA_ArgsPointerMaps:
			{
				stkmap := (*stackmap)(unsafe.Pointer(base))
				sb.WriteString(fmt.Sprintf("\t%s\n", (*stkmap).String()))
			}
		case abi.FUNCDATA_LocalsPointerMaps:
			{
				stkmap := (*stackmap)(unsafe.Pointer(base))
				sb.WriteString(fmt.Sprintf("\t%s\n", (*stkmap).String()))
			}
		case abi.FUNCDATA_StackObjects:
			{
				// see src/runtime/stkframe.go:getStackMap
				p := unsafe.Pointer(base)
				if p != nil {
					n := *(*uintptr)(p)
					p = add(p, uintptr(arch.PtrSize))
					r0 := (*stackObjectRecord)(noescape(p))
					objs := unsafe.Slice(r0, int(n))
					for i := int(0); i < int(n); i++ {
						sb.WriteString(fmt.Sprintf("\tOBJ[%d]:%s\n", i, objs[i].String()))
					}
				}
			}
		case abi.FUNCDATA_InlTree:
			{
				for i := uint64(0); i <= uint64(maxInlTreeIndex); i++ {
					entry := *(*inlinedCall)(unsafe.Pointer(uintptr(base) + uintptr(i*inlinedCallSize)))
					name := datap.funcName(entry.nameOff)
					sb.WriteString(fmt.Sprintf("\tINL[%d]: %s (%s)\n", i, entry.String(), name))
				}
			}
		case abi.FUNCDATA_OpenCodedDeferInfo:
			{
				fd := unsafe.Pointer(base)
				if fd == nil {
					continue
				}
				deferBitsOffset, fd := readvarintUnsafe(fd)
				slotsOffset, fd := readvarintUnsafe(fd)

				sb.WriteString(fmt.Sprintf("\tdeferBitsOffset:0x%x,slotsOffset:0x%x\n", deferBitsOffset, slotsOffset))
			}
		case abi.FUNCDATA_ArgInfo:
			{
				// see src/runtime/traceback.go:printArgs
				const (
					_endSeq         = 0xff
					_startAgg       = 0xfe
					_endAgg         = 0xfd
					_dotdotdot      = 0xfc
					_offsetTooLarge = 0xfb
				)
				const (
					limit    = 10                       // print no more than 10 args/components
					maxDepth = 5                        // no more than 5 layers of nesting
					maxLen   = (maxDepth*3+2)*limit + 1 // max length of _FUNCDATA_ArgInfo (see the compiler side for reasoning)
				)
				for liveIdx := uint64(0); liveIdx <= uint64(maxArgLiveIndex); liveIdx++ {
					sb.WriteString(fmt.Sprintf("\tliveIdx[%d]: ", liveIdx))
					val := (*[maxLen]uint8)(unsafe.Pointer(base))
					liveInfo := unsafe.Pointer(nil)
					startOffset := uint8(0xff)
					if f._func.nfuncdata > abi.FUNCDATA_ArgLiveInfo {
						argLiveInfoBase := funcdata(f, abi.FUNCDATA_ArgLiveInfo)
						// argLiveInfoOff := funcdata[abi.FUNCDATA_ArgLiveInfo]
						// if argLiveInfoOff != ^uint32(0) {
						if argLiveInfoBase != nil {
							// liveInfo = unsafe.Pointer(firstmoduledata.gofunc + uintptr(argLiveInfoOff))
							startOffset = *(*uint8)(argLiveInfoBase)
						}
					}
					if val != nil {
						pi := 0
						slotIdx := uint8(0)
						start := true
						isLive := func(off, slotIdx uint8) bool {
							if liveInfo == nil || liveIdx <= 0 {
								return true // no liveness info, always live
							}
							if off < startOffset {
								return true
							}
							bits := *(*uint8)(add(liveInfo, uintptr(liveIdx)+uintptr(slotIdx/8)))
							return bits&(1<<(slotIdx%8)) != 0
						}
						print1 := func(off, sz, slotIdx uint8) {
							sb.WriteString(fmt.Sprintf("%d-byte", sz))
							if !isLive(off, slotIdx) {
								sb.WriteString("?")
							}
						}

						printcomma := func() {
							if !start {
								sb.WriteString(", ")
							}
						}
					printloop:
						for {
							o := val[pi]
							pi++
							switch o {
							case _endSeq:
								break printloop
							case _startAgg:
								printcomma()
								sb.WriteString("{")
								start = true
								continue
							case _endAgg:
								sb.WriteString("}")
							case _dotdotdot:
								printcomma()
								sb.WriteString("...")
							case _offsetTooLarge:
								printcomma()
								sb.WriteString("_")
							default:
								printcomma()
								sz := val[pi]
								pi++
								print1(o, sz, slotIdx)
								if o >= startOffset {
									slotIdx++
								}
							}
							start = false
						}
					}
					sb.WriteString("\n")
				}
			}
		case abi.FUNCDATA_ArgLiveInfo:
			{
				// val := *(*uint32)(unsafe.Pointer(base))
				// sb.WriteString(fmt.Sprintf("\tVAL:0x%x\n", val))
			}
		case abi.FUNCDATA_WrapInfo:
			{
				val := *(*uint32)(unsafe.Pointer(base))
				if val != 0 {
					startPC := f.datap.textAddr(val)
					// Check findfunctab bounds before calling findfunc to avoid crash
					if err := checkFindfunctabBounds(startPC); err != nil {
						sb.WriteString(fmt.Sprintf("\tVAL:0x%x PC:0x%x (findfunc error: %s)\n", val, startPC, err.Error()))
					} else {
						wf := findfunc(startPC)
						if wf.Valid() {
							wfuncName := datap.funcName(wf._func.nameOff)
							sb.WriteString(fmt.Sprintf("\tVAL:0x%x PC:0x%x (%s)\n", val, startPC, wfuncName))
						} else {
							sb.WriteString(fmt.Sprintf("\tVAL:0x%x PC:0x%x (findfunc returned invalid)\n", val, startPC))
						}
					}
				} else {
					sb.WriteString(fmt.Sprintf("\tVAL:0x%x\n", val))
				}
			}
		}
	}

	return sb.String()
}

func (p *rtype) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(*p)
	v := reflect.ValueOf(*p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	TypeName := p.string()
	sb.WriteString(fmt.Sprintf("TypeName:%s,", TypeName))
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if field.Name == "TFlag" {
			valAdded := false
			val := value.Uint()
			sb.WriteString(fmt.Sprintf("%s:0x%x{", field.Name, val))
			if val&uint64(abi.TFlagUncommon) != 0 {
				sb.WriteString(fmt.Sprintf("Uncommon"))
				valAdded = true
			}
			if val&uint64(abi.TFlagExtraStar) != 0 {
				if valAdded {
					sb.WriteString("|")
				}
				sb.WriteString(fmt.Sprintf("ExtraStar"))
				valAdded = true
			}
			if val&uint64(abi.TFlagNamed) != 0 {
				if valAdded {
					sb.WriteString("|")
				}
				sb.WriteString(fmt.Sprintf("Named"))
				valAdded = true
			}
			if val&uint64(abi.TFlagRegularMemory) != 0 {
				if valAdded {
					sb.WriteString("|")
				}
				sb.WriteString(fmt.Sprintf("RegularMemory"))
				valAdded = true
			}
			sb.WriteString("}")
		} else if field.Name == "Kind_" {
			val := value.Uint()
			t := (*Type)(unsafe.Pointer(p))
			s := t.Kind().String()
			sb.WriteString(fmt.Sprintf("%s:0x%x{%s}", field.Name, val, s))
		} else {
			switch value.Kind() {
			case reflect.Ptr:
				sb.WriteString(fmt.Sprintf("%s:0x%x", field.Name, uint64(value.Pointer())))
			case reflect.Func:
				sb.WriteString(fmt.Sprintf("%s:<FUNC>", field.Name))
			default:
				sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
			}
		}
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	lt := (*Type)(unsafe.Pointer(p))
	if ut := lt.Uncommon(); ut != nil {
		sb.WriteString(", ")
		switch lt.Kind() {
		case abi.Struct:
			{
				st := lt.StructType()
				lst := (*StructType)(unsafe.Pointer(st))
				sb.WriteString(lst.String())
			}
		default:
			{
				lut := (*UncommonType)(unsafe.Pointer(ut))
				sb.WriteString(lut.String())
			}
		}
	}
	return sb.String()
}

func (p *rtype) StringShort() string {
	var sb strings.Builder
	tName := p.string()
	sb.WriteString("{")
	sb.WriteString(fmt.Sprintf("TypeName:%s, ", tName))
	sb.WriteString(fmt.Sprintf("Size_:%#v, ", p.Size_))
	sb.WriteString(fmt.Sprintf("Hash:%#v", p.Hash))
	sb.WriteString("}")
	return sb.String()
}

func (p StructType) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	_ = v
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		switch value.Kind() {
		case reflect.Ptr:
			sb.WriteString(fmt.Sprintf("%s:0x%x", field.Name, uint64(value.Pointer())))
		case reflect.Slice:
			{
				if field.Name == "Fields" {
					var sf []StructField
					sh := (*reflect.SliceHeader)(unsafe.Pointer(&sf))
					sh.Len = value.Len()
					sh.Cap = value.Cap()
					off := uintptr(value.UnsafePointer()) - typesBase
					sh.Data = uintptr(add(unsafe.Pointer(firstmoduledata.types), off))
					sb.WriteString(fmt.Sprintf("%s:{", field.Name))
					for i := range sf {
						sb.WriteString(fmt.Sprintf("FIELD[%d]:%s", i, sf[i].String()))
						if i < len(sf)-1 {
							sb.WriteString(", ")
						}
					}
					sb.WriteString("}")
				} else {
					sb.WriteString(fmt.Sprintf("%s:{Data:0x%x Len:%d Cap:%d}", field.Name, uintptr(value.UnsafePointer()), value.Len(), value.Cap()))
				}
			}
		case reflect.Struct:
			{
				if field.Name == "Type" { // Skip self
					sb.WriteString(fmt.Sprintf("%s:<TBD1>", field.Name))
					if i < t.NumField()-1 {
						sb.WriteString(", ")
					}
					continue
				} else if field.Name == "PkgPath" {
					N := p.PkgPath
					to := uintptr(unsafe.Pointer(p.PkgPath.Bytes)) - typesBase
					N.Bytes = (*byte)(add(unsafe.Pointer(firstmoduledata.types), to))
					sb.WriteString(fmt.Sprintf("%s:%s", field.Name, N.Name()))
					if i < t.NumField()-1 {
						sb.WriteString(", ")
					}
					continue
				}
				method := value.MethodByName("String")
				if method.IsValid() {
					sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
				}
			}
		default:
			sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		}
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (p StructField) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	_ = v
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		switch value.Kind() {
		case reflect.Struct:
			{
				if field.Name == "Name" {
					N := p.Name
					to := uintptr(unsafe.Pointer(p.Name.Bytes)) - typesBase
					N.Bytes = (*byte)(add(unsafe.Pointer(firstmoduledata.types), to))
					sb.WriteString(fmt.Sprintf("%s:%s", field.Name, N.Name()))
				} else {
					method := value.MethodByName("String")
					if method.IsValid() {
						sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
					} else {
						sb.WriteString(fmt.Sprintf("%s:<TBD2>", field.Name))
					}
				}
			}
		case reflect.Ptr:
			{
				if field.Name == "Typ" {
					to := (uintptr)(unsafe.Pointer(p.Typ)) - typesBase
					rt := toRType((*Type)(add(unsafe.Pointer(firstmoduledata.types), to)))
					sb.WriteString(fmt.Sprintf("%s:%s", field.Name, rt.StringShort()))
				} else {
					sb.WriteString(fmt.Sprintf("%s:0x%x", field.Name, uint64(value.Pointer())))
				}
			}
		default:
			sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		}
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (p *UncommonType) String() string {
	var sb strings.Builder
	t := reflect.TypeOf(*p)
	v := reflect.ValueOf(*p)
	tName := t.String()
	lastDot := strings.LastIndex(tName, ".")
	if lastDot != -1 {
		tName = tName[lastDot+1:]
	}
	sb.WriteString(fmt.Sprintf("%s:{", tName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if field.Name == "_" {
			sb.WriteString(fmt.Sprintf("%s", field.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s:%#v", field.Name, value))
		}
		if i < t.NumField()-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func (it *itab) String() string {
	var sb strings.Builder
	sb.WriteString("itab:{")

	interPtr, err := translateToBuffer(unsafe.Pointer(it.Inter))
	if err != nil {
		sb.WriteString("Inter:<invalid>, ")
	} else {
		inter := (*InterfaceType)(interPtr)
		sb.WriteString(fmt.Sprintf("Inter:{%s}, ", inter.String()))
	}

	typePtr, err := translateToBuffer(unsafe.Pointer(it.Type))
	if err != nil {
		sb.WriteString("Type:<invalid>, ")
	} else {
		t := (*_type)(typePtr)
		sb.WriteString(fmt.Sprintf("Type:{%s}, ", toRType(t).StringShort()))
	}

	sb.WriteString(fmt.Sprintf("Hash:0x%x, ", it.Hash))
	sb.WriteString(fmt.Sprintf("Fun:[1]uintptr{0x%x}}", it.Fun[0]))

	return sb.String()
}

func (it *InterfaceType) String() string {
	var sb strings.Builder
	sb.WriteString("InterfaceType:{")

	sb.WriteString(fmt.Sprintf("Type:{%s}, ", toRType((*_type)(unsafe.Pointer(&it.Type))).StringShort()))

	pkgPath, err := translateName(it.PkgPath)
	if err != nil {
		sb.WriteString("PkgPath:<invalid>, ")
	} else {
		sb.WriteString(fmt.Sprintf("PkgPath:%s, ", pkgPath.Name()))
	}

	sb.WriteString("Methods:{")
	methods := it.Methods
	if len(methods) > 0 {
		dataPtr, err := translateToBuffer(unsafe.Pointer(unsafe.SliceData(methods)))
		if err != nil {
			sb.WriteString("<invalid methods data>")
		} else {
			elemSize := unsafe.Sizeof(methods[0])
			for i := 0; i < len(methods); i++ {
				if i > 0 {
					sb.WriteString(", ")
				}
				imethodPtr := add(dataPtr, uintptr(i)*elemSize)
				im := (*Imethod)(imethodPtr)
				sb.WriteString(fmt.Sprintf("IMETHOD[%d]:%s", i, im.String()))
			}
		}
	}
	sb.WriteString("}}")

	return sb.String()
}

func (m *Imethod) String() string {
	var sb strings.Builder
	sb.WriteString("Imethod:{")

	var rt rtype
	nameStr := rt.nameOff(m.Name).Name()
	sb.WriteString(fmt.Sprintf("Name:%s, ", nameStr))

	absTypPtr := unsafe.Pointer(typesBase + uintptr(m.Typ))
	typPtr, err := translateToBuffer(absTypPtr)
	if err != nil {
		sb.WriteString("Typ:<invalid>")
	} else {
		T := (*_type)(typPtr)
		sb.WriteString(fmt.Sprintf("Typ:{%s}", toRType(T).StringShort()))
	}

	sb.WriteString("}")
	return sb.String()
}

// PrintModuledata outputs the moduledata structure to w.
func (p *Moduledata) PrintModuledata(w io.Writer) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "%s\n", p.md.String())
	return bw.Flush()
}

// PrintPCHeader outputs the pcHeader structure to w.
func (p *Moduledata) PrintPCHeader(w io.Writer) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "%s\n", p.ph.String())
	return bw.Flush()
}

func (p *Moduledata) PrintHeader(w io.Writer) error {
	if err := p.PrintModuledata(w); err != nil {
		return err
	}
	return p.PrintPCHeader(w)
}

// PrintTypes outputs type information to w. The filter selects types by name,
// and depth controls the level of detail (0=no details, 1=fields/methods, 2+=nested types).
func (p *Moduledata) PrintTypes(w io.Writer, filter *regexp.Regexp, depth int) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "typelinks:\n")

	var errCount int

	for i := int(0); i < len(firstmoduledata.typelinks); i++ {
		offset := firstmoduledata.typelinks[i]

		absTypPtr := unsafe.Pointer(typesBase + uintptr(offset))
		typPtr, err := translateToBuffer(absTypPtr)
		if err != nil {
			fmt.Fprintf(bw, "TYPE[%d]:<error: %v>\n", i, err)
			errCount++
			continue
		}

		t := (*_type)(typPtr)
		rt := toRType(t)
		tName := rt.string()
		if filter != nil && !filter.MatchString(tName) {
			continue
		}

		typ := (*abi.Type)(typPtr)
		fmt.Fprintf(bw, "TYPE[%d]:%s, Kind:%s, Size:%d, Hash:0x%x", i, tName, typ.Kind().String(), typ.Size_, typ.Hash)

		switch typ.Kind() {
		case abi.Struct:
			fmt.Fprintf(bw, ", Align:%d", typ.FieldAlign_)
			st := typ.StructType()
			if st != nil {
				fieldsData, numFields, err := translateSliceData(unsafe.Pointer(&st.Fields), unsafe.Sizeof(StructField{}))
				if err != nil {
					fmt.Fprintf(bw, ", Fields:<error>\n")
					errCount++
				} else {
					fmt.Fprintf(bw, ", Fields:%d, Hash:0x%x\n", numFields, typ.Hash)
					if depth > 0 {
						var prevFieldEnd uintptr
						for j := 0; j < numFields; j++ {
							field := (*StructField)(add(unsafe.Pointer(fieldsData), uintptr(j)*unsafe.Sizeof(StructField{})))
							if j > 0 && field.Offset > prevFieldEnd {
								gap := field.Offset - prevFieldEnd
								fmt.Fprintf(bw, "  [Padding: %d bytes]\n", gap)
							}
							fieldName := translateFieldName(field.Name)
							var fieldTypeName string
							var fieldTypeHash uint32
							var fieldTypeSize uintptr
							var fieldTypeAlign uint8
							fieldTypePtr, fieldTypeErr := translateToBuffer(unsafe.Pointer(field.Typ))
							if fieldTypeErr != nil {
								fieldTypeName = "<error>"
								errCount++
							} else {
								ft := (*abi.Type)(fieldTypePtr)
								fieldTypeName = toRType((*_type)(fieldTypePtr)).string()
								fieldTypeHash = ft.Hash
								fieldTypeSize = ft.Size_
								fieldTypeAlign = ft.Align_
							}
							fmt.Fprintf(bw, "  Field[%d]: Name:%s, Type:%s, Hash:0x%x, Offset:%d, Size:%d, Align:%d\n",
								j, fieldName, fieldTypeName, fieldTypeHash, field.Offset, fieldTypeSize, fieldTypeAlign)
							prevFieldEnd = field.Offset + fieldTypeSize
							if fieldTypeErr == nil && depth > 1 {
								printFieldTypeDetails(bw, fieldTypePtr, 1, &errCount, depth-1)
							}
						}
					}
				}
			} else {
				fmt.Fprintf(bw, "\n")
			}

		case abi.Interface:
			it := typ.InterfaceType()
			if it != nil {
				methodsData, numMethods, err := translateSliceData(unsafe.Pointer(&it.Methods), unsafe.Sizeof(abi.Imethod{}))
				if err != nil {
					fmt.Fprintf(bw, ", Methods:<error>\n")
					errCount++
				} else {
					fmt.Fprintf(bw, ", Methods:%d, Hash:0x%x\n", numMethods, typ.Hash)
					if depth > 0 {
						for j := 0; j < numMethods; j++ {
							method := (*abi.Imethod)(add(unsafe.Pointer(methodsData), uintptr(j)*unsafe.Sizeof(abi.Imethod{})))
							methodName := "<invalid>"
							translatedName, nameErr := translateName(abi.Name{Bytes: (*byte)(unsafe.Pointer(typesBase + uintptr(method.Name)))})
							if nameErr == nil && translatedName.Bytes != nil {
								methodName = translatedName.Name()
							}
							methodTypeName := fmt.Sprintf("<TypeOff %d>", method.Typ)
							var methodTypeHash uint32
							if method.Typ != 0 {
								methodTypePtr := unsafe.Pointer(typesBase + uintptr(method.Typ))
								if uintptr(methodTypePtr) >= uintptr(firstmoduledata.types) && uintptr(methodTypePtr) < uintptr(firstmoduledata.etypes) {
									methodTypeHash = (*abi.Type)(methodTypePtr).Hash
									methodTypeName = toRType((*_type)(methodTypePtr)).string()
								}
							}
							fmt.Fprintf(bw, "  Method[%d]: Name:%s, Type:%s, Hash:0x%x\n",
								j, methodName, methodTypeName, methodTypeHash)
						}
					}
				}
			} else {
				fmt.Fprintf(bw, "\n")
			}

		case abi.Pointer:
			if elem := typ.Elem(); elem != nil {
				elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
				var elemName string
				var elemHash uint32
				if elemErr != nil {
					elemName = "<error>"
					errCount++
				} else {
					elemPtr := (*abi.Type)(elemTranslated)
					elemName = toRType((*_type)(elemTranslated)).string()
					elemHash = elemPtr.Hash
				}
				fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)

				if elemErr == nil && depth > 1 {
					printFieldTypeDetails(bw, elemTranslated, 1, &errCount, depth-1)
				}
			} else {
				fmt.Fprintf(bw, "\n")
			}

		case abi.Slice:
			if elem := typ.Elem(); elem != nil {
				elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
				var elemName string
				var elemHash uint32
				if elemErr != nil {
					elemName = "<error>"
					errCount++
				} else {
					elemPtr := (*abi.Type)(elemTranslated)
					elemName = toRType((*_type)(elemTranslated)).string()
					elemHash = elemPtr.Hash
				}
				fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			} else {
				fmt.Fprintf(bw, "\n")
			}

		case abi.Array:
			fmt.Fprintf(bw, ", Len:%d", typ.Len())
			if elem := typ.Elem(); elem != nil {
				elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
				var elemName string
				var elemHash uint32
				if elemErr != nil {
					elemName = "<error>"
					errCount++
				} else {
					elemPtr := (*abi.Type)(elemTranslated)
					elemName = toRType((*_type)(elemTranslated)).string()
					elemHash = elemPtr.Hash
				}
				fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			} else {
				fmt.Fprintf(bw, "\n")
			}

		case abi.Map:
			var keyHash, elemHash uint32
			if key := typ.Key(); key != nil {
				keyTranslated, keyErr := translateToBuffer(unsafe.Pointer(key))
				var keyName string
				if keyErr != nil {
					keyName = "<error>"
					errCount++
				} else {
					keyPtr := (*abi.Type)(keyTranslated)
					keyName = toRType((*_type)(keyTranslated)).string()
					keyHash = keyPtr.Hash
				}
				fmt.Fprintf(bw, ", Key:%s, KeyHash:0x%x", keyName, keyHash)
			}
			if elem := typ.Elem(); elem != nil {
				elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
				var elemName string
				if elemErr != nil {
					elemName = "<error>"
					errCount++
				} else {
					elemPtr := (*abi.Type)(elemTranslated)
					elemName = toRType((*_type)(elemTranslated)).string()
					elemHash = elemPtr.Hash
				}
				fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			} else {
				fmt.Fprintf(bw, "\n")
			}

		case abi.Chan:
			fmt.Fprintf(bw, ", Dir:%s", chanDirString(typ.ChanDir()))
			if elem := typ.Elem(); elem != nil {
				elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
				var elemName string
				var elemHash uint32
				if elemErr != nil {
					elemName = "<error>"
					errCount++
				} else {
					elemPtr := (*abi.Type)(elemTranslated)
					elemName = toRType((*_type)(elemTranslated)).string()
					elemHash = elemPtr.Hash
				}
				fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			} else {
				fmt.Fprintf(bw, "\n")
			}

		case abi.Func:
			ft := typ.FuncType()
			if ft != nil {
				fmt.Fprintf(bw, ", In:%d, Out:%d, Hash:0x%x\n", ft.NumIn(), ft.NumOut(), typ.Hash)
				if depth > 0 {
					inSlice := ft.InSlice()
					if len(inSlice) > 0 {
						fmt.Fprintf(bw, "  In:[")
						for k := 0; k < len(inSlice); k++ {
							if k > 0 {
								fmt.Fprintf(bw, ", ")
							}
							if inSlice[k] != nil {
								inPtr, inErr := translateToBuffer(unsafe.Pointer(inSlice[k]))
								if inErr != nil {
									fmt.Fprintf(bw, "<error>")
									errCount++
								} else {
									inType := (*abi.Type)(inPtr)
									fmt.Fprintf(bw, "%s:0x%x", toRType((*_type)(inPtr)).string(), inType.Hash)
								}
							}
						}
						fmt.Fprintf(bw, "]\n")
					}
					outSlice := ft.OutSlice()
					if len(outSlice) > 0 {
						fmt.Fprintf(bw, "  Out:[")
						for k := 0; k < len(outSlice); k++ {
							if k > 0 {
								fmt.Fprintf(bw, ", ")
							}
							if outSlice[k] != nil {
								outPtr, outErr := translateToBuffer(unsafe.Pointer(outSlice[k]))
								if outErr != nil {
									fmt.Fprintf(bw, "<error>")
									errCount++
								} else {
									outType := (*abi.Type)(outPtr)
									fmt.Fprintf(bw, "%s:0x%x", toRType((*_type)(outPtr)).string(), outType.Hash)
								}
							}
						}
						fmt.Fprintf(bw, "]\n")
					}
				}
			} else {
				fmt.Fprintf(bw, "\n")
			}

		default:
			fmt.Fprintf(bw, "\n")
		}
	}

	fmt.Fprintf(bw, "\n")
	if err := bw.Flush(); err != nil {
		return err
	}

	if errCount > 0 {
		return fmt.Errorf("encountered %d errors while printing types", errCount)
	}
	return nil
}

func printTypeDetails(bw *bufio.Writer, typPtr unsafe.Pointer, kind abi.Kind, indent int, errCount *int, maxDepth int) {
	if maxDepth < 0 {
		return
	}

	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	typ := (*abi.Type)(typPtr)

	switch kind {
	case abi.Struct:
		fmt.Fprintf(bw, "%sKind:Struct, Size:%d, Align:%d, Hash:0x%x", prefix, typ.Size_, typ.FieldAlign_, typ.Hash)
		st := typ.StructType()
		if st != nil {
			fieldsData, numFields, err := translateSliceData(unsafe.Pointer(&st.Fields), unsafe.Sizeof(StructField{}))
			if err != nil {
				fmt.Fprintf(bw, ", Fields:<error>\n")
				*errCount++
			} else {
				fmt.Fprintf(bw, ", Fields:%d\n%s{\n", numFields, prefix)
				if maxDepth > 0 {
					var prevFieldEnd uintptr
					for j := 0; j < numFields; j++ {
						field := (*StructField)(add(unsafe.Pointer(fieldsData), uintptr(j)*unsafe.Sizeof(StructField{})))
						if j > 0 && field.Offset > prevFieldEnd {
							gap := field.Offset - prevFieldEnd
							fmt.Fprintf(bw, "%s  [Padding: %d bytes]\n", prefix, gap)
						}
						fieldName := translateFieldName(field.Name)
						var fieldTypeName string
						var fieldTypeHash uint32
						var fieldTypeSize uintptr
						var fieldTypeAlign uint8
						fieldTypePtr, fieldTypeErr := translateToBuffer(unsafe.Pointer(field.Typ))
						if fieldTypeErr != nil {
							fieldTypeName = "<error>"
							*errCount++
						} else {
							ft := (*abi.Type)(fieldTypePtr)
							fieldTypeName = toRType((*_type)(fieldTypePtr)).string()
							fieldTypeHash = ft.Hash
							fieldTypeSize = ft.Size_
							fieldTypeAlign = ft.Align_
						}
						fmt.Fprintf(bw, "%s  Field[%d]: Name:%s, Type:%s, Hash:0x%x, Offset:%d, Size:%d, Align:%d\n",
							prefix, j, fieldName, fieldTypeName, fieldTypeHash, field.Offset, fieldTypeSize, fieldTypeAlign)
						prevFieldEnd = field.Offset + fieldTypeSize
						if fieldTypeErr == nil && maxDepth > 1 {
							printFieldTypeDetails(bw, fieldTypePtr, indent+1, errCount, maxDepth-1)
						}
					}
				}
				fmt.Fprintf(bw, "%s}\n", prefix)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Interface:
		fmt.Fprintf(bw, "%sKind:Interface, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		it := typ.InterfaceType()
		if it != nil {
			methodsData, numMethods, err := translateSliceData(unsafe.Pointer(&it.Methods), unsafe.Sizeof(abi.Imethod{}))
			if err != nil {
				fmt.Fprintf(bw, ", Methods:<error>\n")
				*errCount++
			} else {
				fmt.Fprintf(bw, ", Methods:%d\n%s{\n", numMethods, prefix)
				if maxDepth > 0 {
					for j := 0; j < numMethods; j++ {
						method := (*abi.Imethod)(add(unsafe.Pointer(methodsData), uintptr(j)*unsafe.Sizeof(abi.Imethod{})))
						methodName := "<invalid>"
						translatedName, nameErr := translateName(abi.Name{Bytes: (*byte)(unsafe.Pointer(typesBase + uintptr(method.Name)))})
						if nameErr == nil && translatedName.Bytes != nil {
							methodName = translatedName.Name()
						}
						methodTypeName := fmt.Sprintf("<TypeOff %d>", method.Typ)
						var methodTypeHash uint32
						if method.Typ != 0 {
							methodTypePtr := unsafe.Pointer(typesBase + uintptr(method.Typ))
							if uintptr(methodTypePtr) >= uintptr(firstmoduledata.types) && uintptr(methodTypePtr) < uintptr(firstmoduledata.etypes) {
								methodTypeHash = (*abi.Type)(methodTypePtr).Hash
								methodTypeName = toRType((*_type)(methodTypePtr)).string()
							}
						}
						fmt.Fprintf(bw, "%s  Method[%d]: Name:%s, Type:%s, Hash:0x%x\n",
							prefix, j, methodName, methodTypeName, methodTypeHash)
					}
				}
				fmt.Fprintf(bw, "%s}\n", prefix)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Map:
		fmt.Fprintf(bw, "%sKind:Map, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		var keyHash, elemHash uint32
		if key := typ.Key(); key != nil {
			keyTranslated, keyErr := translateToBuffer(unsafe.Pointer(key))
			var keyName string
			if keyErr != nil {
				keyName = "<error>"
				*errCount++
			} else {
				keyPtr := (*abi.Type)(keyTranslated)
				keyName = toRType((*_type)(keyTranslated)).string()
				keyHash = keyPtr.Hash
			}
			fmt.Fprintf(bw, ", Key:%s, KeyHash:0x%x", keyName, keyHash)
		}
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Slice:
		fmt.Fprintf(bw, "%sKind:Slice, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Array:
		fmt.Fprintf(bw, "%sKind:Array, Size:%d, Len:%d, Hash:0x%x", prefix, typ.Size_, typ.Len(), typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Chan:
		fmt.Fprintf(bw, "%sKind:Chan, Size:%d, Dir:%s, Hash:0x%x", prefix, typ.Size_, chanDirString(typ.ChanDir()), typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Pointer:
		fmt.Fprintf(bw, "%sKind:Ptr, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Func:
		ft := typ.FuncType()
		if ft != nil {
			fmt.Fprintf(bw, "%sKind:Func, Size:%d, In:%d, Out:%d, Hash:0x%x\n", prefix, typ.Size(), ft.NumIn(), ft.NumOut(), typ.Hash)
		} else {
			fmt.Fprintf(bw, "%sKind:Func, Size:%d, Hash:0x%x\n", prefix, typ.Size(), typ.Hash)
		}

	default:
		fmt.Fprintf(bw, "%sKind:%s, Size:%d, Hash:0x%x\n", prefix, kind.String(), typ.Size(), typ.Hash)
	}
}

func printFieldTypeDetails(bw *bufio.Writer, typPtr unsafe.Pointer, indent int, errCount *int, maxDepth int) {
	if maxDepth < 0 {
		return
	}

	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	typ := (*abi.Type)(typPtr)
	kind := typ.Kind()

	switch kind {
	case abi.Struct:
		fmt.Fprintf(bw, "%sKind:Struct, Size:%d, Align:%d, Hash:0x%x", prefix, typ.Size_, typ.FieldAlign_, typ.Hash)
		st := typ.StructType()
		if st != nil {
			fieldsData, numFields, err := translateSliceData(unsafe.Pointer(&st.Fields), unsafe.Sizeof(StructField{}))
			if err != nil {
				fmt.Fprintf(bw, ", Fields:<error>\n")
				*errCount++
			} else {
				fmt.Fprintf(bw, ", Fields:%d\n%s{\n", numFields, prefix)
				if maxDepth > 0 {
					var prevFieldEnd uintptr
					for j := 0; j < numFields; j++ {
						field := (*StructField)(add(unsafe.Pointer(fieldsData), uintptr(j)*unsafe.Sizeof(StructField{})))
						if j > 0 && field.Offset > prevFieldEnd {
							gap := field.Offset - prevFieldEnd
							fmt.Fprintf(bw, "%s  [Padding: %d bytes]\n", prefix, gap)
						}
						fieldName := translateFieldName(field.Name)
						var fieldTypeName string
						var fieldTypeHash uint32
						var fieldTypeSize uintptr
						var fieldTypeAlign uint8
						fieldTypePtr, fieldTypeErr := translateToBuffer(unsafe.Pointer(field.Typ))
						if fieldTypeErr != nil {
							fieldTypeName = "<error>"
							*errCount++
						} else {
							ft := (*abi.Type)(fieldTypePtr)
							fieldTypeName = toRType((*_type)(fieldTypePtr)).string()
							fieldTypeHash = ft.Hash
							fieldTypeSize = ft.Size_
							fieldTypeAlign = ft.Align_
						}
						fmt.Fprintf(bw, "%s  Field[%d]: Name:%s, Type:%s, Hash:0x%x, Offset:%d, Size:%d, Align:%d\n",
							prefix, j, fieldName, fieldTypeName, fieldTypeHash, field.Offset, fieldTypeSize, fieldTypeAlign)
						prevFieldEnd = field.Offset + fieldTypeSize
						if fieldTypeErr == nil && maxDepth > 1 {
							printFieldTypeDetails(bw, fieldTypePtr, indent+1, errCount, maxDepth-1)
						}
					}
				}
				fmt.Fprintf(bw, "%s}\n", prefix)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Interface:
		fmt.Fprintf(bw, "%sKind:Interface, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		it := typ.InterfaceType()
		if it != nil {
			methodsData, numMethods, err := translateSliceData(unsafe.Pointer(&it.Methods), unsafe.Sizeof(abi.Imethod{}))
			if err != nil {
				fmt.Fprintf(bw, ", Methods:<error>\n")
				*errCount++
			} else {
				fmt.Fprintf(bw, ", Methods:%d\n%s{\n", numMethods, prefix)
				if maxDepth > 0 {
					for j := 0; j < numMethods; j++ {
						method := (*abi.Imethod)(add(unsafe.Pointer(methodsData), uintptr(j)*unsafe.Sizeof(abi.Imethod{})))
						methodName := "<invalid>"
						translatedName, nameErr := translateName(abi.Name{Bytes: (*byte)(unsafe.Pointer(typesBase + uintptr(method.Name)))})
						if nameErr == nil && translatedName.Bytes != nil {
							methodName = translatedName.Name()
						}
						methodTypeName := fmt.Sprintf("<TypeOff %d>", method.Typ)
						var methodTypeHash uint32
						if method.Typ != 0 {
							methodTypePtr := unsafe.Pointer(typesBase + uintptr(method.Typ))
							if uintptr(methodTypePtr) >= uintptr(firstmoduledata.types) && uintptr(methodTypePtr) < uintptr(firstmoduledata.etypes) {
								methodTypeHash = (*abi.Type)(methodTypePtr).Hash
								methodTypeName = toRType((*_type)(methodTypePtr)).string()
							}
						}
						fmt.Fprintf(bw, "%s  Method[%d]: Name:%s, Type:%s, Hash:0x%x\n",
							prefix, j, methodName, methodTypeName, methodTypeHash)
					}
				}
				fmt.Fprintf(bw, "%s}\n", prefix)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Slice:
		fmt.Fprintf(bw, "%sKind:Slice, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				elemKind := (*abi.Type)(elemTranslated).Kind()
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
				_ = elemKind
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Array:
		fmt.Fprintf(bw, "%sKind:Array, Size:%d, Len:%d, Hash:0x%x", prefix, typ.Size_, typ.Len(), typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Map:
		fmt.Fprintf(bw, "%sKind:Map, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		var keyHash, elemHash uint32
		if key := typ.Key(); key != nil {
			keyTranslated, keyErr := translateToBuffer(unsafe.Pointer(key))
			var keyName string
			if keyErr != nil {
				keyName = "<error>"
				*errCount++
			} else {
				keyPtr := (*abi.Type)(keyTranslated)
				keyName = toRType((*_type)(keyTranslated)).string()
				keyHash = keyPtr.Hash
			}
			fmt.Fprintf(bw, ", Key:%s, KeyHash:0x%x", keyName, keyHash)
		}
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Chan:
		fmt.Fprintf(bw, "%sKind:Chan, Size:%d, Dir:%s, Hash:0x%x", prefix, typ.Size_, chanDirString(typ.ChanDir()), typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Pointer:
		fmt.Fprintf(bw, "%sKind:Ptr, Size:%d, Hash:0x%x", prefix, typ.Size_, typ.Hash)
		if elem := typ.Elem(); elem != nil {
			elemTranslated, elemErr := translateToBuffer(unsafe.Pointer(elem))
			var elemName string
			var elemHash uint32
			if elemErr != nil {
				elemName = "<error>"
				*errCount++
			} else {
				elemPtr := (*abi.Type)(elemTranslated)
				elemName = toRType((*_type)(elemTranslated)).string()
				elemHash = elemPtr.Hash
			}
			fmt.Fprintf(bw, ", Elem:%s, ElemHash:0x%x\n", elemName, elemHash)
			if elemErr == nil && maxDepth > 1 {
				elemKind := (*abi.Type)(elemTranslated).Kind()
				printFieldTypeDetails(bw, elemTranslated, indent+1, errCount, maxDepth-1)
				_ = elemKind
			}
		} else {
			fmt.Fprintf(bw, "\n")
		}

	case abi.Func:
		ft := typ.FuncType()
		if ft != nil {
			fmt.Fprintf(bw, "%sKind:Func, Size:%d, In:%d, Out:%d, Hash:0x%x\n", prefix, typ.Size(), ft.NumIn(), ft.NumOut(), typ.Hash)
		} else {
			fmt.Fprintf(bw, "%sKind:Func, Size:%d, Hash:0x%x\n", prefix, typ.Size(), typ.Hash)
		}

	case abi.String:
		fmt.Fprintf(bw, "%sKind:String, Size:%d, Hash:0x%x\n", prefix, typ.Size_, typ.Hash)

	default:
		fmt.Fprintf(bw, "%sKind:%s, Size:%d, Hash:0x%x\n", prefix, kind.String(), typ.Size(), typ.Hash)
	}
}

func chanDirString(dir abi.ChanDir) string {
	switch dir {
	case abi.RecvDir:
		return "recv"
	case abi.SendDir:
		return "send"
	case abi.BothDir:
		return "both"
	default:
		return "unknown"
	}
}

// PrintITabLinks outputs interface vtable (itab) links to w.
// Each itab links a concrete type to an interface type.
func (p *Moduledata) PrintITabLinks(w io.Writer, filter *regexp.Regexp) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "itablinks:\n")
	for i := int(0); i < len(firstmoduledata.itablinks); i++ {
		itabPtr := firstmoduledata.itablinks[i]
		if itabPtr == nil {
			continue
		}
		ptr, err := translateToBuffer(unsafe.Pointer(itabPtr))
		if err != nil {
			fmt.Fprintf(bw, "ITAB[%d]:<error: %v>\n", i, err)
			if ferr := bw.Flush(); ferr != nil {
				return ferr
			}
			return err
		}
		lit := (*itab)(ptr)
		fmt.Fprintf(bw, "ITAB[%d]:%s\n", i, lit.String())
	}
	fmt.Fprintf(bw, "\n")
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

// PrintFindfunctab outputs the findfunctab bucket table to w.
// This table is used for PC-to-function lookup during binary execution.
func (p *Moduledata) PrintFindfunctab(w io.Writer) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "findfunctab:\n")

	expectedBuckets := int((firstmoduledata.maxpc - firstmoduledata.minpc) / FuncTabBucketSize)
	bucketSize := int(unsafe.Sizeof(findfuncbucket{}))
	actualBuckets := len(findfunctabBuf) / bucketSize

	nbuckets := expectedBuckets
	if nbuckets > actualBuckets {
		nbuckets = actualBuckets
	}

	// Calculate subbuckets in last bucket
	n := int((firstmoduledata.maxpc - firstmoduledata.minpc + SUBBUCKETSIZE - 1) / SUBBUCKETSIZE)
	lastBucketSubbuckets := n - (nbuckets-1)*SUBBUCKETS
	if lastBucketSubbuckets < 0 {
		lastBucketSubbuckets = 0
	}
	if lastBucketSubbuckets > SUBBUCKETS {
		lastBucketSubbuckets = SUBBUCKETS
	}

	for i := 0; i < nbuckets; i++ {
		ffb := (*findfuncbucket)(add(unsafe.Pointer(firstmoduledata.findfunctab), uintptr(i)*unsafe.Sizeof(findfuncbucket{})))
		basePC := firstmoduledata.minpc + uintptr(i)*FuncTabBucketSize
		fmt.Fprintf(bw, "BUCKET[%d] pc=0x%x idx=%d subbuckets=[", i, basePC, ffb.idx)

		subbucketsToPrint := SUBBUCKETS
		if i == nbuckets-1 {
			subbucketsToPrint = lastBucketSubbuckets
		}
		for j := 0; j < subbucketsToPrint; j++ {
			if j > 0 {
				fmt.Fprintf(bw, " ")
			}
			fmt.Fprintf(bw, "%d", ffb.subbuckets[j])
		}
		if subbucketsToPrint < SUBBUCKETS {
			fmt.Fprintf(bw, " ... (%d/%d subbuckets)", subbucketsToPrint, SUBBUCKETS)
		}
		fmt.Fprintf(bw, "]\n")

		if i == actualBuckets-1 && expectedBuckets > actualBuckets {
			fmt.Fprintf(bw, "\nWARNING: expected %d buckets but buffer only has %d (data truncated beyond this point)\n\n", expectedBuckets, actualBuckets)
		}
	}
	fmt.Fprintf(bw, "\n\n")
	return bw.Flush()
}

// PrintFunc outputs function information to w within the specified PC range.
// The filter selects functions by name. When printFuncData is enabled,
// also outputs funcdata such as stack maps and inlining information.
func (p *Moduledata) PrintFunc(w io.Writer, filter *regexp.Regexp, start, end uint64) error {
	bw := bufio.NewWriter(w)
	nftab := len(firstmoduledata.ftab) - 1
	for i := 0; i < nftab; i++ {
		f := FuncInfo{(*_func)(unsafe.Pointer(&firstmoduledata.pclntable[firstmoduledata.ftab[i].funcoff])), &firstmoduledata}
		datap := f.datap
		funcStart := uint64(f.entry())
		funcEnd := uint64(datap.textAddr(firstmoduledata.ftab[i+1].entryoff))
		if funcEnd <= start || end <= funcStart {
			continue
		}
		funcName := datap.funcName(f._func.nameOff)
		if filter != nil && !filter.MatchString(funcName) {
			continue
		}
		var funcStr string
		if printFuncData {
			funcStr = f.String()
		} else {
			funcStr = f.StringBrief()
		}
		fmt.Fprintf(bw, "ftabOff:0x%x,%s\n", firstmoduledata.ftab[i].funcoff, funcStr)
	}

	return bw.Flush()
}

// Original function src/runtime/symtab.go:pcdatastart
func pcdatastart(f FuncInfo, table uint32) uint32 {
	return *(*uint32)(add(unsafe.Pointer(&f.nfuncdata), unsafe.Sizeof(f.nfuncdata)+uintptr(table)*4))
}

// Original function src/runtime/symtab.go:funcdata
func funcdata(f FuncInfo, i uint8) unsafe.Pointer {
	if i < 0 || i >= f.nfuncdata {
		return nil
	}
	base := f.datap.gofunc // load gofunc address early so that we calculate during cache misses
	p := uintptr(unsafe.Pointer(&f.nfuncdata)) + unsafe.Sizeof(f.nfuncdata) + uintptr(f.npcdata)*4 + uintptr(i)*4
	off := *(*uint32)(unsafe.Pointer(p))
	// Return off == ^uint32(0) ? 0 : f.datap.gofunc + uintptr(off), but without branches.
	// The compiler calculates mask on most architectures using conditional assignment.
	var mask uintptr
	if off == ^uint32(0) {
		mask = 1
	}
	mask--
	raw := base + uintptr(off)
	return unsafe.Pointer(raw & mask)
}

func (p *Moduledata) FuncInfoByName(name string) (FuncInfo, bool) {
	nftab := len(firstmoduledata.ftab) - 1
	for i := 0; i < nftab; i++ {
		f := FuncInfo{(*_func)(unsafe.Pointer(&firstmoduledata.pclntable[firstmoduledata.ftab[i].funcoff])), &firstmoduledata}
		datap := f.datap
		funcName := datap.funcName(f._func.nameOff)
		if funcName == name {
			return f, true
		}
	}
	return FuncInfo{}, false
}

func (f FuncInfo) PCDataValue(tableIndex int, pc uint64) int32 {
	if tableIndex < 0 || tableIndex >= int(f._func.npcdata) {
		return 0
	}
	off := pcdatastart(f, uint32(tableIndex))
	if off == 0 {
		return 0
	}
	p, err := pctabAt(f.datap, off)
	if err != nil {
		return 0
	}
	entry_pc := uint64(f.entry())
	pc_cursor := entry_pc
	val := int32(-1)
	for {
		var ok bool
		prev_pc := pc_cursor
		ok, err = stepBounded(&p, &pc_cursor, &val, prev_pc == entry_pc, arch)
		if err != nil {
			return 0
		}
		if !ok {
			break
		}
		if pc >= prev_pc && pc < pc_cursor {
			return val
		}
	}
	return val
}

func (f FuncInfo) PCSPValue(pc uint64) int32 {
	if f._func.pcsp == 0 {
		return 0
	}
	pp, err := pctabAt(f.datap, f._func.pcsp)
	if err != nil {
		return 0
	}
	entry_pc := uint64(f.entry())
	pc_cursor := entry_pc
	val := int32(-1)
	for {
		var ok bool
		prev_pc := pc_cursor
		ok, err = stepBounded(&pp, &pc_cursor, &val, prev_pc == entry_pc, arch)
		if err != nil {
			return 0
		}
		if !ok {
			break
		}
		if pc >= prev_pc && pc < pc_cursor {
			return val
		}
	}
	return val
}
