// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Objdump disassembles executable files.
//
// Usage:
//
//	go tool objdump [-s symregexp] binary
//
// Objdump prints a disassembly of all text symbols (code) in the binary.
// If the -s option is present, objdump only disassembles
// symbols with names matching the regular expression.
//
// Alternate usage:
//
//	go tool objdump binary start end
//
// In this mode, objdump disassembles the binary starting at the start address and
// stopping at the end address. The start and end addresses are program
// counters written in hexadecimal with optional leading 0x prefix.
// In this mode, objdump prints a sequence of stanzas of the form:
//
//	file:line
//	 address: assembly
//	 address: assembly
//	 ...
//
// Each stanza gives the disassembly for a contiguous range of addresses
// all mapped to the same original source file and line number.
// This mode is intended for use by pprof.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"cmd/internal/disasm"
	"cmd/internal/objfile"
	"cmd/internal/telemetry/counter"
)

var printModuledata = flag.Bool("m", false, "(Experimental, ELF only) print moduledata")
var printPCHeader = flag.Bool("p", false, "(Experimental, ELF only) print pcHeader")
var printITabLinks = flag.Bool("I", false, "(Experimental, ELF only) print moduledata itablinks")
var printTypes = flag.Bool("T", false, "(Experimental, ELF only) print moduledata types")
var printTypesDepth = flag.Int("t", 0, "(Experimental, ELF only) print types detail depth (0=name only, 1=fields/methods, 2+=nested types) (implies -T)")
var funcDataModeFlag = flag.String("f", "", "(Experimental, ELF only) comma-separated list: pcsp,unsafepoint,stackmap,inltree,arglive,block")
var funcDataBothFlag = flag.Bool("F", false, "(Experimental, ELF only) short for -f block,pcsp,unsafepoint")
var moduleDataOnly = flag.Bool("M", false, "(Experimental, ELF only) print all moduledata sections, without assembly")
var omitVersionCheck = flag.Bool("n", false, "Omit version check in moduledata")
var printFindfunctab = flag.Bool("b", false, "(Experimental, ELF only) print findfunctab bucket table")
var verifyModuledataFlag = flag.Bool("d", false, "(Debug) verify moduledata integrity")
var verifyAllBuckets = flag.Bool("D", false, "(Debug) verify all findfunctab buckets (slow, implies -d)")
var printCode = flag.Bool("S", false, "print Go code alongside assembly")
var symregexp = flag.String("s", "", "only dump symbols matching this regexp")
var gnuAsm = flag.Bool("gnu", false, "print GNU assembly next to Go assembly (where supported)")
var symRE *regexp.Regexp

func usage() {
	fmt.Fprintf(os.Stderr, "usage: go tool objdump [options] binary [start end]\n\n")
	fmt.Fprintf(os.Stderr, "Disassembly options:\n")
	fmt.Fprintf(os.Stderr, "  -S        print Go code alongside assembly\n")
	fmt.Fprintf(os.Stderr, "  -gnu      print GNU assembly next to Go assembly\n")
	fmt.Fprintf(os.Stderr, "  -s        only dump symbols matching regexp\n\n")
	fmt.Fprintf(os.Stderr, "Moduledata options (Experimental, ELF only):\n")
	fmt.Fprintf(os.Stderr, "  -m        print moduledata, with assembly\n")
	fmt.Fprintf(os.Stderr, "  -p        print pcHeader (only)\n")
	fmt.Fprintf(os.Stderr, "  -I        print moduledata itablinks (only)\n")
	fmt.Fprintf(os.Stderr, "  -T        print moduledata types (only)\n")
	fmt.Fprintf(os.Stderr, "  -t N      type detail depth (0=no details, 1=fields/methods, 2+=nested)\n")
	fmt.Fprintf(os.Stderr, "  -M        print all moduledata sections (only)\n")
	fmt.Fprintf(os.Stderr, "  -f        comma-separated funcdata list: pcsp,unsafepoint,stackmap,inltree,arglive,block\n")
	fmt.Fprintf(os.Stderr, "  -F        short for -f block,pcsp,unsafepoint\n")
	fmt.Fprintf(os.Stderr, "  -n        omit version check in moduledata\n")
	fmt.Fprintf(os.Stderr, "  -b        print findfunctab bucket table\n")
	fmt.Fprintf(os.Stderr, "  -d        (Debug) verify moduledata integrity\n")
	fmt.Fprintf(os.Stderr, "  -D        (Debug) verify all findfunctab buckets (slow, implies -d)\n\n")
	os.Exit(2)
}

