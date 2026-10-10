// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"os"
	"runtime"
	"syscall"
)

func setDefaultSockopts(s, family, sotype int, ipv6only bool) error {
	if family == syscall.AF_INET6 && sotype != syscall.SOCK_RAW {
		// Allow both IP versions even if the OS default
		// is otherwise. Note that some operating systems
		// never admit this option.
		syscall.SetsockoptInt(s, syscall.IPPROTO_IPV6, syscall.IPV6_V6ONLY, boolint(ipv6only))
	}
	if (sotype == syscall.SOCK_DGRAM || sotype == syscall.SOCK_RAW) && family != syscall.AF_UNIX {
		// Allow broadcast.
		err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		if runtime.GOOS == "openharmony" && sotype == syscall.SOCK_DGRAM && (err == syscall.EACCES || err == syscall.EPERM) {
			// OpenHarmony's shell domain can send unicast datagrams but
			// cannot enable broadcast. Leave broadcast disabled; a later
			// attempt to send to a broadcast address will still fail.
			return nil
		}
		return os.NewSyscallError("setsockopt", err)
	}
	return nil
}

func setDefaultListenerSockopts(s int) error {
	// Allow reuse of recently-used addresses.
	return os.NewSyscallError("setsockopt", syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1))
}

func setDefaultMulticastSockopts(s int) error {
	// Allow multicast UDP and raw IP datagram sockets to listen
	// concurrently across multiple listeners.
	return os.NewSyscallError("setsockopt", syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1))
}
