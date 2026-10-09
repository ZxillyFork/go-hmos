// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package time_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	. "time"
)

func TestOpenHarmonySystemTimeZone(t *testing.T) {
	fixture := func() []byte {
		b := make([]byte, 1024)
		binary.LittleEndian.PutUint32(b[40:], uint32(len(b)-48))
		// Root and the three components of persist.time.timezone.
		for i, key := range []string{"#", "persist", "time", "timezone"} {
			offset := 44 + i*40
			if i != 3 {
				binary.LittleEndian.PutUint32(b[offset+8:], uint32((i+1)*40))
			}
			binary.LittleEndian.PutUint16(b[offset+22:], uint16(len(key)))
			copy(b[offset+24:], key)
		}
		binary.LittleEndian.PutUint32(b[44+3*40+16:], 200)
		const name = "persist.time.timezone"
		b[244+5] = byte(len(name))
		binary.LittleEndian.PutUint16(b[244+6:], uint16(len("Asia/Shanghai")))
		copy(b[252:], name+"=Asia/Shanghai")
		return b
	}
	for _, test := range []struct {
		name   string
		mutate func([]byte) []byte
		want   string
	}{
		{"valid", func(b []byte) []byte { return b }, "Asia/Shanghai"},
		{"truncated", func(b []byte) []byte { return b[:260] }, ""},
		{"offset-outside-workspace", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[52:], 0xfffffff0); return b }, ""},
		{"cyclic-trie", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[84:], 40); b[108] = 'a'; return b }, ""},
		{"write-in-progress", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[244:], 0x80000000); return b }, ""},
		{"wrong-key", func(b []byte) []byte { b[252] = 'x'; return b }, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "parameter")
			if err := os.WriteFile(path, test.mutate(fixture()), 0600); err != nil {
				t.Fatal(err)
			}
			if got := OpenHarmonyReadTimeZone(path); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestOpenHarmonyPackedZones(t *testing.T) {
	makeZone := func() []byte {
		data := make([]byte, 24+48+1)
		copy(data, "tzdata2025a")
		binary.BigEndian.PutUint32(data[12:], 24)
		binary.BigEndian.PutUint32(data[16:], 72)
		copy(data[24:], "Etc/UTC")
		binary.BigEndian.PutUint32(data[68:], 1)
		data[72] = 'Z'
		return data
	}
	for _, tc := range []struct {
		name   string
		zone   string
		mutate func([]byte)
		valid  bool
	}{
		{"valid", "Etc/UTC", func([]byte) {}, true},
		{"prefix-is-not-name", "Etc/U", func([]byte) {}, false},
		{"reverse-offsets", "Etc/UTC", func(b []byte) { binary.BigEndian.PutUint32(b[16:], 1) }, false},
		{"huge-index", "Etc/UTC", func(b []byte) { binary.BigEndian.PutUint32(b[16:], 1<<30) }, false},
		{"partial-entry", "Etc/UTC", func(b []byte) { binary.BigEndian.PutUint32(b[16:], 71) }, false},
		{"huge-payload", "Etc/UTC", func(b []byte) { binary.BigEndian.PutUint32(b[68:], 1<<30) }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := makeZone()
			tc.mutate(data)
			file := filepath.Join(t.TempDir(), "tzdata")
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			got, err := OpenHarmonyLoadTzinfoFromTzdata(file, tc.zone)
			if tc.valid {
				if err != nil || string(got) != "Z" {
					t.Fatalf("got %q, %v", got, err)
				}
			} else if err == nil {
				t.Fatalf("accepted malformed/missing zone: %q", got)
			}
		})
	}
}