func printModuledataIfNeeded(w io.Writer, mdp *objfile.Moduledata, filter *regexp.Regexp) {
	if *moduleDataOnly {
		if err := mdp.PrintModuledata(w); err != nil {
			log.Fatalf("%v", err)
		}
		if err := mdp.PrintPCHeader(w); err != nil {
			log.Fatalf("%v", err)
		}
		if err := mdp.PrintITabLinks(w, filter); err != nil {
			log.Fatalf("%v", err)
		}
		if err := mdp.PrintTypes(w, filter, *printTypesDepth); err != nil {
			log.Fatalf("%v", err)
		}
		return
	}
	if *printModuledata {
		if err := mdp.PrintModuledata(w); err != nil {
			log.Fatalf("%v", err)
		}
	}
	if *printPCHeader {
		if err := mdp.PrintPCHeader(w); err != nil {
			log.Fatalf("%v", err)
		}
	}
	if *printITabLinks {
		if err := mdp.PrintITabLinks(w, filter); err != nil {
			log.Fatalf("%v", err)
		}
	}
	if *printTypes || *printTypesDepth > 0 {
		if err := mdp.PrintTypes(w, filter, *printTypesDepth); err != nil {
			log.Fatalf("%v", err)
		}
	}
	if *printFindfunctab {
		if err := mdp.PrintFindfunctab(w); err != nil {
			log.Fatalf("%v", err)
		}
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("objdump: ")
	counter.Open()

	flag.Usage = usage
	flag.Parse()
	counter.Inc("objdump/invocations")
	counter.CountFlags("objdump/flag:", *flag.CommandLine)
	if flag.NArg() != 1 && flag.NArg() != 3 {
		usage()
	}

	if *symregexp != "" {
		re, err := regexp.Compile(*symregexp)
		if err != nil {
			log.Fatalf("invalid -s regexp: %v", err)
		}
		symRE = re
	}

	f, err := objfile.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var mdp *objfile.Moduledata

	mode := *funcDataModeFlag
	if *funcDataBothFlag {
		mode = "block,pcsp,unsafepoint"
	}
	hasBlock := strings.Contains(mode, "block")
	hasPCSP := strings.Contains(mode, "pcsp")
	hasUnsafePoint := strings.Contains(mode, "unsafepoint")
	hasStackMap := strings.Contains(mode, "stackmap")
	hasInlTree := strings.Contains(mode, "inltree")
	hasArgLive := strings.Contains(mode, "arglive")
	printFuncBlock := hasBlock
	printFuncData := printFuncBlock || hasPCSP || hasUnsafePoint || hasStackMap || hasInlTree || hasArgLive

	mdOpts := disasm.MDOptions{
		PrintFuncData:     printFuncData,
		InlinePCSP:        hasPCSP,
		InlineUnsafePoint: hasUnsafePoint,
		InlineStackMap:    hasStackMap,
		InlineInlTree:     hasInlTree,
		InlineArgLive:     hasArgLive,
	}

	if *printModuledata || *printPCHeader || *printITabLinks || *printTypes || *printTypesDepth > 0 || *moduleDataOnly || printFuncData || *printFindfunctab || *verifyModuledataFlag || *verifyAllBuckets {
		checkVersion := true
		if *omitVersionCheck {
			checkVersion = false
		}
		mdp, err = f.Moduledata(checkVersion)
		if err != nil {
			log.Fatalf("reading moduledata %s: %v", flag.Arg(0), err)
		}
		if printFuncBlock {
			objfile.SetPrintFuncData(true)
		}

		// Run moduledata verification if requested (-d or -D)
		if *verifyModuledataFlag || *verifyAllBuckets {
			fmt.Fprintf(os.Stderr, "objdump: verifying moduledata integrity...\n")

			// Build symbol list from ELF for accurate function boundaries
			syms, err := f.Symbols()
			if err != nil {
				log.Fatalf("reading symbols: %v", err)
			}
			var symbols []objfile.FuncSymbol
			for _, sym := range syms {
				if sym.Code == 'T' || sym.Code == 't' {
					symbols = append(symbols, objfile.FuncSymbol{
						Name: sym.Name,
						Addr: sym.Addr,
						Size: uint64(sym.Size),
					})
				}
			}

			// -D implies full verification (no -s filter), so it always uses VerifyModuledata
			if symRE != nil && !*verifyAllBuckets {
				// -d with -s: verify matching functions, then continue to disassembly
				vr := mdp.VerifyModuledataForFuncs(symbols, symRE)
				if len(vr.Issues) > 0 {
					fmt.Fprintf(os.Stderr, "%s\n", vr.String())
				} else {
					fmt.Fprintf(os.Stderr, "objdump: no issues found\n")
				}
				if vr.ExitCode() != 0 {
					os.Exit(vr.ExitCode())
				}
				// Continue to disassembly below
			} else {
				// -d without -s OR -D: full verification
				vr := mdp.VerifyModuledata(symbols)
				if len(vr.Issues) > 0 {
					fmt.Fprintf(os.Stderr, "%s\n", vr.String())
				} else {
					fmt.Fprintf(os.Stderr, "objdump: no issues found\n")
				}

				// -D: continue with full findfunctab bucket verification
				if *verifyAllBuckets {
					fmt.Fprintf(os.Stderr, "objdump: verifying all findfunctab buckets...\n")
					vr2 := mdp.VerifyAllFindfunctabBuckets()
					if len(vr2.Issues) > 0 {
						fmt.Fprintf(os.Stderr, "%s\n", vr2.String())
					} else {
						fmt.Fprintf(os.Stderr, "objdump: no issues found\n")
					}
					// Exit with error if either verification found issues
					if vr.ExitCode() != 0 || vr2.ExitCode() != 0 {
						os.Exit(1)
					}
					os.Exit(0)
				}

				// -d only: exit after moduledata verification
				os.Exit(vr.ExitCode())
			}
		}
	}

	dis, err := disasm.DisasmForFile(f)
	if err != nil {
		log.Fatalf("disassemble %s: %v", flag.Arg(0), err)
	}
	if mdp != nil {
		dis.SetModuledata(mdp)
	}

	switch flag.NArg() {
	default:
		usage()
	case 1:
		if mdp != nil {
			printModuledataIfNeeded(os.Stdout, mdp, symRE)
			if *moduleDataOnly || *printPCHeader || *printITabLinks || *printTypes || *printFindfunctab {
				return
			}
			if !printFuncData || printFuncBlock {
				if err := mdp.PrintFunc(os.Stdout, symRE, 0, ^uint64(0)); err != nil {
					log.Fatalf("%v", err)
				}
			}
		}
		if mdp == nil || !*moduleDataOnly {
			dis.Print(os.Stdout, symRE, 0, ^uint64(0), *printCode, *gnuAsm, mdOpts)
		}

	case 3:
		start, err := strconv.ParseUint(strings.TrimPrefix(flag.Arg(1), "0x"), 16, 64)
		if err != nil {
			log.Fatalf("invalid start PC: %v", err)
		}
		end, err := strconv.ParseUint(strings.TrimPrefix(flag.Arg(2), "0x"), 16, 64)
		if err != nil {
			log.Fatalf("invalid end PC: %v", err)
		}
		if mdp != nil {
			printModuledataIfNeeded(os.Stdout, mdp, symRE)
			if *moduleDataOnly || *printPCHeader || *printITabLinks || *printTypes || *printFindfunctab {
				return
			}
			if !printFuncData || printFuncBlock {
				if err := mdp.PrintFunc(os.Stdout, symRE, start, end); err != nil {
					log.Fatalf("%v", err)
				}
			}
		}
		if mdp == nil || !*moduleDataOnly {
			dis.Print(os.Stdout, symRE, start, end, *printCode, *gnuAsm, mdOpts)
		}
	}
}
