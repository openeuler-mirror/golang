// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pprof

// perfEventAttr hardware config
const (
	perfCountHwCpuCycles              = 0x0
	perfCountHwInstructions           = 0x1
	perfCountHwCacheReferences        = 0x2
	perfCountHwCacheMisses            = 0x3
	perfCountHwBranchInstructions     = 0x4
	perfCountHwBranchMisses           = 0x5
	perfCountHwBusCycles              = 0x6
	perfCountHwStalledCyclesFrontend  = 0x7
	perfCountHwStalledCyclesBackend   = 0x8
	perfCountHwRefCpuCycles           = 0x9
	perfCountHwMax                    = 0xa

	perfCountHwCacheL1d               = 0x0
	perfCountHwCacheL1i               = 0x1
	perfCountHwCacheLl                = 0x2
	perfCountHwCacheDtlb              = 0x3
	perfCountHwCacheItlb              = 0x4
	perfCountHwCacheBpu               = 0x5
	perfCountHwCacheNode              = 0x6
	perfCountHwCacheMax               = 0x7

	perfCountHwCacheOpRead            = 0x0
	perfCountHwCacheOpWrite           = 0x1
	perfCountHwCacheOpPrefetch        = 0x2
	perfCountHwCacheOpMax             = 0x3

	perfCountHwCacheResultAccess      = 0x0
	perfCountHwCacheResultMiss        = 0x1
	perfCountHwCacheResultMax         = 0x2
)

// perfEventAttr software config
const (
	perfCountSwCpuClock               = 0x0
	perfCountSwTaskClock              = 0x1
	perfCountSwPageFaults             = 0x2
	perfCountSwContextSwitches        = 0x3
	perfCountSwCpuMigrations          = 0x4
	perfCountSwPageFaultsMin          = 0x5
	perfCountSwPageFaultsMaj          = 0x6
	perfCountSwAlignmentFaults        = 0x7
	perfCountSwEmulationFaults        = 0x8
	perfCountSwDummy                  = 0x9
	perfCountSwBpfOutput              = 0xa
	perfCountSwMax                    = 0xc
)

// perfEventAttr type
const (
	perfTypeHardware                  = 0x0
	perfTypeSoftware                  = 0x1
	perfTypeTracepoint                = 0x2
	perfTypeHwCache                   = 0x3
	perfTypeRaw                       = 0x4
	perfTypeBreakpoint                = 0x5
	perfTypeMax                       = 0x6
)

const (
	branchMisses            = "branch-misses"
	busCycles               = "bus-cycles"
	cacheMisses             = "cache-misses"
	refCycles               = "ref-cycles"
	branches                = "branches"
	branchInstructions      = "branch-instructions"
	cacheReferences         = "cache-references"
	cpuCycles               = "cpu-cycles"
	cycles                  = "cycles"
	instructions            = "instructions"
	stalledCyclesBackend    = "stalled-cycles-backend"
	stalledCyclesFrontend   = "stalled-cycles-frontend"

	l1DcacheLoadMisses      = "L1-dcache-load-misses"
	l1DcacheLoads           = "L1-dcache-loads"
	l1DcacheStoreMisses     = "L1-dcache-store-misses"
	l1DcacheStores          = "L1-dcache-stores"
	idleCyclesBackend       = "idle-cycles-backend"
	l1IcacheLoadMisses      = "L1-icache-load-misses"
	idleCyclesFrontend      = "idle-cycles-frontend"
	l1IcacheLoads           = "L1-icache-loads"
	llcLoadMisses           = "LLC-load-misses"
	llcLoads                = "LLC-loads"
	llcStoreMisses          = "LLC-store-misses"
	llcStores               = "LLC-stores"
	branchLoadMisses        = "branch-load-misses"
	branchLoads             = "branch-loads"
	dtlbLoadMisses          = "dTLB-load-misses"
	dtlbLoads               = "dTLB-loads"
	dtlbStoreMisses         = "dTLB-store-misses"
	dtlbStores              = "dTLB-stores"
	itlbLoadMisses          = "iTLB-load-misses"
	itlbLoads               = "iTLB-loads"
	nodeLoadMisses          = "node-load-misses"
	nodeLoads               = "node-loads"
	nodeStoreMisses         = "node-store-misses"
	nodeStores              = "node-stores"

	alignmentFaults         = "alignment-faults"
	bpfOutput               = "bpf-output"
	contextSwitches         = "context-switches"
	cs                      = "cs"
	cpuClock                = "cpu-clock"
	cpuMigrations           = "cpu-migrations"
	migrations              = "migrations"
	dummy                   = "dummy"
	emulationFaults         = "emulation-faults"
	majorFaults             = "major-faults"
	minorFaults             = "minor-faults"
	pageFaults              = "page-faults"
	faults                  = "faults"
	taskClock               = "task-clock"
)

