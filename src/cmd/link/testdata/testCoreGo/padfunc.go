package main

import (
	"fmt"
	"reflect"
	"runtime"
)

//go:noinline
func zyes() bool {
	return true
}

//go:noinline
func zno() bool {
	return false
}

func main() {
	padfunccheck()
}

const PADMINFUNC = 32 // padding the minimum function size to 32 bytes

func padfunccheck() {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		fmt.Print("SKIP\n")
		return
	}
	var zyesptr uintptr = reflect.ValueOf(zyes).Pointer()
	var znoptr uintptr = reflect.ValueOf(zno).Pointer()
	fmt.Printf("zyes:0x%x zno:0x%x\n", zyesptr, znoptr)
	if (znoptr - zyesptr) != PADMINFUNC {
		panic("FAIL")
	}
	fmt.Print("OK\n")
}
