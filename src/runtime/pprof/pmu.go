// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// To use the PMU collection, add equivalent profiling support to a standalone 
// program, add code like the following to your main function:
//	var profile = flag.String("profile", "", "write profile to `file`")

//	func main() {
//	    flag.Parse()
//	    if *profile != "" {
//	        f, err := os.Create(*cpuprofile)
//	        if err != nil {
//	            log.Fatal("could not create CPU profile: ", err)
//	        }
//	        defer f.Close() // error handling omitted for example
//          pmuAttr := new(PMUAttr)
//          pmuAttr.EvtList = []string{"cycles"}
//          pmuAttr.EnableBRBE = true
//          pmuAttr.Duration = 20 // unit: s
//          pmuAttr.Period = 1000
//	        if err := pprof.StartPMUProfile(f, pmuAttr); err != nil {
//	            log.Fatal("PMU collection failed: ", err)
//	        }
//	    }
//
//	    // ... rest of the program ...
//	}
//
// The standard HTTP interface to collect PMU data like:
//
// import _ "net/http/pprof"
// ...
//	go func() {
//		log.Println(http.ListenAndServe("localhost:6060", nil))
//	}()
//
// To collect a 10-second execution to get the BRBE prof file
// curl -o brbe.prof "http://localhost:6060/debug/pprof/profile?event=cycles&freq=1000&brbe=true&seconds=10"

package pprof

import (
	"fmt"
	"time"
	"io"
)

// StartPMUProfile enables PMU collection for profiling of the current
// process. After collection, the profile will be written to w.
// StartPMUProfile returns an error when input parameter is invalid
// or PMU syscall execution failed.
//
// Please check whether your environment supports the collections of BRBE.
func StartPMUProfile(w io.Writer, pmuAttr *PMUAttr) (<-chan error, error) {
	if pmuAttr == nil {
		return nil, fmt.Errorf("PMUAttr is nil")
	}

	pd, err := pmuOpen(pmuAttr)
	if err != nil {
		return nil, err
	}

	errCh := make(chan error, 1)

	go func() {
		var retErr error
		enabled := false

		disable := func() error {
			if !enabled {
				return nil
			}
			err := pmuDisable(pd)
			enabled = false
			return err
		}

		defer func() {
			if r := recover(); r != nil {
				retErr = fmt.Errorf("PMU profile panic: %v", r)
			}

			if err := disable(); err != nil && retErr == nil {
				retErr = fmt.Errorf("Disable PMU failed: %w", err)
			}

			pmuClose(pd)

			errCh <- retErr
			close(errCh)
		}()

		if err := pmuReset(pd); err != nil {
			retErr = fmt.Errorf("Reset PMU failed: %w", err)
			return
		}

		if err := pmuEnable(pd); err != nil {
			retErr = fmt.Errorf("Enable PMU failed: %w", err)
			return
		}
		enabled = true

		allSamples := []sample{}
		for i := int64(0); i < pmuAttr.Duration; i++ {
			time.Sleep(time.Second)

			samples, err := pmuRead(pd)
			if err != nil {
				retErr = fmt.Errorf("Read PMU failed: %w", err)
				return
			}
			if len(samples) > 0 {
				allSamples = append(allSamples, samples...)
			}
		}

		if err := disable(); err != nil {
			retErr = fmt.Errorf("Disable PMU failed: %w", err)
			return
		}

		if err := writeProf(w, allSamples, pmuAttr.Duration); err != nil {
			retErr = fmt.Errorf("Write prof file failed: %w", err)
			return
		}
	}()

	return errCh, nil
}