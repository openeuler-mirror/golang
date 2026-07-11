// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cmd/internal/hash"
	"flag"
	"fmt"
	"internal/platform"
	"internal/testenv"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestMain executes the test binary as the objdump command if
// GO_OBJDUMPTEST_IS_OBJDUMP is set, and runs the test otherwise.
func TestMain(m *testing.M) {
	if os.Getenv("GO_OBJDUMPTEST_IS_OBJDUMP") != "" {
		main()
		os.Exit(0)
	}

	os.Setenv("GO_OBJDUMPTEST_IS_OBJDUMP", "1")
	os.Exit(m.Run())
}

var x86Need = []string{ // for both 386 and AMD64
	"JMP main.main(SB)",
	"CALL main.Println(SB)",
	"RET",
}

var amd64GnuNeed = []string{
	"jmp",
	"callq",
	"cmpb",
}

var i386GnuNeed = []string{
	"jmp",
	"call",
	"cmp",
}

var armNeed = []string{
	"B main.main(SB)",
	"BL main.Println(SB)",
	"RET",
}

var arm64Need = []string{
	"JMP main.main(SB)",
	"CALL main.Println(SB)",
	"RET",
}

var armGnuNeed = []string{ // for both ARM and AMR64
	"ldr",
	"bl",
	"cmp",
}

var loong64Need = []string{
	"JMP main.main(SB)",
	"CALL main.Println(SB)",
	"RET",
}

var loong64GnuNeed = []string{
	"ld.b",
	"bl",
	"beq",
}

var ppcNeed = []string{
	"BR main.main(SB)",
	"CALL main.Println(SB)",
	"RET",
}

var ppcPIENeed = []string{
	"BR",
	"CALL",
	"RET",
}

var ppcGnuNeed = []string{
	"mflr",
	"lbz",
	"beq",
}

var s390xGnuNeed = []string{
	"brasl",
	"j",
	"clije",
}

func mustHaveDisasm(t *testing.T) {
	switch runtime.GOARCH {
	case "mips", "mipsle", "mips64", "mips64le":
		t.Skipf("skipping on %s, issue 12559", runtime.GOARCH)
	}
}

var target = flag.String("target", "", "test disassembly of `goos/goarch` binary")

// objdump is fully cross platform: it can handle binaries
// from any known operating system and architecture.
// We could in principle add binaries to testdata and check
// all the supported systems during this test. However, the
// binaries would be about 1 MB each, and we don't want to
// add that much junk to the hg repository. Instead, build a
// binary for the current system (only) and test that objdump
// can handle that one.

