// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pprof

const (
	// Syscall number for perf_event_open on ARM64 linux kernel
	sysPerfEventOpen    = 241
	// IOCTL command: disable the selected perf event
	perfEventIocDisable = 0x2401
	// IOCTL command: enable the selected perf event
	perfEventIocEnable  = 0x2400
	// IOCTL command: reset the selected perf event's counters/buffers
	perfEventIocReset   = 0x2403
)

// The page size setting for ring buffer mmap
const (
	defaultSamplePages = 128
	brbeSamplePages     = 1024
)

// PMU collection params for go pprof
type PMUAttr struct {
	Period     uint64
	Freq       uint64
	Duration   int64
	EvtList    []string
	EnableBRBE bool
}

// The definition of a perf event
type perfEventAttr struct {
	evtType            uint32
	size               uint32
	config             uint64
	sample             uint64
	sampleType         uint64
	readFormat         uint64
	bits               uint64
	wakeup             uint32
	bpType             uint32
	ext1               uint64
	ext2               uint64
	branchSampleType   uint64
	sampleRegsUser     uint64
	sampleStackUser    uint32
	clockid            int32
	sampleRegsIntr     uint64
	auxWatermark       uint32
	sampleMaxStack     uint16
	_                  uint16
	auxSampleSize      uint32
	_                  uint32
	sigData            uint64
}

// The definition of the PMU sampling header
type perfEventHeader struct {
	typ  uint32
	misc uint16
	size uint16
}

// The definition of the BRBE data
type perfBranchEntry struct {
	fromAddr uint64
	toAddr   uint64
	flags    uint64
}

// The definition of the page that can be mapped via mmap
type perfEventMmapPage struct {
	version        uint32
	compatVersion  uint32
	lock           uint32
	index          uint32
	offset         int64
	timeEnabled    uint64
	timeRunning    uint64
	capabilities   uint64
	pmcWidth       uint16
	timeShift      uint16
	timeMult       uint32
	timeOffset     uint64
	timeZero       uint64
	size           uint32
	_              uint32
	timeCycles     uint64
	timeMask       uint64
	_              [928]uint8
	dataHead       uint64
	dataTail       uint64
	dataOffset     uint64
	dataSize       uint64
	auxHead        uint64
	auxTail        uint64
	auxOffset      uint64
	auxSize        uint64
}

// The sampling data for statistics
type sample struct {
	ip        uint64
	pid, tid  uint32
	time      uint64
	id        uint64
	period    uint64
	branches  []perfBranchEntry
	callchain []uint64
	evtName   string
}

// Set perfEventAttr.Bits
const (
	perfAttrDisabled                 = 1 << 0
	perfAttrInherit                  = 1 << 1
	perfAttrPinned                   = 1 << 2
	perfAttrExclusive                = 1 << 3
	perfAttrExcludeUser              = 1 << 4
	perfAttrExcludeKernel            = 1 << 5
	perfAttrExcludeHv                = 1 << 6
	perfAttrExcludeIdle              = 1 << 7
	perfAttrMmap                     = 1 << 8
	perfAttrComm                     = 1 << 9
	perfAttrFreq                     = 1 << 10
	perfAttrInheritStat              = 1 << 11
	perfAttrEnableOnExec             = 1 << 12
	perfAttrTask                     = 1 << 13
	perfAttrWatermark                = 1 << 14
	perfAttrPreciseIpMask            = 0x3 << 15
	perfAttrMmapData                 = 1 << 17
	perfAttrSampleIdAll              = 1 << 18
	perfAttrExcludeHost              = 1 << 19
	perfAttrExcludeGuest             = 1 << 20
	perfAttrExcludeCallchainKernel   = 1 << 21
	perfAttrExcludeCallchainUser     = 1 << 22
	perfAttrMmap2                    = 1 << 23
	perfAttrCommExec                 = 1 << 24
	perfAttrUseClockid               = 1 << 25
	perfAttrContextSwitch            = 1 << 26
	perfAttrWriteBackward            = 1 << 27
	perfAttrNamespaces               = 1 << 28
	perfAttrKsymbol                  = 1 << 29
	perfAttrBpfEvent                 = 1 << 30
	perfAttrAuxOutput                = 1 << 31
	perfAttrCgroup                   = 1 << 32
	perfAttrTextPoke                 = 1 << 33
	perfAttrBuildId                  = 1 << 34
	perfAttrInheritThread            = 1 << 35
	perfAttrRemoveOnexec             = 1 << 36
	perfAttrSigtrap                  = 1 << 37
)

