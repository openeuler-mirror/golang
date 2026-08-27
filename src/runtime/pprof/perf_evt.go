// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pprof

import (
	"unsafe"
	"fmt"
	"os"
	"bufio"
	"errors"
	"strconv"
	"strings"
	"syscall"
	"sync"
	"sync/atomic"
	"encoding/binary"
)

type pmuEvent struct {
	fd        int
	name      string
	buf       []byte
	meta      *perfEventMmapPage
	data      []byte
	prev      uint64
	recordBuf []byte
}

type pmuList struct {
	pd      int
	attr    PMUAttr
	mu      sync.Mutex
	events  map[int]*pmuEvent // key = fd
	enabled bool
}

var errSetOutput = errors.New("set perf event output")

var (
	pdMu      sync.Mutex
	pdCounter int
	pdMap     = make(map[int]*pmuList)
)

func initMmap(fd int, enableBRBE bool) ([]byte, error){
	//init mmap ring buffer
	samplePages :=  defaultSamplePages
	if enableBRBE {
		samplePages = brbeSamplePages
	}
	// 1 meta data page and N data pages
	mmapSize := (1 + samplePages) * os.Getpagesize()
	buf, err := syscall.Mmap(fd, 0, mmapSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func perfEventOpen(attr *perfEventAttr, pid, cpu, groupFd, flags int) (fd int, err error) {
	r0, _, e1 := syscall.Syscall6(
		sysPerfEventOpen,
		uintptr(unsafe.Pointer(attr)),
		uintptr(pid),
		uintptr(cpu),
		uintptr(groupFd),
		uintptr(flags),
		0,
	)
	fd = int(r0)
	if e1 != 0 {
		return -1, errnoErr(e1)
	}
	return fd, nil
}

func beginSampling(evtName string, inputAttr *PMUAttr, tid, cpu int) (int, error) {
	var attr perfEventAttr
	evtConfig, evtType := getCoreEvent(evtName)
	if evtConfig == -1 || evtType == -1 {
		return -1, fmt.Errorf("Can not get the config and type of event: %s", evtName)
	}
	attr.config = uint64(evtConfig)
	attr.evtType = uint32(evtType)
	attr.size = uint32(unsafe.Sizeof(attr))
	attr.bits = perfAttrSampleIdAll | perfAttrDisabled | perfAttrInherit | perfAttrExcludeGuest |
		perfAttrExcludeKernel | perfAttrExcludeHv |
		perfAttrMmap | perfAttrComm | perfAttrPinned | perfAttrTask | perfAttrMmap2
	if inputAttr.Period > 0 {
		attr.sample = inputAttr.Period
	} else {
		attr.sample = inputAttr.Freq
		attr.bits |= perfAttrFreq
	}
	attr.sampleType = getSampleType(inputAttr.EnableBRBE)
	if inputAttr.EnableBRBE {
		attr.branchSampleType = perfSampleBranchAny | perfSampleBranchUser
	}

	fd, err := perfEventOpen(&attr, tid, cpu, -1, 0)
	if fd < 0 {
		return -1, err
	}

	return fd, nil
}

func newPd() int {
	pdMu.Lock()
	defer pdMu.Unlock()

	if pdCounter == int(^uint(0)>>1) {
		return -1
	}

	pd := pdCounter
	pdCounter++
	return pd
}

func freePd(pd int) error {
	pdMu.Lock()
	pl, ok := pdMap[pd]
	if ok {
		delete(pdMap, pd)
	}
	pdMu.Unlock()
	if !ok {
		return nil
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()

	var errs []error
	for _, e := range pl.events {
		if len(e.buf) > 0 {
			if err := syscall.Munmap(e.buf); err != nil {
				errs = append(errs, fmt.Errorf("munmap fd %d: %w", e.fd, err))
			}
			e.buf = nil
			e.meta = nil
			e.data = nil
		}

		if e.fd >= 0 {
			if err := syscall.Close(e.fd); err != nil {
				errs = append(errs, fmt.Errorf("close fd %d: %w", e.fd, err))
			}
			e.fd = -1
		}
	}

	return errors.Join(errs...)
}

func getPmuList(pd int) (*pmuList, bool) {
	pdMu.Lock()
	defer pdMu.Unlock()
	pl, ok := pdMap[pd]
	return pl, ok
}

func checkTimingParam(EvtList []string, period uint64, freq uint64) error {
	if period > 0 && freq > 0 {
		return fmt.Errorf("Invalid param: only one of 'Period' or 'Freq' can be set for PMU collection")
	}

	if period == 0 && freq == 0 {
		return fmt.Errorf("Invalid param: a valid 'Period' or 'Freq' must be provided for PMU collection")
	}

	if freq > 0 {  // check max sample rate
		const sysSampleRate = "/proc/sys/kernel/perf_event_max_sample_rate"
		data, err := os.ReadFile(sysSampleRate)
		if err != nil {
			return fmt.Errorf("Get perf_event_max_sample_rate failed, cannot validate the perdiod value")
		}
	
		text := strings.TrimSpace(string(data))
		maxRate, err := strconv.ParseUint(text, 10, 64)
		if err != nil {
			return fmt.Errorf("Cannot parse the value of perf_event_max_sample_rate file")
		}
	
		if freq > maxRate {
			return fmt.Errorf("Invalid sample rate, please check /proc/sys/kernel/perf_event_max_sample_rate file")
		}
	}

	return nil
}

func checkAttr(inputAttr *PMUAttr) error {
	var cpuType = getCpuType()
	if cpuType == undefinedType {
		return fmt.Errorf("Unsupported architecture for PMU collection")
	}

	if len(inputAttr.EvtList) == 1 && inputAttr.EvtList[0] == "spe" {
		return fmt.Errorf("SPE collection not support now")
	}

	if err := checkTimingParam(inputAttr.EvtList, inputAttr.Period, inputAttr.Freq); err != nil {
		return err
	}

	if inputAttr.EnableBRBE == true {
		if len(inputAttr.EvtList) != 1 || inputAttr.EvtList[0] != "cycles" {
			return fmt.Errorf("In BRBE collection, only cycles can be set for event")
		}
		if cpuType != hipF && cpuType != hipG {
			return fmt.Errorf("The current cpu type does not support BRBE collection")
		}
	}

	coreEvents := queryCoreEvent()
	for _, evtName := range inputAttr.EvtList {
		if !containsCoreEvent(coreEvents, evtName) {
			return fmt.Errorf("Unknown perf core event: %s", evtName)
		}
	}
	return nil
}

func getTids(pid int) ([]int, error) {
	taskDir := fmt.Sprintf("/proc/%d/task", pid)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, err
	}
	tids := []int{}
	for _, e := range entries {
		if tid, err := strconv.Atoi(e.Name()); err == nil {
			tids = append(tids, tid)
		}
	}
	return tids, nil
}

func getAllowedCPUs() ([]int, error) {
	status, err := os.Open("/proc/self/status")
	if err != nil {
		return nil, err
	}
	defer status.Close()

	const prefix = "Cpus_allowed_list:"
	scanner := bufio.NewScanner(status)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return parseCPUList(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%s not found in /proc/self/status", prefix)
}

func parseCPUList(list string) ([]int, error) {
	if list == "" {
		return nil, errors.New("CPU list is empty")
	}

	cpus := make([]int, 0)
	for _, part := range strings.Split(list, ",") {
		startText, endText, hasRange := strings.Cut(strings.TrimSpace(part), "-")
		start, err := strconv.Atoi(startText)
		if err != nil || start < 0 {
			return nil, errors.New("invalid CPU list")
		}
		end := start
		if hasRange {
			end, err = strconv.Atoi(endText)
			if err != nil || end < start {
				return nil, errors.New("invalid CPU list")
			}
		}

		for cpu := start; ; cpu++ {
			cpus = append(cpus, cpu)
			if cpu == end {
				break
			}
		}
	}
	return cpus, nil
}

type openErrGroup struct {
	event  string
	reason string
	count  int
}

func summarizeOpenErrors(groups map[string]*openErrGroup) error {
	if len(groups) == 0 {
		return nil
	}

	msgs := make([]string, 0, len(groups))
	for _, g := range groups {
		msgs = append(msgs,
			fmt.Sprintf("event %s failed %d times: %s", g.event, g.count, g.reason),
		)
	}

	return errors.New(strings.Join(msgs, "; "))
}

type eventOpenError struct {
	event string
	err   error
}

func perfEventIoctl(fd int, command uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), command, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func perfEventSetOutput(fd, outputFd int) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		perfEventIocSetOutput,
		uintptr(outputFd),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func closePmuEvent(evt *pmuEvent) error {
	var errs []error
	if len(evt.buf) > 0 {
		if err := syscall.Munmap(evt.buf); err != nil {
			errs = append(errs, err)
		}
		evt.buf = nil
		evt.meta = nil
		evt.data = nil
	}
	if evt.fd >= 0 {
		if err := syscall.Close(evt.fd); err != nil {
			errs = append(errs, err)
		}
		evt.fd = -1
	}
	return errors.Join(errs...)
}

func (pl *pmuList) newEvent(evtName string, tid, cpu, outputFd int) (*pmuEvent, error) {
	fd, err := beginSampling(evtName, &pl.attr, tid, cpu)
	if err != nil || fd < 0 {
		if err == nil {
			err = fmt.Errorf("beginSampling returned invalid fd")
		}
		return nil, err
	}

	evt := &pmuEvent{
		fd:   fd,
		name: evtName,
	}

	// only the first TID for an event/CPU pair owns an mmap ring
	if outputFd >= 0 {
		if err := perfEventSetOutput(fd, outputFd); err != nil {
			closeErr := closePmuEvent(evt)
			return nil, errors.Join(fmt.Errorf("%w fd %d: %w", errSetOutput, outputFd, err), closeErr)
		}
	} else {
		buf, err := initMmap(fd, pl.attr.EnableBRBE)
		if err != nil {
			closeErr := closePmuEvent(evt)
			return nil, errors.Join(fmt.Errorf("initMmap failed: %w", err), closeErr)
		}
		evt.buf = buf
		evt.meta = (*perfEventMmapPage)(unsafe.Pointer(&buf[0]))
		dataOffset := evt.meta.dataOffset
		dataSize := evt.meta.dataSize
		if dataOffset > uint64(len(buf)) || dataSize > uint64(len(buf))-dataOffset {
			closeErr := closePmuEvent(evt)
			return nil, errors.Join(fmt.Errorf("invalid perf mmap data range"), closeErr)
		}
		evt.data = buf[dataOffset : dataOffset+dataSize]
	}

	if pl.enabled {
		if err := perfEventIoctl(fd, perfEventIocReset); err != nil {
			closeErr := closePmuEvent(evt)
			return nil, errors.Join(fmt.Errorf("reset new event: %w", err), closeErr)
		}
		if err := perfEventIoctl(fd, perfEventIocEnable); err != nil {
			closeErr := closePmuEvent(evt)
			return nil, errors.Join(fmt.Errorf("enable new event: %w", err), closeErr)
		}
	}

	return evt, nil
}

// pmuOpen initializes a new PMU context with the given attributes.
// It opens all requested events and mmaps one ring per event/CPU pair,
// returning a pd handle or an error if setup fails.
func pmuOpen(pmuInputAttr *PMUAttr) (int, error) {
	if err := checkAttr(pmuInputAttr); err != nil {
		return -1, err
	}

	pd := newPd()
	if pd == -1 {
		return -1, fmt.Errorf("no available pd")
	}

	attr := *pmuInputAttr
	attr.EvtList = append([]string(nil), pmuInputAttr.EvtList...)
	pl := &pmuList{
		pd:     pd,
		attr:   attr,
		events: make(map[int]*pmuEvent),
	}

	pdMu.Lock()
	pdMap[pd] = pl
	pdMu.Unlock()
	cleanupAndReturn := func(mainErr error) (int, error) {
		if cleanupErr := freePd(pd); cleanupErr != nil {
			return -1, errors.Join(mainErr, fmt.Errorf("free pd %d failed: %w", pd, cleanupErr))
		}
		return -1, mainErr
	}

	tids, err := getTids(os.Getpid())
	if err != nil {
		return cleanupAndReturn(fmt.Errorf("get tids failed: %w", err))
	}
	cpus, err := getAllowedCPUs()
	if err != nil {
		return cleanupAndReturn(fmt.Errorf("get allowed CPUs failed: %w", err))
	}

	errGroups := make(map[string]*openErrGroup)
	var setOutputErrs []error
	addErr := func(openErr eventOpenError) {
		reason := openErr.err.Error()
		key := openErr.event + "|" + reason
		group, ok := errGroups[key]
		if !ok {
			group = &openErrGroup{event: openErr.event, reason: reason}
			errGroups[key] = group
		}
		group.count++
	}

	pl.mu.Lock()
	for _, evtName := range pl.attr.EvtList {
		for _, cpu := range cpus {
			outputFd := -1
			for _, tid := range tids {
				evt, err := pl.newEvent(evtName, tid, cpu, outputFd)
				if err != nil {
					addErr(eventOpenError{event: evtName, err: err})
					if errors.Is(err, errSetOutput) {
						setOutputErrs = append(setOutputErrs, err)
					}
					continue
				}
				pl.events[evt.fd] = evt
				if outputFd < 0 {
					outputFd = evt.fd
				}
			}
		}
	}
	eventCount := len(pl.events)
	pl.mu.Unlock()

	if len(setOutputErrs) > 0 {
		return cleanupAndReturn(errors.Join(setOutputErrs...))
	}
	if eventCount > 0 {
		return pd, nil
	}
	if len(errGroups) == 0 {
		return cleanupAndReturn(fmt.Errorf("no event opened"))
	}
	return cleanupAndReturn(summarizeOpenErrors(errGroups))
}

func getSampleType(enableBRBE bool) uint64 {
	base := perfSampleIp | perfSampleTid | perfSampleTime |
		perfSampleId | perfSamplePeriod | perfSampleCallchain

	if enableBRBE {
		base |= perfSampleBranchStack
	}

	return uint64(base)
}

func isValidIp(ip uint64) bool {
	v := int64(ip)
	return v != perfContextHv &&
		v != perfContextKernel &&
		v != perfContextUser &&
		v != perfContextGuest &&
		v != perfContextGuestKernel &&
		v != perfContextGuestUser &&
		v != perfContextMax
}

func parseSample(b []byte, sampleType uint64, name string) sample {
	var s sample
	s.evtName = name
	off := 0

	// perfSampleIp
	if sampleType&perfSampleIp != 0 {
		if len(b)-off >= 8 {
			s.ip = binary.LittleEndian.Uint64(b[off:])
			off += 8
		}
	}

	// perfSampleTid
	if sampleType&perfSampleTid != 0 {
		if len(b)-off >= 8 {
			s.pid = binary.LittleEndian.Uint32(b[off:])
			s.tid = binary.LittleEndian.Uint32(b[off+4:])
			off += 8
		}
	}

	// perfSampleTime
	if sampleType&perfSampleTime != 0 {
		if len(b)-off >= 8 {
			s.time = binary.LittleEndian.Uint64(b[off:])
			off += 8
		}
	}

	// perfSampleId
	if sampleType&perfSampleId != 0 {
		if len(b)-off >= 8 {
			s.id = binary.LittleEndian.Uint64(b[off:])
			off += 8
		}
	}

	// perfSamplePeriod
	if sampleType&perfSamplePeriod != 0 {
		if len(b)-off >= 8 {
			s.period = binary.LittleEndian.Uint64(b[off:])
			off += 8
		}
	}

	// perfSampleCallchain
	if sampleType&perfSampleCallchain != 0 {
		if len(b)-off >= 8 {
			nr := binary.LittleEndian.Uint64(b[off:])
			off += 8
			for i := uint64(0); i < nr; i++ {
				if len(b)-off < 8 {
					break
				}
				ip := binary.LittleEndian.Uint64(b[off:])
				if isValidIp(ip) {
					s.callchain = append(s.callchain, ip)
				}
				off += 8
			}
		}
	}

	// perfSampleBranchStack
	if sampleType&perfSampleBranchStack != 0 {
		if len(b)-off >= 8 {
			nr := binary.LittleEndian.Uint64(b[off:])
			off += 8
			for i := 0; i < int(nr); i++ {
				if len(b)-off < 24 {
					break
				}
				var e perfBranchEntry
				e.fromAddr = binary.LittleEndian.Uint64(b[off:])
				e.toAddr = binary.LittleEndian.Uint64(b[off+8:])
				e.flags = binary.LittleEndian.Uint64(b[off+16:])
				off += 24
				s.branches = append(s.branches, e)
			}
		}
	}

	return s
}

func copyRing(dst, data []byte, pos uint64) {
	offset := int(pos % uint64(len(data)))
	n := copy(dst, data[offset:])
	copy(dst[n:], data[:len(dst)-n])
}

func readRingHeader(data []byte, pos uint64) (perfEventHeader, error) {
	const headerSize = 8
	if len(data) < headerSize {
		return perfEventHeader{}, errors.New("perf ring is smaller than an event header")
	}

	var raw [headerSize]byte
	copyRing(raw[:], data, pos)
	return perfEventHeader{
		typ:  binary.LittleEndian.Uint32(raw[0:4]),
		misc: binary.LittleEndian.Uint16(raw[4:6]),
		size: binary.LittleEndian.Uint16(raw[6:8]),
	}, nil
}

func ringRecord(evt *pmuEvent, pos uint64, size int) []byte {
	if cap(evt.recordBuf) < size {
		evt.recordBuf = make([]byte, size)
	}
	record := evt.recordBuf[:size]
	copyRing(record, evt.data, pos)
	return record
}

func readRingRecords(evt *pmuEvent, start, end, sampleType uint64, handler func(sample)) (uint64, error) {
	headerSize := uint64(unsafe.Sizeof(perfEventHeader{}))
	capacity := uint64(len(evt.data))
	for start != end {
		remaining := end - start
		if remaining < headerSize {
			return start, fmt.Errorf("incomplete perf record header: %d bytes remain", remaining)
		}

		header, err := readRingHeader(evt.data, start)
		if err != nil {
			return start, err
		}
		size := uint64(header.size)
		if size < headerSize || size > capacity || size > remaining {
			return start, fmt.Errorf("invalid perf record size %d with %d bytes remaining", size, remaining)
		}

		record := ringRecord(evt, start, int(size))
		if header.typ == perfRecordSample {
			payload := record[headerSize:size]
			handler(parseSample(payload, sampleType, evt.name))
		}
		start += size
		evt.prev = start
		atomic.StoreUint64(&evt.meta.dataTail, start)
	}
	return start, nil
}

func readSamples(evt *pmuEvent, sampleType uint64, handler func(sample)) error {
	if evt.meta == nil || len(evt.data) == 0 {
		return nil
	}

	head := atomic.LoadUint64(&evt.meta.dataHead)
	start := evt.prev
	if head-start > uint64(len(evt.data)) {
		// Match libkperf's forward-ring behavior: if the reader falls behind
		// the mapped capacity, discard the unread range and resume at head.
		evt.prev = head
		atomic.StoreUint64(&evt.meta.dataTail, head)
		return nil
	}

	_, err := readRingRecords(evt, start, head, sampleType, handler)
	return err
}

// PmuRead collects samples from all events under a pd.
// Returns all gathered samples or an error if reading fails.
func pmuRead(pd int) ([]sample, error) {
	pl, ok := getPmuList(pd)
	if !ok {
		return nil, fmt.Errorf("PmuRead failed. Invalid pd %d", pd)
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()
	sampleType := getSampleType(pl.attr.EnableBRBE)

	var samples []sample
	for _, evt := range pl.events {
		err := readSamples(evt, sampleType, func(s sample) {
			samples = append(samples, s)
		})
		if err != nil {
			return samples, err
		}
	}

	return samples, nil
}

// PmuReset resets all events under a given pd.
// Returns an error if the ioctl reset call fails.
func pmuReset(pd int) error {
	pl, ok := getPmuList(pd)
	if !ok {
		return fmt.Errorf("PmuReset failed. Invalid pd: %d", pd)
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()
	for _, evt := range pl.events {
		if err := perfEventIoctl(evt.fd, perfEventIocReset); err != nil {
			return err
		}
	}
	return nil
}

// PmuEnable enables all events under a given pd.
// Returns an error if the ioctl enable call fails.
func pmuEnable(pd int) error {
	pl, ok := getPmuList(pd)
	if !ok {
		return fmt.Errorf("PmuEnable failed. Invalid pd: %d", pd)
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()
	if pl.enabled {
		return nil
	}

	var enabledEvents []*pmuEvent
	for _, evt := range pl.events {
		if err := perfEventIoctl(evt.fd, perfEventIocEnable); err != nil {
			errs := []error{err}
			for _, enabledEvt := range enabledEvents {
				if disableErr := perfEventIoctl(enabledEvt.fd, perfEventIocDisable); disableErr != nil {
					errs = append(errs, fmt.Errorf("rollback disable fd %d: %w", enabledEvt.fd, disableErr))
				}
			}
			return errors.Join(errs...)
		}
		enabledEvents = append(enabledEvents, evt)
	}
	pl.enabled = true
	return nil
}

// PmuDisable disables all events under a given pd.
// Returns an error if the ioctl disable call fails.
func pmuDisable(pd int) error {
	pl, ok := getPmuList(pd)
	if !ok {
		return fmt.Errorf("PmuDisable failed. Invalid pd: %d", pd)
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()
	if !pl.enabled {
		return nil
	}

	var errs []error
	for _, evt := range pl.events {
		if err := perfEventIoctl(evt.fd, perfEventIocDisable); err != nil {
			errs = append(errs, err)
		}
	}
	pl.enabled = false
	return errors.Join(errs...)
}

// pmuClose closes all perf event file descriptors and unmaps their buffers for a given pd.
// It attempts to clean up all events even if some cleanup operations fail.
func pmuClose(pd int) error {
	pdMu.Lock()
	pl, ok := pdMap[pd]
	if ok {
		delete(pdMap, pd)
	}
	pdMu.Unlock()
	if !ok {
		return fmt.Errorf("PmuClose failed. Invalid pd: %d", pd)
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()

	var errs []error
	for fd, evt := range pl.events {
		if err := closePmuEvent(evt); err != nil {
			errs = append(errs, fmt.Errorf("close event failed for %s(fd=%d): %w", evt.name, fd, err))
		}
	}
	return errors.Join(errs...)
}
