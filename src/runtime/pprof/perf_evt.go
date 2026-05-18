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
	"sync/atomic"
	"encoding/binary"
)

type pmuEvent struct {
	fd    int
	name  string
	buf   []byte
	meta  *perfEventMmapPage
	data  []byte
}

type pmuList struct {
	pd          int
	events      map[int]*pmuEvent  // key = fd
	enableBRBE  bool
}

var pdCounter int
var pdMap = make(map[int]*pmuList)

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

func beginSampling(evtName string, inputAttr *PMUAttr, pid int, cpu int) (int, error) {
	var attr perfEventAttr
	evtConfig, evtType := getCoreEvent(evtName)
	if evtConfig == -1 || evtType == -1 {
		return -1, fmt.Errorf("Can not get the config and type of event: %s", evtName)
	}
	attr.config = uint64(evtConfig)
	attr.evtType = uint32(evtType)
	attr.size = uint32(unsafe.Sizeof(attr))
	attr.bits = perfAttrSampleIdAll | perfAttrDisabled | perfAttrExcludeGuest | perfAttrInherit |
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

	fd, err := perfEventOpen(&attr, pid, cpu, -1, 0)
	if fd < 0 {
		return -1, err
	}

	return fd, nil
}

func newPd() int {
	for i := 0; i < pdCounter; i++ {
		if _, exists := pdMap[i]; !exists {
			return i
		}
	}

	if pdCounter == int(^uint(0)>>1) {
		return -1
	}

	pd := pdCounter
	pdCounter++
	return pd
}

