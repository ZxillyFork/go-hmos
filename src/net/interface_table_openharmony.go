// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build cgo && openharmony

package net

/*
#define _GNU_SOURCE 1
#include <ifaddrs.h>
#include <sys/socket.h>
*/
import "C"

import (
	"os"
	"syscall"
	"unsafe"
)

type ifreq struct {
	Name [16]byte
	Ifru [24]byte
}

func interfaceTable(ifindex int) ([]Interface, error) {
	// get all internet address
	var res *C.struct_ifaddrs
	gerrno, err := C.getifaddrs(&res)
	if gerrno != 0 {
		return nil, os.NewSyscallError("getifaddrs", err)
	}
	defer C.freeifaddrs(res)

	// create a socket for ioctl syscall
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer syscall.Close(s)

	var ifts []Interface
	processed := make(map[int]bool)
	for r := res; r != nil; r = r.ifa_next {
		ifaAddr := r.ifa_addr
		if ifaAddr == nil {
			continue
		}
		ift := Interface{
			Name:  C.GoString(r.ifa_name),
			Flags: linkFlags(uint32(r.ifa_flags)),
		}

		ifr := &ifreq{}
		copy(ifr.Name[:], ift.Name)
		// retrieve index
		_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), syscall.SIOCGIFINDEX, uintptr(unsafe.Pointer(ifr)))
		if ep != 0 {
			continue
		}
		ift.Index = int(*(*uint32)(unsafe.Pointer(&ifr.Ifru[0])))
		if processed[ift.Index] || (ifindex != 0 && ift.Index != ifindex) {
			continue
		}
		// retrieve mtu
		_, _, ep = syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), syscall.SIOCGIFMTU, uintptr(unsafe.Pointer(ifr)))
		if ep == 0 {
			ift.MTU = int(*(*uint32)(unsafe.Pointer(&ifr.Ifru[0])))
		}
		// retrieve mac addr
		_, _, ep = syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), syscall.SIOCGIFHWADDR, uintptr(unsafe.Pointer(ifr)))
		if ep == 0 {
			// ifr_ifru.ifru_hwaddr.sa_data, skip address family and length(2 bytes).
			var nonzero bool
			for _, b := range ifr.Ifru[2:8] {
				if b != 0 {
					nonzero = true
					break
				}
			}
			if nonzero {
				ift.HardwareAddr = ifr.Ifru[2:8]
			}
		}

		ifts = append(ifts, ift)
		processed[ift.Index] = true
	}

	return ifts, nil
}

func interfaceAddrTable(ifi *Interface) ([]Addr, error) {
	// get all internet address
	var res *C.struct_ifaddrs
	gerrno, err := C.getifaddrs(&res)
	if gerrno != 0 {
		return nil, os.NewSyscallError("getifaddrs", err)
	}
	defer C.freeifaddrs(res)

	// create a socket for ioctl syscall
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer syscall.Close(s)

	var addrs []Addr
	for r := res; r != nil; r = r.ifa_next {
		ifaAddr := r.ifa_addr
		if ifaAddr == nil {
			continue
		}
		ifr := &ifreq{}
		copy(ifr.Name[:], C.GoString(r.ifa_name))
		// retrieve index
		_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), syscall.SIOCGIFINDEX, uintptr(unsafe.Pointer(ifr)))
		if ep != 0 {
			continue
		}
		index := int(*(*uint32)(unsafe.Pointer(&ifr.Ifru[0])))
		if ifi != nil && ifi.Index != index {
			continue
		}

		rsa := (*syscall.RawSockaddrAny)(unsafe.Pointer(ifaAddr))
		switch rsa.Addr.Family {
		case syscall.AF_INET:
			sa := (*syscall.RawSockaddrInet4)(unsafe.Pointer(rsa))
			ipv4 := &IPNet{IP: IPv4(sa.Addr[0], sa.Addr[1], sa.Addr[2], sa.Addr[3])}

			maskAddr := r.ifa_netmask
			if maskAddr != nil {
				maskSa := (*syscall.RawSockaddrInet4)(unsafe.Pointer(maskAddr))
				ipv4.Mask = IPMask(append([]byte(nil), maskSa.Addr[:]...))
			}
			addrs = append(addrs, ipv4)
		case syscall.AF_INET6:
			sa := (*syscall.RawSockaddrInet6)(unsafe.Pointer(rsa))
			ipv6 := &IPNet{IP: make(IP, IPv6len)}
			copy(ipv6.IP, sa.Addr[:])
			maskAddr := r.ifa_netmask
			if maskAddr != nil {
				maskSa := (*syscall.RawSockaddrInet6)(unsafe.Pointer(maskAddr))
				ipv6.Mask = IPMask(append([]byte(nil), maskSa.Addr[:]...))
			}
			addrs = append(addrs, ipv6)
		}
	}

	return addrs, nil
}
