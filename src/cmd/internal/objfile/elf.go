// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parsing of ELF executables (Linux, FreeBSD, and so on).

package objfile

import (
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

type elfFile struct {
	elf *elf.File
	r   io.ReaderAt
}

func openElf(r io.ReaderAt) (rawFile, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	return &elfFile{f, r}, nil
}

func (f *elfFile) symbols() ([]Sym, error) {
	elfSyms, err := f.elf.Symbols()
	if err != nil {
		elfDSyms, err2 := f.elf.DynamicSymbols()
		if err2 != nil {
			return nil, err2
		}
		elfSyms = elfDSyms
	}

	var syms []Sym
	for _, s := range elfSyms {
		sym := Sym{Addr: s.Value, Name: s.Name, Size: int64(s.Size), Code: '?'}
		switch s.Section {
		case elf.SHN_UNDEF:
			sym.Code = 'U'
		case elf.SHN_COMMON:
			sym.Code = 'B'
		default:
			i := int(s.Section)
			if i < 0 || i >= len(f.elf.Sections) {
				break
			}
			sect := f.elf.Sections[i]
			switch sect.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
			case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
				sym.Code = 'T'
			case elf.SHF_ALLOC:
				sym.Code = 'R'
			case elf.SHF_ALLOC | elf.SHF_WRITE:
				sym.Code = 'D'
			}
		}
		if elf.ST_BIND(s.Info) == elf.STB_LOCAL {
			sym.Code += 'a' - 'A'
		}
		syms = append(syms, sym)
	}

	return syms, nil
}

func (f *elfFile) symbolSize(name string) (uint64, error) {
	elfSyms, err := f.elf.Symbols()
	if err != nil {
		return 0, err
	}
	for _, s := range elfSyms {
		if s.Name == name {
			return s.Size, nil
		}
	}
	return 0, fmt.Errorf("symbol %q not found", name)
}

func (f *elfFile) pcln() (textStart uint64, symtab, pclntab []byte, err error) {
	if sect := f.elf.Section(".text"); sect != nil {
		textStart = sect.Addr
	}

	sect := f.elf.Section(".gosymtab")
	if sect == nil {
		// try .data.rel.ro.gosymtab, for PIE binaries
		sect = f.elf.Section(".data.rel.ro.gosymtab")
	}
	if sect != nil {
		if symtab, err = sect.Data(); err != nil {
			return 0, nil, nil, err
		}
	} else {
		// if both sections failed, try the symbol
		symtab = f.symbolData("runtime.symtab", "runtime.esymtab")
	}

	sect = f.elf.Section(".gopclntab")
	if sect == nil {
		// try .data.rel.ro.gopclntab, for PIE binaries
		sect = f.elf.Section(".data.rel.ro.gopclntab")
	}
	if sect != nil {
		if pclntab, err = sect.Data(); err != nil {
			return 0, nil, nil, err
		}
	} else {
		// if both sections failed, try the symbol
		pclntab = f.symbolData("runtime.pclntab", "runtime.epclntab")
	}

	return textStart, symtab, pclntab, nil
}

func (f *elfFile) firstmoduledata() ([]byte, error) {
	size := uint64(unsafe.Sizeof(*(*moduledata)(nil)))
	buf := f.symbolDataSz("runtime.firstmoduledata", size)
	if buf == nil {
		return nil, fmt.Errorf("reading runtime.firstmoduledata symbol")
	}
	return buf, nil
}

func (f *elfFile) buildVersion() (string, error) {
	buf := f.symbolDataSz("runtime.buildVersion.str", 6)
	if buf == nil {
		return "", fmt.Errorf("reading runtime.buildVersion symbol")
	}
	return fmt.Sprintf("%s", buf), nil
}

func (f *elfFile) pcHeader(addr uint64) ([]byte, error) {
	var ph pcHeader
	size := uint64(unsafe.Sizeof(ph))
	return f.tableBufAt(addr, size)
}

func (f *elfFile) tableBufAt(addr uint64, size uint64) ([]byte, error) {
	return f.segmentDataAt(addr, size)
}

