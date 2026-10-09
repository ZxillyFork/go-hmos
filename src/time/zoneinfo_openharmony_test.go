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