func testDisasm(t *testing.T, srcfname string, printCode bool, printGnuAsm bool, flags ...string) {
	mustHaveDisasm(t)
	goarch := runtime.GOARCH
	if *target != "" {
		f := strings.Split(*target, "/")
		if len(f) != 2 {
			t.Fatalf("-target argument must be goos/goarch")
		}
		defer os.Setenv("GOOS", os.Getenv("GOOS"))
		defer os.Setenv("GOARCH", os.Getenv("GOARCH"))
		os.Setenv("GOOS", f[0])
		os.Setenv("GOARCH", f[1])
		goarch = f[1]
	}

	hash := hash.Sum16([]byte(fmt.Sprintf("%v-%v-%v-%v", srcfname, flags, printCode, printGnuAsm)))
	tmp := t.TempDir()
	hello := filepath.Join(tmp, fmt.Sprintf("hello-%x.exe", hash))
	args := []string{"build", "-o", hello}
	args = append(args, flags...)
	args = append(args, srcfname)
	cmd := testenv.Command(t, testenv.GoToolPath(t), args...)
	// "Bad line" bug #36683 is sensitive to being run in the source directory.
	cmd.Dir = "testdata"
	t.Logf("Running %v", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build %s: %v\n%s", srcfname, err, out)
	}
	need := []string{
		"TEXT main.main(SB)",
	}

	if printCode {
		need = append(need, `	Println("hello, world")`)
	} else {
		need = append(need, srcfname+":6")
	}

	switch goarch {
	case "amd64", "386":
		need = append(need, x86Need...)
	case "arm":
		need = append(need, armNeed...)
	case "arm64":
		need = append(need, arm64Need...)
	case "loong64":
		need = append(need, loong64Need...)
	case "ppc64", "ppc64le":
		var pie bool
		for _, flag := range flags {
			if flag == "-buildmode=pie" {
				pie = true
				break
			}
		}
		if pie {
			// In PPC64 PIE binaries we use a "local entry point" which is
			// function symbol address + 8. Currently we don't symbolize that.
			// Expect a different output.
			need = append(need, ppcPIENeed...)
		} else {
			need = append(need, ppcNeed...)
		}
	}

	if printGnuAsm {
		switch goarch {
		case "amd64":
			need = append(need, amd64GnuNeed...)
		case "386":
			need = append(need, i386GnuNeed...)
		case "arm", "arm64":
			need = append(need, armGnuNeed...)
		case "loong64":
			need = append(need, loong64GnuNeed...)
		case "ppc64", "ppc64le":
			need = append(need, ppcGnuNeed...)
		case "s390x":
			need = append(need, s390xGnuNeed...)
		}
	}
	args = []string{
		"-s", "main.main",
		hello,
	}

	if printCode {
		args = append([]string{"-S"}, args...)
	}

	if printGnuAsm {
		args = append([]string{"-gnu"}, args...)
	}
	cmd = testenv.Command(t, testenv.Executable(t), args...)
	cmd.Dir = "testdata" // "Bad line" bug #36683 is sensitive to being run in the source directory
	out, err = cmd.CombinedOutput()
	t.Logf("Running %v", cmd.Args)

	if err != nil {
		exename := srcfname[:len(srcfname)-len(filepath.Ext(srcfname))] + ".exe"
		t.Fatalf("objdump %q: %v\n%s", exename, err, out)
	}

	text := string(out)
	ok := true
	for _, s := range need {
		if !strings.Contains(text, s) {
			t.Errorf("disassembly missing '%s'", s)
			ok = false
		}
	}
	if goarch == "386" {
		if strings.Contains(text, "(IP)") {
			t.Errorf("disassembly contains PC-Relative addressing on 386")
			ok = false
		}
	}

	if !ok || testing.Verbose() {
		t.Logf("full disassembly:\n%s", text)
	}
}

func testGoAndCgoDisasm(t *testing.T, printCode bool, printGnuAsm bool) {
	t.Parallel()
	testDisasm(t, "fmthello.go", printCode, printGnuAsm)
	if testenv.HasCGO() {
		testDisasm(t, "fmthellocgo.go", printCode, printGnuAsm)
	}
}

func TestDisasm(t *testing.T) {
	testGoAndCgoDisasm(t, false, false)
}

func TestDisasmCode(t *testing.T) {
	testGoAndCgoDisasm(t, true, false)
}

func TestDisasmGnuAsm(t *testing.T) {
	testGoAndCgoDisasm(t, false, true)
}

func TestDisasmExtld(t *testing.T) {
	testenv.MustHaveCGO(t)
	switch runtime.GOOS {
	case "plan9":
		t.Skipf("skipping on %s", runtime.GOOS)
	}
	t.Parallel()
	testDisasm(t, "fmthello.go", false, false, "-ldflags=-linkmode=external")
}

func TestDisasmPIE(t *testing.T) {
	if !platform.BuildModeSupported("gc", "pie", runtime.GOOS, runtime.GOARCH) {
		t.Skipf("skipping on %s/%s, PIE buildmode not supported", runtime.GOOS, runtime.GOARCH)
	}
	if !platform.InternalLinkPIESupported(runtime.GOOS, runtime.GOARCH) {
		// require cgo on platforms that PIE needs external linking
		testenv.MustHaveCGO(t)
	}
	t.Parallel()
	testDisasm(t, "fmthello.go", false, false, "-buildmode=pie")
}

