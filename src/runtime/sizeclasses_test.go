// Copyright (c) Huawei Technologies Co., Ltd. 2025-2025. All rights reserved.

package runtime_test

import (
	"internal/goexperiment"
	"internal/testenv"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func getSizeClassMaxValue[T ~uint8 | ~uint16](slice []T) T {
	if len(slice) == 0 {
		panic("empty slice")
	}
	maxVal := slice[0]
	for _, v := range slice {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func TestSizeClassArrayMaxValue(t *testing.T) {
	// This testcase is used to check the maximum values of some arrays in runtime/sizeclasses.go and runtime/sizeclasses_expanded_by_eight.go.
	// The maximum values are used in cmd/compile/internal/ssa/prove.go^getGlobalConstArrayUpperLimit.
	//
	// If you are sure to change these values, you need also change these values in getGlobalConstArrayUpperLimit.

	if getSizeClassMaxValue(runtime.Class_to_size[:]) != 32768 {
		t.Errorf("expect the max value in array class_to_size is 32768, see runtime/sizeclasses.go and runtime/sizeclasses_expanded_span_size.go, limited by cmd/compile/internal/ssa/prove.go")
	}
	if !goexperiment.PageShift14 && getSizeClassMaxValue(runtime.Size_to_class8[:]) != 32 {
		t.Errorf("when pagesize is 8k, expect the max value in array Size_to_class8 is 32, see runtime/sizeclasses.go and runtime/sizeclasses_expanded_span_size.go, limited by cmd/compile/internal/ssa/prove.go")
	} else if goexperiment.PageShift14 && getSizeClassMaxValue(runtime.Size_to_class8[:]) != 34 {
		t.Errorf("when pagesize is 16k, expect the max value in array Size_to_class8 is 34, see runtime/sizeclasses_14.go, limited by cmd/compile/internal/ssa/prove.go")
	}
	if !goexperiment.PageShift14 && getSizeClassMaxValue(runtime.Size_to_class128[:]) != 67 {
		t.Errorf("when pagesize is 8k, expect the max value in array Size_to_class128 is 67, see runtime/sizeclasses.go and runtime/sizeclasses_expanded_span_size.go, limited by cmd/compile/internal/ssa/prove.go")
	} else if goexperiment.PageShift14 && getSizeClassMaxValue(runtime.Size_to_class128[:]) != 70 {
		t.Errorf("when pagesize is 16k, expect the max value in array Size_to_class128 is 70, see runtime/sizeclasses_14.go, limited by cmd/compile/internal/ssa/prove.go")
	}
}

func runTestWithOptions(t *testing.T, test string, options ...string) {
	testenv.MustHaveExec(t)
	gotool := testenv.GoToolPath(t)

	arg := []string{"test", "-run=" + test}
	arg = append(arg, options...)
	cmd := exec.Command(gotool, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'go test failed: (%v) %s", err, output)
	}

	content := string(output)
	want := "PASS"
	if !strings.Contains(content, want) {
		t.Errorf("%s with %s: want %s, got %v", test, options, want, content)
	}
}

func TestSizeClassArrayMaxValueWithTags(t *testing.T) {
	runTestWithOptions(t, "TestSizeClassArrayMaxValue$")
}