func (f *elfFile) text() (textStart uint64, text []byte, err error) {
	sect := f.elf.Section(".text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return
}

func (f *elfFile) goarch() string {
	switch f.elf.Machine {
	case elf.EM_386:
		return "386"
	case elf.EM_X86_64:
		return "amd64"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_AARCH64:
		return "arm64"
	case elf.EM_LOONGARCH:
		return "loong64"
	case elf.EM_PPC64:
		if f.elf.ByteOrder == binary.LittleEndian {
			return "ppc64le"
		}
		return "ppc64"
	case elf.EM_RISCV:
		if f.elf.Class == elf.ELFCLASS64 {
			return "riscv64"
		}
	case elf.EM_S390:
		return "s390x"
	}
	return ""
}

func (f *elfFile) loadAddress() (uint64, error) {
	for _, p := range f.elf.Progs {
		if p.Type == elf.PT_LOAD && p.Flags&elf.PF_X != 0 {
			// The memory mapping that contains the segment
			// starts at an aligned address. Apparently this
			// is what pprof expects, as it uses this and the
			// start address of the mapping to compute PC
			// delta.
			return p.Vaddr - p.Vaddr%p.Align, nil
		}
	}
	return 0, fmt.Errorf("unknown load address")
}

func (f *elfFile) dwarf() (*dwarf.Data, error) {
	return f.elf.DWARF()
}

func (f *elfFile) symbolData(start, end string) []byte {
	elfSyms, err := f.elf.Symbols()
	if err != nil {
		elfDSyms, err2 := f.elf.DynamicSymbols()
		if err2 != nil {
			return nil
		}
		elfSyms = elfDSyms
	}
	var addr, eaddr uint64
	for _, s := range elfSyms {
		if s.Name == start {
			addr = s.Value
		} else if s.Name == end {
			eaddr = s.Value
		}
		if addr != 0 && eaddr != 0 {
			break
		}
	}
	if addr == 0 || eaddr < addr {
		return nil
	}
	size := eaddr - addr
	return f.symbolDataAt(addr, size)
}

func (f *elfFile) symbolDataSz(start string, size uint64) []byte {
	elfSyms, err := f.elf.Symbols()
	if err != nil {
		elfDSyms, err2 := f.elf.DynamicSymbols()
		if err2 != nil {
			return nil
		}
		elfSyms = elfDSyms
	}
	var addr uint64
	for _, s := range elfSyms {
		if s.Name == start {
			addr = s.Value
			break
		}
	}
	if addr == 0 {
		return nil
	}
	return f.symbolDataAt(addr, size)
}

func (f *elfFile) symbolDataAt(addr uint64, size uint64) []byte {
	if size == 0 {
		return make([]byte, 0)
	}
	buf, err := f.segmentDataAt(addr, size)
	if err != nil {
		return nil
	}
	return buf
}

func (f *elfFile) findSectionForAddr(addr uint64) *elf.Section {
	for _, sect := range f.elf.Sections {
		if sect.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		if sect.Type == elf.SHT_NOBITS {
			continue
		}
		if sect.Addr <= addr && addr < sect.Addr+sect.Size {
			return sect
		}
	}
	return nil
}

func (f *elfFile) addrToFileOffset(addr uint64) (uint64, *elf.Prog, bool) {
	for _, prog := range f.elf.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		vmaEnd := prog.Vaddr + prog.Filesz
		if prog.Vaddr <= addr && addr < vmaEnd {
			return prog.Off + (addr - prog.Vaddr), prog, true
		}
	}
	return 0, nil, false
}

func (f *elfFile) segmentDataAt(addr uint64, size uint64) ([]byte, error) {
	if size == 0 {
		return []byte{}, nil
	}

	fileOff, prog, found := f.addrToFileOffset(addr)
	if !found {
		return nil, fmt.Errorf("address 0x%x not in any PT_LOAD segment", addr)
	}

	endOff := fileOff + size
	segEnd := prog.Off + prog.Filesz

	if endOff > segEnd {
		return nil, fmt.Errorf("address 0x%x (size %d) extends past PT_LOAD segment boundary (segment end 0x%x)", addr, size, segEnd)
	}

	data := make([]byte, size)
	_, err := f.r.ReadAt(data, int64(fileOff))
	if err != nil {
		return nil, fmt.Errorf("reading at file offset 0x%x: %v", fileOff, err)
	}
	return data, nil
}
