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
	"syscall"
)

func init() {
	platformZoneSources = []string{
		"/etc/zoneinfo/tzdata", // Packed tzdata used by earlier releases.
		"/etc/zoneinfo/",       // TZif layouts used by current OHOS musl.
		"/usr/share/zoneinfo/",
		"/share/zoneinfo/",
	}

	loadTzinfoFromTzdata = ohosLoadTzinfoFromTzdata
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