// Set perfEventAttr.Sample_type
const (
	perfSampleIp             = 1 << 0
	perfSampleTid            = 1 << 1
	perfSampleTime           = 1 << 2
	perfSampleAddr           = 1 << 3
	perfSampleRead           = 1 << 4
	perfSampleCallchain      = 1 << 5
	perfSampleId             = 1 << 6
	perfSampleCpu            = 1 << 7
	perfSamplePeriod         = 1 << 8
	perfSampleStreamId       = 1 << 9
	perfSampleRaw            = 1 << 10
	perfSampleBranchStack    = 1 << 11
	perfSampleRegsUser       = 1 << 12
	perfSampleStackUser      = 1 << 13
	perfSampleWeight         = 1 << 14
	perfSampleDataSrc        = 1 << 15
	perfSampleIdentifier     = 1 << 16
	perfSampleTransaction    = 1 << 17
	perfSampleRegsIntr       = 1 << 18
	perfSamplePhysAddr       = 1 << 19
	perfSampleAux            = 1 << 20
	perfSampleCgroup         = 1 << 21
	perfSampleDataPageSize   = 1 << 22
	perfSampleCodePageSize   = 1 << 23
	perfSampleWeightStruct   = 1 << 24
	perfSampleMax            = 1 << 25
)

// Set perfEventAttr.Branch_sample_type
const (
	perfSampleBranchUserShift       = 0x0
	perfSampleBranchKernelShift     = 0x1
	perfSampleBranchHvShift         = 0x2
	perfSampleBranchAnyShift        = 0x3
	perfSampleBranchAnyCallShift    = 0x4
	perfSampleBranchAnyReturnShift  = 0x5
	perfSampleBranchIndCallShift    = 0x6
	perfSampleBranchAbortTxShift    = 0x7
	perfSampleBranchInTxShift       = 0x8
	perfSampleBranchNoTxShift       = 0x9
	perfSampleBranchCondShift       = 0xa
	perfSampleBranchCallStackShift  = 0xb
	perfSampleBranchIndJumpShift    = 0xc
	perfSampleBranchCallShift       = 0xd
	perfSampleBranchNoFlagsShift    = 0xe
	perfSampleBranchNoCyclesShift   = 0xf
	perfSampleBranchTypeSaveShift   = 0x10
	perfSampleBranchHwIndexShift    = 0x11
	perfSampleBranchPrivSaveShift   = 0x12
	perfSampleBranchCounters        = 0x80000
	perfSampleBranchMaxShift        = 0x14
	perfSampleBranchUser            = 0x1
	perfSampleBranchKernel          = 0x2
	perfSampleBranchHv              = 0x4
	perfSampleBranchAny             = 0x8
	perfSampleBranchAnyCall         = 0x10
	perfSampleBranchAnyReturn       = 0x20
	perfSampleBranchIndCall         = 0x40
	perfSampleBranchAbortTx         = 0x80
	perfSampleBranchInTx            = 0x100
	perfSampleBranchNoTx            = 0x200
	perfSampleBranchCond            = 0x400
	perfSampleBranchCallStack       = 0x800
	perfSampleBranchIndJump         = 0x1000
	perfSampleBranchCall            = 0x2000
	perfSampleBranchNoFlags         = 0x4000
	perfSampleBranchNoCycles        = 0x8000
	perfSampleBranchTypeSave        = 0x10000
	perfSampleBranchHwIndex         = 0x20000
	perfSampleBranchPrivSave        = 0x40000
	perfSampleBranchMax             = 0x100000
)

// The types of perf record
const (
	perfRecordMmap             = 0x1
	perfRecordLost             = 0x2
	perfRecordComm             = 0x3
	perfRecordExit             = 0x4
	perfRecordThrottle         = 0x5
	perfRecordUnthrottle       = 0x6
	perfRecordFork             = 0x7
	perfRecordRead             = 0x8
	perfRecordSample           = 0x9
	perfRecordMmap2            = 0xa
	perfRecordAux              = 0xb
	perfRecordItraceStart      = 0xc
	perfRecordLostSamples      = 0xd
	perfRecordSwitch           = 0xe
	perfRecordSwitchCpuWide    = 0xf
	perfRecordNamespaces       = 0x10
	perfRecordKsymbol          = 0x11
	perfRecordBpfEvent         = 0x12
	perfRecordCgroup           = 0x13
	perfRecordTextPoke         = 0x14
	perfRecordAuxOutputHwId    = 0x15
	perfRecordMax              = 0x16

	perfRecordKsymbolTypeUnknown = 0x0
	perfRecordKsymbolTypeBpf     = 0x1
	perfRecordKsymbolTypeOol     = 0x2
	perfRecordKsymbolTypeMax     = 0x3
)

// Invalid ip list
const (
	perfContextHv          = -0x20
	perfContextKernel      = -0x80
	perfContextUser        = -0x200
	perfContextGuest       = -0x800
	perfContextGuestKernel = -0x880
	perfContextGuestUser   = -0xa00
	perfContextMax         = -0xfff
)
