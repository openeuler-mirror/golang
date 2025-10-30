// buildrun -gcflags="all=-aggressiveprove"

// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	// Get the assembly for runtime.mallocgc of the current executable
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("go", "tool", "objdump", "-s", "^runtime\\.mallocgc$", exePath)
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	// Scan each line of the objdump output for CALL to runtime.panic*
	callRegex := regexp.MustCompile(`\bCALL\s+runtime\.panic[^\s]*`)
	count := 0
	for _, line := range strings.Split(string(output), "\n") {
		if callRegex.MatchString(line) {
			count++
		}
	}
	fmt.Printf("Number of calls to runtime.panic* in runtime.mallocgc: %d\n", count)
	if count != 0 {
		panic("expected no panic")
	}
}
