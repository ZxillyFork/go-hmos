// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package buildid

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenHarmonyELFNote(t *testing.T) {
	// The SDK emits .note.ohos.ident with section alignment 8, but its
	// namesz=12 and descsz=4 fields use the usual 4-byte note alignment.
	// It need not be skipped when looking for another note.
	var data bytes.Buffer
	write := func(v any) {
		t.Helper()
		if err := binary.Write(&data, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	names := "\x00.shstrtab\x00.note.ohos.ident\x00.note.go.buildid\x00"
	const start = 64 + 4*64
	write(elf.Header64{
		Ident: [16]byte{0x7f, 'E', 'L', 'F', byte(elf.ELFCLASS64), byte(elf.ELFDATA2LSB), byte(elf.EV_CURRENT)},
		Type:  uint16(elf.ET_REL), Machine: uint16(elf.EM_AARCH64), Version: uint32(elf.EV_CURRENT),
		Shoff: 64, Ehsize: 64, Shentsize: 64, Shnum: 4, Shstrndx: 1,
	})
	write(elf.Section64{})
	write(elf.Section64{Name: 1, Type: uint32(elf.SHT_STRTAB), Off: start, Size: uint64(len(names)), Addralign: 1})
	off := uint64(start+len(names)+7) &^ 7
	write(elf.Section64{Name: 11, Type: uint32(elf.SHT_NOTE), Off: off, Size: 28, Addralign: 8})
	write(elf.Section64{Name: 28, Type: uint32(elf.SHT_NOTE), Off: off + 28, Size: 20, Addralign: 4})
	data.WriteString(names)
	data.Write(make([]byte, int(off)-data.Len()))
	write([3]uint32{12, 4, 1})
	data.WriteString("OHOS\x00\x00\x00\x00\x00\x00\x00\x00")
	write(uint32(1))
	write([3]uint32{4, 4, 4})
	data.WriteString("Go\x00\x00test")
	file := filepath.Join(t.TempDir(), "notes.o")
	if err := os.WriteFile(file, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		typ  int32
		want string
	}{{"OHOS\x00\x00\x00\x00\x00\x00\x00\x00", 1, "\x01\x00\x00\x00"}, {"Go\x00\x00", 4, "test"}} {
		got, err := ReadELFNote(file, tt.name, tt.typ)
		if err != nil || string(got) != tt.want {
			t.Errorf("ReadELFNote(%q) = %q, %v; want %q, nil", tt.name, got, err, tt.want)
		}
	}
}
