// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parse the "tzdata" packed timezone file used on Openharmony.
// The format is lifted from __tz.c in
// openharmony/third_party_musl/porting/linux/user/src/time in the OHOS.

//go:build openharmony

package time

import (
	"errors"
	"internal/bytealg"
	"internal/byteorder"
	"syscall"
)

func init() {
	platformLocalZone = ohosLocalZone
	platformZoneSources = []string{
		"/etc/zoneinfo/tzdata", // Packed tzdata used by earlier releases.
		"/etc/zoneinfo/",       // TZif layouts used by current OHOS musl.
		"/usr/share/zoneinfo/",
		"/share/zoneinfo/",
	}

	loadTzinfoFromTzdata = ohosLoadTzinfoFromTzdata
}

func ohosLocalZone() string {
	// Parameter workspaces are split by SELinux label on standard systems.
	// param_storage is the equivalent workspace on systems without SELinux.
	for _, path := range []string{
		"/dev/__parameters__/u:object_r:time_param:s0",
		"/dev/__parameters__/param_storage",
	} {
		if value := ohosReadTimeZone(path); value != "" {
			return value
		}
	}
	return ""
}

// ohosReadTimeZone reads persist.time.timezone from the parameter service's
// shared trie. Use pread rather than a persistent mapping so initialization
// cannot retain a stale mapping when the service grows its workspace.
func ohosReadTimeZone(path string) string {
	fd, err := open(path)
	if err != nil {
		return ""
	}
	defer closefd(fd)
	var header [44]byte // ParamTrieHeader, through dataSize (before tail padding).
	if preadn(fd, header[:], 0) != nil {
		return ""
	}
	size := byteorder.LEUint32(header[40:44])
	if size < 24 || size > 10<<20 {
		return ""
	}
	read := func(b []byte, offset uint32) bool {
		return uint64(offset)+uint64(len(b)) <= uint64(size) && preadn(fd, b, int(offset)+44) == nil
	}
	var node [24]byte
	if !read(node[:], byteorder.LEUint32(header[36:40])) {
		return ""
	}
	steps := 0
	for _, part := range []string{"persist", "time", "timezone"} {
		offset := byteorder.LEUint32(node[8:12])
		for {
			steps++
			if offset == 0 || steps > 256 || !read(node[:], offset) {
				return ""
			}
			n := int(byteorder.LEUint16(node[22:24]))
			if n == 0 || n > 96 {
				return ""
			}
			var key [96]byte
			if !read(key[:n], offset+24) {
				return ""
			}
			if string(key[:n]) == part {
				break
			}
			// ParamTrieNode orders longer keys before shorter keys.
			if n > len(part) || n == len(part) && string(key[:n]) < part {
				offset = byteorder.LEUint32(node[0:4])
			} else {
				offset = byteorder.LEUint32(node[4:8])
			}
		}
	}
	offset := byteorder.LEUint32(node[16:20])
	if offset == 0 {
		return ""
	}
	const name = "persist.time.timezone"
	for tries := 0; tries < 3; tries++ {
		var entry [8]byte
		if !read(entry[:], offset) || entry[4] != 0 || int(entry[5]) != len(name) {
			return ""
		}
		commit := byteorder.LEUint32(entry[:4])
		if commit&0x80000000 != 0 { // PARAM_FLAGS_MODIFY
			continue
		}
		n := int(byteorder.LEUint16(entry[6:8]))
		if n == 0 || n >= 96 {
			return ""
		}
		var data [len(name) + 1 + 96]byte
		if !read(data[:len(name)+1+n], offset+8) || string(data[:len(name)]) != name || data[len(name)] != 0 {
			return ""
		}
		if !read(entry[:], offset) {
			return ""
		}
		if byteorder.LEUint32(entry[:4]) == commit {
			return string(data[len(name)+1 : len(name)+1+n])
		}
	}
	return ""
}

func ohosLoadTzinfoFromTzdata(file, name string) ([]byte, error) {
	const (
		headersize = 12 + 3*4
		namesize   = 40
		entrysize  = namesize + 2*4
	)
	if len(name) > namesize {
		return nil, errors.New(name + " is longer than the maximum zone name length (40 bytes)")
	}
	fd, err := open(file)
	if err != nil {
		return nil, err
	}
	defer closefd(fd)

	buf := make([]byte, headersize)
	if err := preadn(fd, buf, 0); err != nil {
		return nil, errors.New("corrupt tzdata file " + file)
	}
	d := dataIO{buf, false}
	if magic := d.read(6); string(magic) != "tzdata" {
		return nil, errors.New("corrupt tzdata file " + file)
	}
	d = dataIO{buf[12:], false}
	indexOff, _ := d.big4()
	dataOff, _ := d.big4()
	if indexOff < headersize || dataOff < indexOff || dataOff-indexOff > 16<<20 || (dataOff-indexOff)%entrysize != 0 {
		return nil, errors.New("corrupt tzdata file " + file)
	}
	indexSize := dataOff - indexOff
	entrycount := indexSize / entrysize
	buf = make([]byte, indexSize)
	if err := preadn(fd, buf, int(indexOff)); err != nil {
		return nil, errors.New("corrupt tzdata file " + file)
	}
	for i := 0; i < int(entrycount); i++ {
		entry := buf[i*entrysize : (i+1)*entrysize]
		// len(name) <= namesize is checked at function entry
		end := bytealg.IndexByte(entry[:namesize], 0)
		if end < 0 {
			end = namesize
		}
		if string(entry[:end]) != name {
			continue
		}
		d := dataIO{entry[namesize:], false}
		off, _ := d.big4()
		size, _ := d.big4()
		if size == 0 || size > 10<<20 || uint64(off)+uint64(dataOff)+uint64(size) > 1<<31-1 {
			return nil, errors.New("corrupt tzdata file " + file)
		}
		buf := make([]byte, size)
		if err := preadn(fd, buf, int(off+dataOff)); err != nil {
			return nil, errors.New("corrupt tzdata file " + file)
		}
		return buf, nil
	}
	return nil, syscall.ENOENT
}
