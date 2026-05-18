// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pprof

import (
	"os"
	"strings"
	"strconv"
	"path/filepath"
)

var coreEventList []string
var pmuDevice     string

const sysDevicePath = "/sys/bus/event_source/devices/"
const midrEl1Path = "/sys/devices/system/cpu/cpu0/regs/identification/midr_el1";

type chipType int
const (
	undefinedType chipType = iota    // 0
	hipA                    // 1
	hipB                    // 2
	hipC                    // 3
	hipF                    // 4
	hipE                    // 5
	hipG                    // 6
)

var chipMap = map[string]chipType{
	"0x00000000481fd010": hipA,
	"0x00000000480fd020": hipB,
	"0x00000000480fd030": hipC,
	"0x00000000480fd220": hipF,
	"0x00000000480fd450": hipE,
	"0x00000000480fd060": hipG,
}

// GetCpuType reads the midr_el1 file and looks up the CPU type in the map
// Returns the corresponding type number, or -1 with an error if not found
func getCpuType() chipType {
	data, err := os.ReadFile(midrEl1Path)
	if err != nil {
		return undefinedType
	}
	midr := strings.TrimSpace(string(data))
	if val, ok := chipMap[midr]; ok {
		return val
	} else {
		return undefinedType
	}
}

// SampleBRBE checks the current CPU type.
// It returns true if the CPU is identified as hipF or hipG, otherwise false.
func SampleBRBE() bool {
	var cpuType = getCpuType()
	if cpuType == hipF || cpuType == hipG {
		return true
	}
	return false
}

// SamplePMU checks the current CPU type.
// It returns true if the CPU is identified as PMU collection supported hip, otherwise false.
func SamplePMU() bool {
	var cpuType = getCpuType()
	if cpuType == undefinedType {
		return false
	}
	return true
}

func getPmuDevicePath() string {
	if pmuDevice != "" {
		return pmuDevice
	}

	entries, err := os.ReadDir(sysDevicePath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." || name == "cpu" {
			continue
		}

		armPmuPath := filepath.Join(sysDevicePath, name, "cpus")
		if _, err := os.Stat(armPmuPath); err == nil {
			pmuDevice = filepath.Join(sysDevicePath, name)
			break
		}
	}

	return pmuDevice
}

func queryCoreEvent() []string {
	if len(coreEventList) > 0 {
		return coreEventList
	}

	for evt := range coreEventMap {
		coreEventList = append(coreEventList, evt)
	}

	pmuDevPath := getPmuDevicePath()
	if pmuDevPath == "" {
		return coreEventList
	}

	eventsPath := filepath.Join(pmuDevPath, "events")
	entries, err := os.ReadDir(eventsPath)
	if err != nil {
		return coreEventList
	}

	for _, entry := range entries {
		if entry.Type().IsRegular() {
			coreEventList = append(coreEventList, entry.Name())
		}
	}

	return coreEventList
}

func containsCoreEvent(events []string, name string) bool {
	for _, e := range events {
		if e == name {
			return true
		}
	}
	return false
}

func getKernelCoreEventConfig(name string) int64 {
	pmuDevicePath := getPmuDevicePath()
	if pmuDevicePath == "" {
		return -1
	}

	eventPath := filepath.Join(pmuDevicePath, "events", name)
	realPath := getRealPath(eventPath)
	if !isValidPath(realPath) {
		return -1
	}

	data, err := os.ReadFile(realPath)
	if err != nil {
		return -1
	}

	configStr := strings.TrimSpace(string(data))
	idx := strings.Index(configStr, "=")
	if idx == -1 {
		return -1
	}

	subStr := configStr[idx+1:]
	val, err := strconv.ParseInt(subStr, 0, 64)
	if err != nil {
		return -1
	}
	return val
}

func isValidPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getRealPath(path string) string {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return ""
	}
	return real
}

func getKernelCoreEventType() int64 {
	pmuDevicePath := getPmuDevicePath()
	if pmuDevicePath == "" {
		return -1
	}

	eventPath := filepath.Join(pmuDevicePath, "type")
	realPath := getRealPath(eventPath)
	if !isValidPath(realPath) {
		return -1
	}

	data, err := os.ReadFile(realPath)
	if err != nil {
		return -1
	}

	typeStr := strings.TrimSpace(string(data))
	val, err := strconv.ParseInt(typeStr, 0, 64)
	if err != nil {
		return -1
	}
	return val
}

func getCoreEvent(evtName string) (evtConfig, evtType int64) {
	coreEvt, ok := coreEventMap[evtName]
	if ok {
		return coreEvt.config, coreEvt.typ
	}

	return getKernelCoreEventConfig(evtName), getKernelCoreEventType()
}