func freePd(pd int) error {
	pl, ok := pdMap[pd]
	if !ok {
		return nil
	}

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

	delete(pdMap, pd)
	return errors.Join(errs...)
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

func getPhysicalCore() (int, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "processor") {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return count, nil
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

// pmuOpen initializes a new PMU context with the given attributes.
// It opens and mmaps all requested events, returning a pd handle or an error if setup fails.
func pmuOpen(pmuInputAttr *PMUAttr) (int, error) {
	if err := checkAttr(pmuInputAttr); err != nil {
		return -1, err
	}

	pd := newPd()
	if pd == -1 {
		return -1, fmt.Errorf("no available pd")
	}

	pl := &pmuList{
		pd:        pd,
		events:    make(map[int]*pmuEvent),
		enableBRBE: pmuInputAttr.EnableBRBE,
	}

	pdMap[pd] = pl
	cleanupAndReturn := func(mainErr error) (int, error) {
		if cleanupErr := freePd(pd); cleanupErr != nil {
			return -1, errors.Join(
				mainErr,
				fmt.Errorf("free pd %d failed: %w", pd, cleanupErr),
			)
		}
		return -1, mainErr
	}

	tids, err := getTids(os.Getpid())
	if err != nil {
		return cleanupAndReturn(fmt.Errorf("get tids failed: %w", err))
	}
	cpuNum, err := getPhysicalCore()
	if err != nil {
		return cleanupAndReturn(fmt.Errorf("get physical core failed: %w", err))
	}

	errGroups := make(map[string]*openErrGroup)
	addErr := func(evtName string, err error) {
		reason := err.Error()
		key := evtName + "|" + reason

		g, ok := errGroups[key]
		if !ok {
			g = &openErrGroup{
				event:  evtName,
				reason: reason,
			}
			errGroups[key] = g
		}
		g.count++
	}

	closeErrGroups := make(map[string]int)
	for cpu := 0; cpu < cpuNum; cpu++ {
		for _, tid := range tids {
			for _, evtName := range pmuInputAttr.EvtList {
				fd, err := beginSampling(evtName, pmuInputAttr, tid, cpu)
				if err != nil || fd == -1 {
					if err == nil {
						err = fmt.Errorf("beginSampling returned invalid fd")
					}
					addErr(evtName, err)
					continue
				}

				buf, err := initMmap(fd, pmuInputAttr.EnableBRBE)
				if err != nil {
					addErr(evtName, fmt.Errorf("initMmap failed: %w", err))

					if closeErr := syscall.Close(fd); closeErr != nil {
						closeErrGroups[closeErr.Error()]++
					}
					continue
				}

				meta := (*perfEventMmapPage)(unsafe.Pointer(&buf[0]))
				pl.events[fd] = &pmuEvent{
					fd:   fd,
					buf:  buf,
					meta: meta,
					data: buf[meta.dataOffset : meta.dataOffset+meta.dataSize],
					name: fmt.Sprintf("%s-tid-%d", evtName, tid),
				}
			}
		}
	}
	if len(pl.events) > 0 {
		return pd, nil
	}

	if len(errGroups) == 0 {
		return cleanupAndReturn(fmt.Errorf("no event opened"))
	}

	mainErr := summarizeOpenErrors(errGroups)
	if len(closeErrGroups) > 0 {
		closeMsgs := make([]string, 0, len(closeErrGroups))
		for reason, count := range closeErrGroups {
			closeMsgs = append(closeMsgs,
				fmt.Sprintf("cleanup close failed %d times: %s", count, reason),
			)
		}
		mainErr = fmt.Errorf("%w; %s", mainErr, strings.Join(closeMsgs, "; "))
	}
	return cleanupAndReturn(mainErr)
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

func readSamples(evt *pmuEvent, sampleType uint64, handler func(sample)) error {
	head := atomic.LoadUint64(&evt.meta.dataHead)
	tail := atomic.LoadUint64(&evt.meta.dataTail)

	for tail < head {
		offset := tail % uint64(len(evt.data))
		if offset+uint64(unsafe.Sizeof(perfEventHeader{})) > uint64(len(evt.data)) {
			break  // incomplete header
		}

		header := (*perfEventHeader)(unsafe.Pointer(&evt.data[offset]))
		headerSize := uint64(unsafe.Sizeof(*header))

		if header.size < uint16(headerSize) || offset+uint64(header.size) > uint64(len(evt.data)) {
			break  // invalid header size
		}

		switch header.typ {
		case perfRecordSample:
			payload := evt.data[offset+headerSize : offset+uint64(header.size)]
			s := parseSample(payload, sampleType, evt.name)
			handler(s)
		default:
			// do nothing
		}

		tail += uint64(header.size)
	}

	atomic.StoreUint64(&evt.meta.dataTail, head)
	return nil
}

// PmuRead collects samples from all events under a pd.
// Returns all gathered samples or an error if reading fails.
func pmuRead(pd int) ([]sample, error) {
	pl, ok := pdMap[pd]
	if !ok {
		return nil, fmt.Errorf("PmuRead failed. Invalid pd %d", pd)
	}

	sampleType := getSampleType(pl.enableBRBE)

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
	pl, ok := pdMap[pd]
	if !ok {
		return fmt.Errorf("PmuReset failed. Invalid pd: %d", pd)
	}

	for _, evt := range pl.events {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(evt.fd),
			perfEventIocReset,
			0)
		if errno != 0 {
			return errno
		}
	}
	return nil
}

// PmuEnable enables all events under a given pd.
// Returns an error if the ioctl enable call fails.
func pmuEnable(pd int) error {
	pl, ok := pdMap[pd]
	if !ok {
		return fmt.Errorf("PmuEnable failed. Invalid pd: %d", pd)
	}

	for _, evt := range pl.events {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(evt.fd),
			perfEventIocEnable,
			0)
		if errno != 0 {
			return errno
		}
	}
	return nil
}

// PmuDisable disables all events under a given pd.
// Returns an error if the ioctl disable call fails.
func pmuDisable(pd int) error {
	pl, ok := pdMap[pd]
	if !ok {
		return fmt.Errorf("PmuDisable failed. Invalid pd: %d", pd)
	}

	for _, evt := range pl.events {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(evt.fd),
			perfEventIocDisable,
			0)
		if errno != 0 {
			return errno
		}
	}
	return nil
}

// pmuClose closes all perf event file descriptors and unmaps their buffers for a given pd.
// It attempts to clean up all events even if some cleanup operations fail.
func pmuClose(pd int) error {
	pl, ok := pdMap[pd]
	if !ok {
		return fmt.Errorf("PmuClose failed. Invalid pd: %d", pd)
	}

	var errs []error
	for fd, evt := range pl.events {
		if len(evt.buf) > 0 {
			if err := syscall.Munmap(evt.buf); err != nil {
				errs = append(errs, fmt.Errorf("munmap failed for %s(fd=%d): %w", evt.name, fd, err))
			}
		}

		if err := syscall.Close(fd); err != nil {
			errs = append(errs, fmt.Errorf("close fd failed for %s(fd=%d): %w", evt.name, fd, err))
		}
	}

	delete(pdMap, pd)
	return errors.Join(errs...)
}