func TestDisasmGoobj(t *testing.T) {
	mustHaveDisasm(t)
	testenv.MustHaveGoBuild(t)

	tmp := t.TempDir()

	importcfgfile := filepath.Join(tmp, "hello.importcfg")
	testenv.WriteImportcfg(t, importcfgfile, nil, "testdata/fmthello.go")

	hello := filepath.Join(tmp, "hello.o")
	args := []string{"tool", "compile", "-p=main", "-importcfg=" + importcfgfile, "-o", hello}
	args = append(args, "testdata/fmthello.go")
	out, err := testenv.Command(t, testenv.GoToolPath(t), args...).CombinedOutput()
	if err != nil {
		t.Fatalf("go tool compile fmthello.go: %v\n%s", err, out)
	}
	need := []string{
		"main(SB)",
		"fmthello.go:6",
	}

	args = []string{
		"-s", "main",
		hello,
	}

	out, err = testenv.Command(t, testenv.Executable(t), args...).CombinedOutput()
	if err != nil {
		t.Fatalf("objdump fmthello.o: %v\n%s", err, out)
	}

	text := string(out)
	ok := true
	for _, s := range need {
		if !strings.Contains(text, s) {
			t.Errorf("disassembly missing '%s'", s)
			ok = false
		}
	}
	if runtime.GOARCH == "386" {
		if strings.Contains(text, "(IP)") {
			t.Errorf("disassembly contains PC-Relative addressing on 386")
			ok = false
		}
	}
	if !ok {
		t.Logf("full disassembly:\n%s", text)
	}
}

func TestGoobjFileNumber(t *testing.T) {
	// Test that file table in Go object file is parsed correctly.
	testenv.MustHaveGoBuild(t)
	mustHaveDisasm(t)

	t.Parallel()

	tmp := t.TempDir()

	obj := filepath.Join(tmp, "p.a")
	cmd := testenv.Command(t, testenv.GoToolPath(t), "build", "-o", obj)
	cmd.Dir = filepath.Join("testdata/testfilenum")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	cmd = testenv.Command(t, testenv.Executable(t), obj)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("objdump failed: %v\n%s", err, out)
	}

	text := string(out)
	for _, s := range []string{"a.go", "b.go", "c.go"} {
		if !strings.Contains(text, s) {
			t.Errorf("output missing '%s'", s)
		}
	}

	if t.Failed() {
		t.Logf("output:\n%s", text)
	}
}

func TestGoObjOtherVersion(t *testing.T) {
	t.Parallel()

	obj := filepath.Join("testdata", "go116.o")
	cmd := testenv.Command(t, testenv.Executable(t), obj)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("objdump go116.o succeeded unexpectedly")
	}
	if !strings.Contains(string(out), "go object of a different version") {
		t.Errorf("unexpected error message:\n%s", out)
	}
}

