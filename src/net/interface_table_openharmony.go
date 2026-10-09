// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package net

import (
	"internal/bytealg"
	"internal/byteorder"
	"os"
	"syscall"
	"unsafe"
)

// The platform's getifaddrs sends its request on an unbound netlink socket.
// Unlike syscall.NetlinkRIB, do not bind: the app sandbox can allow route
// dumps while denying an explicit bind. The kernel assigns the port on send.
func openharmonyNetlinkMessages(proto uint16) ([]syscall.NetlinkMessage, error) {
	s, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, syscall.NETLINK_ROUTE)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}
	defer syscall.Close(s)
	var request [20]byte // nlmsghdr followed by rtgenmsg and alignment padding.
	byteorder.LEPutUint32(request[:4], uint32(len(request)))
	byteorder.LEPutUint16(request[4:6], proto)
	byteorder.LEPutUint16(request[6:8], syscall.NLM_F_DUMP|syscall.NLM_F_REQUEST)
	byteorder.LEPutUint32(request[8:12], 1)
	if err := syscall.Sendto(s, request[:], 0, nil); err != nil {
		return nil, os.NewSyscallError("sendto", err)
	}
	var result []syscall.NetlinkMessage
	for {
		// Keep each buffer alive for the messages whose Data slices refer to it.
		buf := make([]byte, 8192)
		n, from, err := syscall.Recvfrom(s, buf, syscall.MSG_DONTWAIT)
		if err != nil {
			return nil, os.NewSyscallError("recvfrom", err)
		}
		if sender, ok := from.(*syscall.SockaddrNetlink); !ok || sender.Pid != 0 {
			return nil, os.NewSyscallError("netlink", syscall.EINVAL)
		}
		messages, err := syscall.ParseNetlinkMessage(buf[:n])
		if err != nil {
			return nil, os.NewSyscallError("parsenetlinkmessage", err)
		}
		for _, m := range messages {
			if m.Header.Seq != 1 {
				return nil, os.NewSyscallError("netlink", syscall.EINVAL)
			}
			switch m.Header.Type {
			case syscall.NLMSG_DONE:
				return result, nil
			case syscall.NLMSG_ERROR:
				if len(m.Data) < 4 {
					return nil, os.NewSyscallError("netlink", syscall.EINVAL)
				}
				if code := int32(byteorder.LEUint32(m.Data[:4])); code < 0 {
					return nil, os.NewSyscallError("netlink", syscall.Errno(-code))
				}
			default:
				result = append(result, m)
			}
		}
	}
}

// OpenHarmony can deny RTM_GETLINK while allowing RTM_GETADDR. Recover
// interface indices from the address dump, as musl's getifaddrs does, and
// retrieve link properties through ioctl.
func openharmonyInterfaceTable(ifindex int) ([]Interface, error) {
	msgs, err := openharmonyNetlinkMessages(syscall.RTM_GETADDR)
	if err != nil {
		return nil, err
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
				for _, b := range ifr.data[2:8] {
					if b != 0 {
						ifi.HardwareAddr = append(HardwareAddr(nil), ifr.data[2:8]...)
						break
					}
				}
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
	msgs, err := openharmonyNetlinkMessages(syscall.RTM_GETLINK)
	if err != nil {
		return openharmonyInterfaceTable(ifindex)
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
	msgs, err := openharmonyNetlinkMessages(syscall.RTM_GETADDR)
	if err != nil {
		return nil, err
	}
	ifat, err := addrTable(ifi, msgs)
	if err != nil {
		return nil, err
	}
	return ifat, nil
}
