// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pprof

import (
	"fmt"
	"syscall"
)

// Return error info of perf_event_open syscall
func errnoErr(e syscall.Errno) error {
	switch e {
	case syscall.EPERM, syscall.EACCES:
		return fmt.Errorf("No permission")
	case syscall.EBUSY:
		return fmt.Errorf("Pmu device is busy")
	case syscall.EINVAL:
		return fmt.Errorf("Invalid event for pmu device")
	case syscall.ESRCH:
		return fmt.Errorf("No such process")
	case syscall.EMFILE:
		return fmt.Errorf("Too many open files")
	case syscall.ENOENT:
		return fmt.Errorf("Invalid event")	
	case syscall.ENOTSUP:
		return fmt.Errorf("Operation not supported")
	default:
		return fmt.Errorf("Unknown error: %v", e)
	}
}