func canTestModuledata(t *testing.T) bool {
	switch runtime.GOOS {
	case "darwin", "windows", "plan9":
		t.Skipf("moduledata support is ELF-only, skipping on %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "mips", "mipsle", "mips64", "mips64le":
		t.Skipf("skipping on %s", runtime.GOARCH)
	}
	return true
}

func buildTestBinary(t *testing.T) string {
	tmp := t.TempDir()
	hello := filepath.Join(tmp, "hello")
	args := []string{"build", "-o", hello}
	args = append(args, filepath.Join("testdata", "fmthello.go"))
	cmd := testenv.Command(t, testenv.GoToolPath(t), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build fmthello.go: %v\n%s", err, out)
	}
	return hello
}

func runObjdump(t *testing.T, hello string, flags []string) ([]byte, error) {
	cmd := testenv.Command(t, testenv.Executable(t), flags...)
	cmd.Args = append(cmd.Args, hello)
	out, err := cmd.CombinedOutput()
	t.Logf("Running %v", cmd.Args)
	return out, err
}

func testModuledata(t *testing.T, flags []string, need []string, dontNeed []string) {
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, flags)
	if err != nil {
		t.Fatalf("objdump %v: %v\n%s", flags, err, out)
	}

	text := string(out)
	for _, s := range need {
		if !strings.Contains(text, s) {
			t.Errorf("objdump %v: missing '%s' in output", flags, s)
		}
	}
	for _, s := range dontNeed {
		if strings.Contains(text, s) {
			t.Errorf("objdump %v: unexpected '%s' in output", flags, s)
		}
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataFlags(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	tests := []struct {
		name     string
		flags    []string
		need     []string
		dontNeed []string
	}{
		{"m", []string{"-m"}, []string{"moduledata:"}, nil},
		{"p", []string{"-p"}, []string{"pcHeader:"}, nil},
		{"mp", []string{"-m", "-p"}, []string{"moduledata:", "pcHeader:"}, nil},
		{"I", []string{"-I"}, []string{"itablinks:"}, nil},
		{"T", []string{"-T"}, []string{"typelinks:"}, nil},
		{"mit", []string{"-m", "-I", "-T"}, []string{"moduledata:", "itablinks:", "typelinks:"}, nil},
		{"mIPAll", []string{"-m", "-p", "-I", "-T"}, []string{"moduledata:", "pcHeader:", "itablinks:", "typelinks:"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runObjdump(t, hello, tt.flags)
			if err != nil {
				t.Fatalf("objdump %v: %v\n%s", tt.flags, err, out)
			}

			text := string(out)
			for _, s := range tt.need {
				if !strings.Contains(text, s) {
					t.Errorf("objdump %v: missing '%s' in output", tt.flags, s)
				}
			}
			if tt.dontNeed != nil {
				for _, s := range tt.dontNeed {
					if strings.Contains(text, s) {
						t.Errorf("objdump %v: unexpected '%s' in output", tt.flags, s)
					}
				}
			}

			if t.Failed() {
				t.Logf("full output:\n%s", text)
			}
		})
	}
}

func TestModuledataWithFilter(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m", "-s", "main.main"})
	if err != nil {
		t.Fatalf("objdump -m -s main.main: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "main.main") {
		t.Errorf("objdump -m -s main.main: missing 'main.main' in output")
	}
	if !strings.Contains(text, "ftabOff:") {
		t.Errorf("objdump -m -s main.main: missing 'ftabOff:' in output")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataWithFuncData(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m", "-s", "main.main", "-F"})
	if err != nil {
		t.Fatalf("objdump -m -s main.main -F: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "PCSP:") || !strings.Contains(text, "PCDATA_") || !strings.Contains(text, "FUNCDATA_") {
		t.Errorf("objdump -m -s main.main -F: missing funcdata in output")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataWithoutFuncData(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m", "-s", "main.main"})
	if err != nil {
		t.Fatalf("objdump -m -s main.main: %v\n%s", err, out)
	}

	text := string(out)
	if strings.Contains(text, "PCSP:") || strings.Contains(text, "PCDATA_") || strings.Contains(text, "FUNCDATA_") {
		t.Errorf("objdump -m -s main.main: unexpected funcdata in output (should only appear with -F)")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataOnlyM(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-M"})
	if err != nil {
		t.Fatalf("objdump -M: %v\n%s", err, out)
	}

	text := string(out)
	for _, s := range []string{"moduledata:", "pcHeader:", "itablinks:", "typelinks:"} {
		if !strings.Contains(text, s) {
			t.Errorf("objdump -M: missing '%s' in output", s)
		}
	}
	if strings.Contains(text, "TEXT main.main") || strings.Contains(text, "fmthello.go:") {
		t.Errorf("objdump -M: unexpected assembly output")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataWithAssembly(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m"})
	if err != nil {
		t.Fatalf("objdump -m: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "moduledata:") {
		t.Errorf("objdump -m: missing moduledata output")
	}
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -m: missing assembly output (TEXT main.main)")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataVerification(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	tests := []struct {
		name     string
		flags    []string
		need     []string
		exitCode int
		canFail  bool
	}{
		{
			name:     "verificationFlagExists",
			flags:    []string{"-d"},
			need:     []string{"verifying moduledata integrity", "no issues found"},
			exitCode: 0,
		},
		{
			name:     "verificationWithMFlag",
			flags:    []string{"-d", "-M"},
			need:     []string{"verifying moduledata integrity", "no issues found"},
			exitCode: 0,
		},
		{
			name:     "verificationWithMAndPFlags",
			flags:    []string{"-d", "-m", "-p"},
			need:     []string{"verifying moduledata integrity", "no issues found"},
			exitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := testenv.Command(t, testenv.Executable(t), tt.flags...)
			cmd.Args = append(cmd.Args, hello)
			out, _ := cmd.CombinedOutput()
			t.Logf("Running %v", cmd.Args)

			text := string(out)
			for _, s := range tt.need {
				if !strings.Contains(text, s) {
					t.Errorf("objdump %v: missing '%s' in output", tt.flags, s)
				}
			}

			if t.Failed() {
				t.Logf("full output:\n%s", text)
			}
		})
	}
}

func TestModuledataVerificationWithSymbolFilter(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	// Test -d with -s: should verify and produce disassembly
	cmd := testenv.Command(t, testenv.Executable(t), []string{"-d", "-s", "main.main"}...)
	cmd.Args = append(cmd.Args, hello)
	out, _ := cmd.CombinedOutput()
	t.Logf("Running %v", cmd.Args)

	text := string(out)
	if !strings.Contains(text, "verifying moduledata integrity") {
		t.Errorf("objdump -d -s: missing verification message")
	}

	if !strings.Contains(text, "no issues found") {
		t.Errorf("objdump -d -s on clean binary should print 'no issues found'")
	}

	// With -s, disassembly should be produced
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -d -s: should produce disassembly for matching function")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestITabLinks(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}

	tmp := t.TempDir()
	hello := filepath.Join(tmp, "nested_structs_demo")

	args := []string{"build", "-gcflags=-N -l", "-o", hello, "testdata/nested_structs_demo.go"}
	cmd := testenv.Command(t, testenv.GoToolPath(t), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build nested_structs_demo.go: %v\n%s", err, out)
	}

	out, err = runObjdump(t, hello, []string{"-I"})
	if err != nil {
		t.Fatalf("objdump -I: %v\n%s", err, out)
	}

	text := string(out)

	checks := make(map[string]bool)

	checks["itablinks header"] = strings.Contains(text, "itablinks:")
	checks["itab structure"] = strings.Contains(text, "itab:{Inter:{InterfaceType:")
	checks["Imethod entries"] = strings.Contains(text, "Imethod:{Name:")
	checks["no corruption (binary)"] = !strings.Contains(text, "binary")

	interfaceTypes := []string{
		"TypeName:main.CustomReader",
		"TypeName:main.CustomWriter",
		"TypeName:main.CustomCloser",
		"TypeName:main.ReadWriteCloser",
		"TypeName:main.Formatter",
		"TypeName:main.Stringer",
	}
	for _, it := range interfaceTypes {
		checks[it] = strings.Contains(text, it)
	}

	checks["error interface"] = strings.Contains(text, "error")

	imethods := []string{"Error", "Read", "Write", "Close", "Format", "String"}
	for _, name := range imethods {
		checks["Name:"+name] = strings.Contains(text, "Name:"+name)
	}

	for name, ok := range checks {
		if !ok {
			t.Errorf("failed: %s", name)
		}
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestPrintTypes(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}

	tmp := t.TempDir()
	hello := filepath.Join(tmp, "types_demo")

	args := []string{"build", "-gcflags=-N -l", "-o", hello, "testdata/types_demo.go"}
	cmd := testenv.Command(t, testenv.GoToolPath(t), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build types_demo.go: %v\n%s", err, out)
	}

	out, err = runObjdump(t, hello, []string{"-T"})
	if err != nil {
		t.Fatalf("objdump -T: %v\n%s", err, out)
	}

	text := string(out)

	checks := make(map[string]bool)

	checks["typelinks header"] = strings.Contains(text, "typelinks:")
	checks["SimpleStruct pointer"] = strings.Contains(text, "*main.SimpleStruct")
	checks["WithArray pointer"] = strings.Contains(text, "*main.WithArray")
	checks["WithSlice pointer"] = strings.Contains(text, "*main.WithSlice")
	checks["WithMap pointer"] = strings.Contains(text, "*main.WithMap")
	checks["SimpleInterface pointer"] = strings.Contains(text, "*main.SimpleInterface")
	checks["AnotherInterface pointer"] = strings.Contains(text, "*main.AnotherInterface")
	checks["Implementor pointer"] = strings.Contains(text, "*main.Implementor")
	checks["types printed as pointers (Kind:ptr)"] = strings.Contains(text, "Kind:ptr")
	checks["Elem info present for pointers"] = strings.Contains(text, "Elem:main.SimpleStruct")
	checks["ElemHash present for pointers"] = strings.Contains(text, "ElemHash:0x")
	checks["Size info present"] = strings.Contains(text, "Size:")
	checks["Hash info present"] = strings.Contains(text, "Hash:0x")
	checks["no corruption error"] = !strings.Contains(text, "signal SIGSEGV")

	for name, ok := range checks {
		if !ok {
			t.Errorf("failed: %s", name)
		}
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestPrintTypesNestedStructs(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}

	tmp := t.TempDir()
	hello := filepath.Join(tmp, "nested_structs_demo")

	args := []string{"build", "-gcflags=-N -l", "-o", hello, "testdata/nested_structs_demo.go"}
	cmd := testenv.Command(t, testenv.GoToolPath(t), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build nested_structs_demo.go: %v\n%s", err, out)
	}

	out, err = runObjdump(t, hello, []string{"-T", "-t", "1"})
	if err != nil {
		t.Fatalf("objdump -T -t 1: %v\n%s", err, out)
	}

	text := string(out)

	checks := make(map[string]bool)

	checks["no SIGSEGV"] = !strings.Contains(text, "signal SIGSEGV")
	checks["typelinks header"] = strings.Contains(text, "typelinks:")
	checks["has types output"] = strings.Contains(text, "TYPE[")

	checks["types printed as pointers (Kind:ptr)"] = strings.Contains(text, "Kind:ptr")
	checks["Elem info present"] = strings.Contains(text, "Elem:main.")

	checks["Hash values present"] = strings.Contains(text, "Hash:0x")

	for name, ok := range checks {
		if !ok {
			t.Errorf("failed: %s", name)
		}
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestPrintTypesDepthLevels(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}

	tmp := t.TempDir()
	hello := filepath.Join(tmp, "nested_structs_demo")

	args := []string{"build", "-gcflags=-N -l", "-o", hello, "testdata/nested_structs_demo.go"}
	cmd := testenv.Command(t, testenv.GoToolPath(t), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build nested_structs_demo.go: %v\n%s", err, out)
	}

	for _, depth := range []string{"0", "1", "2"} {
		t.Run("depth="+depth, func(t *testing.T) {
			out, err = runObjdump(t, hello, []string{"-T", "-t", depth})
			if err != nil {
				t.Fatalf("objdump -T -t %s: %v\n%s", depth, err, out)
			}
			text := string(out)

			checks := make(map[string]bool)
			checks["no SIGSEGV"] = !strings.Contains(text, "signal SIGSEGV")
			checks["has output"] = len(text) > 100
			checks["typelinks header"] = strings.Contains(text, "typelinks:")

			switch depth {
			case "0":
				checks["no method details at depth 0"] = !strings.Contains(text, "Method[")
				checks["no field details at depth 0"] = !strings.Contains(text, "Field[")
			case "1", "2":
				checks["types printed as pointers"] = strings.Contains(text, "Kind:ptr")
				checks["Elem info present"] = strings.Contains(text, "Elem:")
				checks["Hash info present"] = strings.Contains(text, "Hash:0x")
			}

			for name, ok := range checks {
				if !ok {
					t.Errorf("depth %s: failed: %s", depth, name)
				}
			}

			if t.Failed() {
				t.Logf("full output:\n%s", text)
			}
		})
	}
}

func TestPrintTypesAlignmentGaps(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}

	tmp := t.TempDir()
	hello := filepath.Join(tmp, "types_demo")

	args := []string{"build", "-gcflags=-N -l", "-o", hello, "testdata/types_demo.go"}
	cmd := testenv.Command(t, testenv.GoToolPath(t), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build types_demo.go: %v\n%s", err, out)
	}

	out, err = runObjdump(t, hello, []string{"-T", "-t", "2"})
	if err != nil {
		t.Fatalf("objdump -T -t 2: %v\n%s", err, out)
	}

	text := string(out)

	checks := make(map[string]bool)

	checks["StructWithGaps present"] = strings.Contains(text, "*main.StructWithGaps")
	checks["Padding detected"] = strings.Contains(text, "[Padding:")

	checks["CompactStruct present"] = strings.Contains(text, "*main.CompactStruct")

	checks["7 bytes padding"] = strings.Contains(text, "[Padding: 7 bytes]")
	checks["3 bytes padding"] = strings.Contains(text, "[Padding: 3 bytes]")

	for name, ok := range checks {
		if !ok {
			t.Errorf("failed: %s", name)
		}
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataVerificationExitCodes(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	cmd := testenv.Command(t, testenv.Executable(t), []string{"-d"}...)
	cmd.Args = append(cmd.Args, hello)
	out, _ := cmd.CombinedOutput()
	t.Logf("Running %v", cmd.Args)

	text := string(out)
	if !strings.Contains(text, "verifying moduledata integrity") {
		t.Errorf("objdump -d: missing verification message")
	}

	if !strings.Contains(text, "no issues found") {
		t.Errorf("objdump -d on clean binary should print 'no issues found'")
	}

	if strings.Contains(text, "[ERROR]") {
		t.Errorf("objdump -d on clean binary should not have ERROR issues")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataVerificationWithFuncData(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	cmd := testenv.Command(t, testenv.Executable(t), []string{"-d", "-F"}...)
	cmd.Args = append(cmd.Args, hello)
	out, _ := cmd.CombinedOutput()
	t.Logf("Running %v", cmd.Args)

	text := string(out)
	if !strings.Contains(text, "verifying moduledata integrity") {
		t.Errorf("objdump -d -F: missing verification message")
	}

	if !strings.Contains(text, "no issues found") {
		t.Errorf("objdump -d -F on clean binary should print 'no issues found'")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataP(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-p"})
	if err != nil {
		t.Fatalf("objdump -p: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "pcHeader:") {
		t.Errorf("objdump -p: missing pcHeader output")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataAllWithAssembly(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m"})
	if err != nil {
		t.Fatalf("objdump -m: %v\n%s", err, out)
	}

	text := string(out)
	for _, s := range []string{"moduledata:", "pcHeader:", "itablinks:", "typelinks:"} {
		if !strings.Contains(text, s) {
			t.Errorf("objdump -m: missing '%s' in output", s)
		}
	}
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -m: missing assembly output")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestFilterWithAssembly(t *testing.T) {
	mustHaveDisasm(t)
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-s", "main\\.main"})
	if err != nil {
		t.Fatalf("objdump -s main\\.main: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -s main\\.main: missing main.main in output")
	}
	if strings.Contains(text, "TEXT main.Println") {
		t.Errorf("objdump -s main\\.main: unexpected main.Println in output")
	}
	if strings.Contains(text, "TEXT runtime.") {
		t.Errorf("objdump -s main\\.main: unexpected runtime symbols in output")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataWithSourceCode(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m", "-S", "-s", "main.main"})
	if err != nil {
		t.Fatalf("objdump -m -S -s main.main: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "moduledata:") {
		t.Errorf("objdump -m -S: missing moduledata output")
	}
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -m -S: missing assembly output")
	}
	if !strings.Contains(text, `Println("hello, world")`) {
		t.Errorf("objdump -m -S: missing source code line (Println)")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataWithGnuAsm(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m", "-gnu", "-s", "main.main"})
	if err != nil {
		t.Fatalf("objdump -m -gnu -s main.main: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "moduledata:") {
		t.Errorf("objdump -m -gnu: missing moduledata output")
	}
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -m -gnu: missing assembly output")
	}
	switch runtime.GOARCH {
	case "amd64", "386", "arm", "arm64":
		if !strings.Contains(text, "CALL") && !strings.Contains(text, "call") {
			t.Errorf("objdump -m -gnu: missing GNU assembly CALL instruction")
		}
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestFilterModuledataAndAssembly(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m", "-s", "main\\.(main|Print)"})
	if err != nil {
		t.Fatalf("objdump -m -s main\\.(main|Print): %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "moduledata:") {
		t.Errorf("objdump -m -s main\\.(main|Print): missing moduledata output")
	}
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -m -s main\\.(main|Print): missing main.main assembly")
	}
	if !strings.Contains(text, "TEXT main.Println") {
		t.Errorf("objdump -m -s main\\.(main|Print): missing main.Println assembly")
	}
	if strings.Contains(text, "TEXT runtime.") {
		t.Errorf("objdump -m -s main\\.(main|Print): unexpected runtime symbols in assembly")
	}
	if strings.Contains(text, "TEXT main.init") {
		t.Errorf("objdump -m -s main\\.(main|Print): unexpected main.init in assembly")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledata3ArgMode(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-s", "main.main"})
	if err != nil {
		t.Fatalf("objdump -s main.main: %v\n%s", err, out)
	}

	text := string(out)
	lines := strings.Split(text, "\n")
	var startPC string
	for _, line := range lines {
		if strings.Contains(line, "TEXT main.main(SB)") {
			continue
		}
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "\t0x") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.HasPrefix(p, "0x") {
					startPC = p
					break
				}
			}
		}
		if startPC != "" {
			break
		}
	}
	if startPC == "" {
		t.Skip("could not find start PC for main.main")
	}

	numPC, err := strconv.ParseUint(strings.TrimPrefix(startPC, "0x"), 16, 64)
	if err != nil {
		t.Fatalf("invalid start PC %q: %v", startPC, err)
	}
	endPC := fmt.Sprintf("0x%x", numPC+0x100)

	cmd := testenv.Command(t, testenv.Executable(t), "-m")
	cmd.Args = append(cmd.Args, hello, startPC, endPC)
	out, err = cmd.CombinedOutput()
	t.Logf("Running %v", cmd.Args)
	if err != nil {
		t.Fatalf("objdump -m binary 0x.. 0x..: %v\n%s", err, out)
	}

	text = string(out)
	if !strings.Contains(text, "moduledata:") {
		t.Errorf("3-arg mode: missing moduledata output")
	}
}

func TestModuledataFList(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	tests := []struct {
		name string
		flag string
		need string
	}{
		{"pcsp", "pcsp", "PCSP:"},
		{"unsafepoint", "unsafepoint", "PCDATA_UnsafePoint:"},
		{"stackmap", "stackmap", "PCDATA_StackMapIndex:"},
		{"inltree", "inltree", "PCDATA_InlTreeIndex:"},
		{"arglive", "arglive", "PCDATA_ArgLiveIndex:"},
		{"block", "block", "FUNCDATA_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runObjdump(t, hello, []string{"-f", tt.flag, "-s", "main.main"})
			if err != nil {
				t.Fatalf("objdump -f %s -s main.main: %v\n%s", tt.flag, err, out)
			}

			text := string(out)
			if !strings.Contains(text, tt.need) {
				t.Errorf("objdump -f %s -s main.main: missing '%s' in output", tt.flag, tt.need)
			}
		})
	}
}

func TestModuledataNOmitVersionCheck(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-m", "-n", "-s", "main.main"})
	if err != nil {
		t.Fatalf("objdump -m -n -s main.main: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "moduledata:") {
		t.Errorf("objdump -m -n: missing moduledata output")
	}
}

func TestModuledataFWithFilter(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-F", "-s", "main\\.main"})
	if err != nil {
		t.Fatalf("objdump -F -s main.main: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "PCSP:") || !strings.Contains(text, "PCDATA_") || !strings.Contains(text, "FUNCDATA_") {
		t.Errorf("objdump -F -s main.main: missing funcdata in output")
	}
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -F -s main.main: missing main.main assembly")
	}
	if strings.Contains(text, "TEXT runtime.") {
		t.Errorf("objdump -F -s main.main: unexpected runtime symbols in assembly")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataFWithMultiple(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-F", "-s", "main\\.(main|Print)"})
	if err != nil {
		t.Fatalf("objdump -F -s main.(main|Print): %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -F -s main.(main|Print): missing main.main assembly")
	}
	if !strings.Contains(text, "TEXT main.Println") {
		t.Errorf("objdump -F -s main.(main|Print): missing main.Println assembly")
	}
	if strings.Contains(text, "TEXT runtime.") {
		t.Errorf("objdump -F -s main.(main|Print): unexpected runtime symbols in assembly")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}

func TestModuledataFWithoutFilter(t *testing.T) {
	if !canTestModuledata(t) {
		return
	}
	hello := buildTestBinary(t)

	out, err := runObjdump(t, hello, []string{"-F"})
	if err != nil {
		t.Fatalf("objdump -F: %v\n%s", err, out)
	}

	text := string(out)
	if !strings.Contains(text, "PCSP:") || !strings.Contains(text, "PCDATA_") || !strings.Contains(text, "FUNCDATA_") {
		t.Errorf("objdump -F: missing funcdata in output")
	}
	if !strings.Contains(text, "TEXT main.main") {
		t.Errorf("objdump -F: missing main.main assembly")
	}
	if !strings.Contains(text, "main.Println") {
		t.Errorf("objdump -F: missing main.Println")
	}

	if t.Failed() {
		t.Logf("full output:\n%s", text)
	}
}
