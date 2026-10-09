// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package net

import (
	"internal/bytealg"
	"os"
	"syscall"
	"unsafe"
)

// OpenHarmony can deny RTM_GETLINK while allowing RTM_GETADDR. Recover
// interface indices from the address dump, as musl's getifaddrs does, and
// retrieve link properties through ioctl.
func openharmonyInterfaceTable(ifindex int) ([]Interface, error) {
	tab, err := syscall.NetlinkRIB(syscall.RTM_GETADDR, syscall.AF_UNSPEC)
	if err != nil {
		return nil, os.NewSyscallError("netlinkrib", err)
	}
	msgs, err := syscall.ParseNetlinkMessage(tab)
	if err != nil {
		return nil, os.NewSyscallError("parsenetlinkmessage", err)
	}
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer syscall.Close(s)
	var ift []Interface
	seen := make(map[int]bool)
	for _, m := range msgs {
		if m.Header.Type != syscall.RTM_NEWADDR || len(m.Data) < syscall.SizeofIfAddrmsg {
			continue
		}
		index := int((*syscall.IfAddrmsg)(unsafe.Pointer(&m.Data[0])).Index)
		if seen[index] || ifindex != 0 && index != ifindex {
			continue
		}
		seen[index] = true
		var ifr struct {
			name [16]byte
			data [24]byte
		}
		ioctl := func(cmd uintptr) error {
			_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), cmd, uintptr(unsafe.Pointer(&ifr)))
			if errno != 0 {
				return os.NewSyscallError("ioctl", errno)
			}
			return nil
		}
		*(*int32)(unsafe.Pointer(&ifr.data[0])) = int32(index)
		if err := ioctl(syscall.SIOCGIFNAME); err != nil {
			continue // The interface may have disappeared since the dump.
		}
		n := bytealg.IndexByte(ifr.name[:], 0)
		if n <= 0 {
			continue
		}
		ifi := Interface{Index: index, Name: string(ifr.name[:n])}
		if err := ioctl(syscall.SIOCGIFFLAGS); err != nil {
			return nil, err
		}
		ifi.Flags = linkFlags(uint32(*(*uint16)(unsafe.Pointer(&ifr.data[0]))))
		if err := ioctl(syscall.SIOCGIFMTU); err == nil {
			ifi.MTU = int(*(*int32)(unsafe.Pointer(&ifr.data[0])))
		}
		if err := ioctl(syscall.SIOCGIFHWADDR); err == nil {
			// Only Ethernet addresses have the six-byte layout used here.
			if *(*uint16)(unsafe.Pointer(&ifr.data[0])) == 1 {
				ifi.HardwareAddr = append(HardwareAddr(nil), ifr.data[2:8]...)
			}
		}
		ift = append(ift, ifi)
	}
	return ift, nil
}

// If the ifindex is zero, interfaceTable returns mappings of all
// network interfaces. Otherwise it returns a mapping of a specific
// interface.
func interfaceTable(ifindex int) ([]Interface, error) {
	tab, err := syscall.NetlinkRIB(syscall.RTM_GETLINK, syscall.AF_UNSPEC)
	if err != nil {
		return openharmonyInterfaceTable(ifindex)
	}
	msgs, err := syscall.ParseNetlinkMessage(tab)
	if err != nil {
		return nil, os.NewSyscallError("parsenetlinkmessage", err)
	}
	var ift []Interface
loop:
	for _, m := range msgs {
		switch m.Header.Type {
		case syscall.NLMSG_DONE:
			break loop
		case syscall.RTM_NEWLINK:
			ifim := (*syscall.IfInfomsg)(unsafe.Pointer(&m.Data[0]))
			if ifindex == 0 || ifindex == int(ifim.Index) {
				attrs, err := syscall.ParseNetlinkRouteAttr(&m)
				if err != nil {
					return nil, os.NewSyscallError("parsenetlinkrouteattr", err)
				}
				ift = append(ift, *newLink(ifim, attrs))
				if ifindex == int(ifim.Index) {
					break loop
				}
			}
		}
	}
	return ift, nil
}

// If the ifi is nil, interfaceAddrTable returns addresses for all
// network interfaces. Otherwise it returns addresses for a specific
// interface.
func interfaceAddrTable(ifi *Interface) ([]Addr, error) {
	tab, err := syscall.NetlinkRIB(syscall.RTM_GETADDR, syscall.AF_UNSPEC)
	if err != nil {
		return nil, os.NewSyscallError("netlinkrib", err)
	}
	msgs, err := syscall.ParseNetlinkMessage(tab)
	if err != nil {
		return nil, os.NewSyscallError("parsenetlinkmessage", err)
	}
	ifat, err := addrTable(ifi, msgs)
	if err != nil {
		return nil, err
	}
	return ifat, nil
}
