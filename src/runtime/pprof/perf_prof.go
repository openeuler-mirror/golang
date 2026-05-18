// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pprof

import (
	"io"
	"fmt"
	"time"
	"sync"
	"runtime"
	"os"
	"path/filepath"
	"internal/profile"
)

var (
	minpc, maxpc uintptr
	pcRangeOnce  sync.Once
)

func isValidPC(pc uintptr) bool {
	pcRangeOnce.Do(func() {
		minpc, maxpc = runtime.GetFirstModuledataPcRange()
	})
	return pc >= minpc && pc <= maxpc
}

func getFuncLineByPC(pc uintptr) (file string, line int, fnName string) {
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "", 0, ""
	}
	file, line = f.FileLine(pc)
	fnName = f.Name()
	return
}

func (e *perfBranchEntry) decode() (cycles uint16, misPred, predicted bool) {
	misPred = (e.flags & 0x1) != 0
	predicted = (e.flags & 0x2) != 0
	cycles = uint16((e.flags >> 4) & 0xFFFF)
	return
}

func getLocation(ip uint64, p *profile.Profile, locMap map[uint64]*profile.Location,
					fnMap map[uint64]*profile.Function, nextLocID, nextFnID *uint64,) *profile.Location {
	if loc, ok := locMap[ip]; ok {
		return loc
	}

	var funcName, fileLine string
	pc := uintptr(ip)
	if isValidPC(pc) {
		file, line, fn := getFuncLineByPC(pc)
		if fn != "" {
			funcName = fn
			fileLine = fmt.Sprintf("%s:%d", file, line)
		}
	}

	if funcName == "" {
		funcName = fmt.Sprintf("0x%x", ip)
		fileLine = fmt.Sprintf("0x%x", ip)
	}

	fn, ok := fnMap[ip]
	if !ok {
		fn = &profile.Function{
			ID:         *nextFnID,
			Name:       funcName,
			SystemName: funcName,
			Filename:   fileLine,
		}
		fnMap[ip] = fn
		p.Function = append(p.Function, fn)
		*nextFnID++
	}

	loc := &profile.Location{
		ID:      *nextLocID,
		Address: ip,
		Line: []profile.Line{
			{Function: fn, Line: 1},
		},
	}
	locMap[ip] = loc
	p.Location = append(p.Location, loc)
	*nextLocID++

	return loc
}

func buildLocations(callchain []uint64, p *profile.Profile,
	locMap map[uint64]*profile.Location,
	fnMap map[uint64]*profile.Function,
	nextLocID, nextFnID *uint64) []*profile.Location {

	var locs []*profile.Location
	for _, ip := range callchain {
		locs = append(locs, getLocation(ip, p, locMap, fnMap, nextLocID, nextFnID))
	}
	return locs
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func buildSample(s sample, locs []*profile.Location) *profile.Sample {
	sample := &profile.Sample{
		Location: locs,
		Value:    []int64{0},
		Label:    make(map[string][]string),
	}

	sample.Value[0] = int64(s.period)
	sample.Label["event"] = []string{s.evtName}
	for i, br := range s.branches {
		cycles, misPred, predicted := br.decode()
		branchInfo := fmt.Sprintf("%x %x %d %d %d",
						br.fromAddr, br.toAddr, cycles, btoi(misPred), btoi(predicted))
		key := fmt.Sprintf("%d", i)
		sample.Label[key] = []string{branchInfo}
	}

	return sample
}

func convertSamples(samples []sample, duration int64) *profile.Profile {
	if len(samples) == 0 {
		return nil
	}

	exe, err := os.Executable()
	if err != nil || exe == "" {
		exe = filepath.Join(".", "unknown")
	}

	const mapID = uint64(1) // ≥1

	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "pmu", Unit: "count"},
		},
		PeriodType:        &profile.ValueType{Type: "samples", Unit: "count"},
		Period:            1,
		DefaultSampleType: "pmu",
		TimeNanos:         time.Now().UnixNano(),
		DurationNanos:     duration * 1000000000,  //seconds to nanoseconds
		Mapping: []*profile.Mapping{
			{
				ID:              mapID,
				Start:           0,
				Limit:           ^uint64(0),
				File:            exe,
				HasFunctions:    true,
				HasFilenames:    true,
				HasLineNumbers:  true,
				HasInlineFrames: true,
			},
		},
	}

	locMap := map[uint64]*profile.Location{}
	fnMap := map[uint64]*profile.Function{}
	nextLocID := uint64(1)
	nextFnID := uint64(1)

	for _, s := range samples {
		locs := buildLocations(s.callchain, p, locMap, fnMap, &nextLocID, &nextFnID)
		sample := buildSample(s, locs)
		p.Sample = append(p.Sample, sample)
	}
	return p
}

// writeProf converts PMU profiling samples and writes them to the given writer.
// Returns an error if conversion or writing fails.
func writeProf(w io.Writer, allSamples []sample, duration int64) error {
	prof := convertSamples(allSamples, duration)
	if prof == nil {
		// create a empty PMU profile file
		empty := &profile.Profile{
			SampleType: []*profile.ValueType{
				{Type: "pmu", Unit: "count"},
			},
			TimeNanos:     time.Now().UnixNano(),
			DurationNanos: duration * int64(time.Second),
		}
		if err := empty.Write(w); err != nil {
			return fmt.Errorf("write empty profile failed: %w", err)
		}
		return nil
	}

	if err := prof.Write(w); err != nil {
		return fmt.Errorf("write profile failed: %w", err)
	}

	return nil
}
