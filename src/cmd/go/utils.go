// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

var coreList = []struct {
	implementer int64
	variant     int64
	part        int64
	name        string
}{
	{0x41, -1, 0xd02, "cortex-a34"},
	{0x41, -1, 0xd04, "cortex-a35"},
	{0x41, -1, 0xd03, "cortex-a53"},
	{0x41, -1, 0xd07, "cortex-a57"},
	{0x41, -1, 0xd08, "cortex-a72"},
	{0x41, -1, 0xd09, "cortex-a73"},
	{0x43, -1, 0x0a0, "thunderx"},
	{0x43, 0x0, 0x0a1, "thunderxt88p1"},
	{0x43, -1, 0x0a1, "thunderxt88"},
	{0x43, -1, 0x0a0, "octeontx"},
	{0x43, -1, 0x0a2, "octeontx81"},
	{0x43, -1, 0x0a3, "octeontx83"},
	{0x43, -1, 0x0a2, "thunderxt81"},
	{0x43, -1, 0x0a3, "thunderxt83"},
	{0xc0, -1, 0xac3, "ampere1"},
	{0xc0, -1, 0xac4, "ampere1a"},
	{0x50, 0x3, 0x000, "emag"},
	{0x50, -1, 0x000, "xgene1"},
	{0x51, -1, 0xc00, "falkor"},
	{0x51, -1, 0xc00, "qdf24xx"},
	{0x53, -1, 0x001, "exynos-m1"},
	{0x68, -1, 0x000, "phecda"},
	{0x42, -1, 0x516, "thunderx2t99p1"},
	{0x42, -1, 0x516, "vulcan"},
	{0x43, -1, 0x0af, "thunderx2t99"},
	{0x41, -1, 0xd05, "cortex-a55"},
	{0x41, -1, 0xd0a, "cortex-a75"},
	{0x41, -1, 0xd0b, "cortex-a76"},
	{0x41, -1, 0xd0e, "cortex-a76ae"},
	{0x41, -1, 0xd0d, "cortex-a77"},
	{0x41, -1, 0xd41, "cortex-a78"},
	{0x41, -1, 0xd42, "cortex-a78ae"},
	{0x41, -1, 0xd4b, "cortex-a78c"},
	{0x41, -1, 0xd06, "cortex-a65"},
	{0x41, -1, 0xd43, "cortex-a65ae"},
	{0x41, -1, 0xd44, "cortex-x1"},
	{0x41, -1, 0xd0c, "ares"},
	{0x41, -1, 0xd0c, "neoverse-n1"},
	{0x41, -1, 0xd4a, "neoverse-e1"},
	{0x43, -1, 0x0b0, "octeontx2"},
	{0x43, -1, 0x0b1, "octeontx2t98"},
	{0x43, -1, 0x0b2, "octeontx2t96"},
	{0x43, -1, 0x0b2, "octeontx2t93"},
	{0x43, -1, 0x0b3, "octeontx2f95"},
	{0x43, -1, 0x0b4, "octeontx2f95n"},
	{0x43, -1, 0x0b5, "octeontx2f95mm"},
	{0x46, -1, 0x001, "a64fx"},
	{0x48, -1, 0xd01, "tsv110"},
	{0x48, -1, 0xd02, "hip09"},
	{0x48, -1, 0xd45, "hip10c"},
	{0x48, -1, 0xd22, "hip11"},
	{0x43, 0xa, 0x0b8, "thunderx3t110"},
	{0x41, -1, 0xd40, "zeus"},
	{0x41, -1, 0xd40, "neoverse-v1"},
	{0xff, -1, 0xffffffff, "neoverse-512tvb"},
	{0x51, -1, 0xc01, "saphira"},
	{0x41, -1, 0xd07d03, "cortex-a57.cortex-a53"},
	{0x41, -1, 0xd08d03, "cortex-a72.cortex-a53"},
	{0x41, -1, 0xd09d04, "cortex-a73.cortex-a35"},
	{0x41, -1, 0xd09d03, "cortex-a73.cortex-a53"},
	{0x41, -1, 0xd0ad05, "cortex-a75.cortex-a55"},
	{0x41, -1, 0xd0bd05, "cortex-a76.cortex-a55"},
	{0x41, -1, 0xd15, "cortex-r82"},
	{0x41, -1, 0xd46, "cortex-a510"},
	{0x41, -1, 0xd47, "cortex-a710"},
	{0x41, -1, 0xd48, "cortex-x2"},
	{0x41, -1, 0xd49, "neoverse-n2"},
	{0x41, -1, 0xd4f, "demeter"},
	{0x41, -1, 0xd4f, "neoverse-v2"},
	{0x48, -1, 0xd03, "hip10a"},
	{0x48, -1, 0xd06, "hip12"},
}

func getSha256(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)
	return hex.EncodeToString(hashBytes)
}

func readFloatFromFile(file *os.File) float32 {
	var hexFloat [4]byte
	for i := 0; i < 4; i++ {
		_, err := fmt.Fscanf(file, "%2x", &hexFloat[i])
		if err != nil {
			return 0
		}
	}
	uint32Float := binary.LittleEndian.Uint32(hexFloat[:])
	xFloat := math.Float32frombits(uint32Float)
	return xFloat
}

func readByteFromFile(file *os.File) byte {
	var xByte byte
	_, err := fmt.Fscanf(file, "%2x", &xByte)
	if err != nil {
		return 0
	}
	return xByte
}

func parseValue(data string) int64 {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) > 1 {
		value, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 0, 64)
		return value
	}
	return 0
}

func parseInfo(file *os.File) (int64, int64, int64) {
	scanner := bufio.NewScanner(file)
	implementer := int64(-1)
	variant := int64(-1)
	part := int64(-1)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.Contains(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "processor") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				processor, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				if processor != 0 {
					break
				}
			}
		} else if strings.HasPrefix(line, "CPU implementer") {
			implementer = parseValue(line)
		} else if strings.HasPrefix(line, "CPU variant") {
			variant = parseValue(line)
		} else if strings.HasPrefix(line, "CPU part") {
			part = parseValue(line)
		}
	}
	return implementer, variant, part
}

func getCPUInfo() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "native"
	}
	defer file.Close()
	implementer, variant, part := parseInfo(file)
	for _, core := range coreList {
		if core.implementer == implementer && (core.variant == -1 || core.variant == variant) && core.part == part {
			return core.name
		}
	}
	return "native"
}