type perfEventDesc struct {
	typ    int64
	config int64
}

var coreEventMap = map[string]perfEventDesc{
	// hardware events
	"branch-instructions":     {typ: perfTypeHardware, config: perfCountHwBranchInstructions},
	"branch-misses":           {typ: perfTypeHardware, config: perfCountHwBranchMisses},
	"branches":                {typ: perfTypeHardware, config: perfCountHwBranchInstructions},
	"bus-cycles":              {typ: perfTypeHardware, config: perfCountHwBusCycles},
	"cache-misses":            {typ: perfTypeHardware, config: perfCountHwCacheMisses},
	"cache-references":        {typ: perfTypeHardware, config: perfCountHwCacheReferences},
	"cpu-cycles":              {typ: perfTypeHardware, config: perfCountHwCpuCycles},
	"cycles":                  {typ: perfTypeHardware, config: perfCountHwCpuCycles},
	"idle-cycles-backend":     {typ: perfTypeHardware, config: perfCountHwStalledCyclesBackend},
	"idle-cycles-frontend":    {typ: perfTypeHardware, config: perfCountHwStalledCyclesFrontend},
	"instructions":            {typ: perfTypeHardware, config: perfCountHwInstructions},
	"ref-cycles":              {typ: perfTypeHardware, config: perfCountHwRefCpuCycles},
	"stalled-cycles-backend":  {typ: perfTypeHardware, config: perfCountHwStalledCyclesBackend},
	"stalled-cycles-frontend": {typ: perfTypeHardware, config: perfCountHwStalledCyclesFrontend},

	// software events
	"alignment-faults":        {typ: perfTypeSoftware, config: perfCountSwAlignmentFaults},
	"bpf-output":              {typ: perfTypeSoftware, config: perfCountSwBpfOutput},
	"context-switches":        {typ: perfTypeSoftware, config: perfCountSwContextSwitches},
	"cpu-clock":               {typ: perfTypeSoftware, config: perfCountSwCpuClock},
	"cpu-migrations":          {typ: perfTypeSoftware, config: perfCountSwCpuMigrations},
	"cs":                      {typ: perfTypeSoftware, config: perfCountSwContextSwitches},
	"dummy":                   {typ: perfTypeSoftware, config: perfCountSwDummy},
	"emulation-faults":        {typ: perfTypeSoftware, config: perfCountSwEmulationFaults},
	"faults":                  {typ: perfTypeSoftware, config: perfCountSwPageFaults},
	"major-faults":            {typ: perfTypeSoftware, config: perfCountSwPageFaultsMaj},
	"migrations":              {typ: perfTypeSoftware, config: perfCountSwCpuMigrations},
	"minor-faults":            {typ: perfTypeSoftware, config: perfCountSwPageFaultsMin},
	"page-faults":             {typ: perfTypeSoftware, config: perfCountSwPageFaults},
	"task-clock":              {typ: perfTypeSoftware, config: perfCountSwTaskClock},

	// hardware cache events
	"branch-load-misses":      {typ: perfTypeHwCache, config: 0x10005},
	"branch-loads":            {typ: perfTypeHwCache, config: 0x5},
	"dtlb-load-misses":        {typ: perfTypeHwCache, config: 0x10003},
	"dtlb-loads":              {typ: perfTypeHwCache, config: 0x3},
	"dtlb-store-misses":       {typ: perfTypeHwCache, config: 0x10103},
	"dtlb-stores":             {typ: perfTypeHwCache, config: 0x103},
	"itlb-load-misses":        {typ: perfTypeHwCache, config: 0x10004},
	"itlb-loads":              {typ: perfTypeHwCache, config: 0x4},
	"l1-dcache-load-misses":   {typ: perfTypeHwCache, config: 0x10000},
	"l1-dcache-loads":         {typ: perfTypeHwCache, config: 0x0},
	"l1-dcache-store-misses":  {typ: perfTypeHwCache, config: 0x10100},
	"l1-dcache-stores":        {typ: perfTypeHwCache, config: 0x100},
	"l1-icache-load-misses":   {typ: perfTypeHwCache, config: 0x10001},
	"l1-icache-loads":         {typ: perfTypeHwCache, config: 0x1},
	"llc-load-misses":         {typ: perfTypeHwCache, config: 0x10002},
	"llc-loads":               {typ: perfTypeHwCache, config: 0x2},
	"llc-store-misses":        {typ: perfTypeHwCache, config: 0x10102},
	"llc-stores":              {typ: perfTypeHwCache, config: 0x102},
	"node-load-misses":        {typ: perfTypeHwCache, config: 0x10006},
	"node-loads":              {typ: perfTypeHwCache, config: 0x6},
	"node-store-misses":       {typ: perfTypeHwCache, config: 0x10106},
	"node-stores":             {typ: perfTypeHwCache, config: 0x106},
}
