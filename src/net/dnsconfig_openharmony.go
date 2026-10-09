// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"internal/bytealg"
	"internal/byteorder"
	"io"
	"net/netip"
	"syscall"
	"time"
)

func dnsReadConfig(filename string) *dnsConfig {
	if filename == "/etc/resolv.conf" {
		if conf, err := openharmonyDNSConfig("/dev/unix/socket/dnsproxyd"); err == nil {
			return conf
		}
	}
	return dnsReadConfigFile(filename)
}

// openharmonyDNSConfig uses the NetSys GET_CONFIG protocol, also used by
// musl's NetSysGetResolvConf. Network ID zero selects the calling UID's
// default network, including its VPN configuration.
func openharmonyDNSConfig(path string) (*dnsConfig, error) {
	deadline := time.Now().Add(5 * time.Second)
	c, err := (&Dialer{Deadline: deadline}).Dial("unix", path)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if err := c.SetDeadline(deadline); err != nil {
		return nil, err
	}
	var request [12]byte
	byteorder.LEPutUint32(request[:4], uint32(syscall.Getuid()))
	byteorder.LEPutUint32(request[4:8], 1) // GET_CONFIG
	if _, err := c.Write(request[:]); err != nil {
		return nil, err
	}
	// ResolvConfig has four 32-bit fields, five 51-byte nameservers,
	// and one byte of trailing structure padding on both supported ABIs.
	var response [272]byte
	if _, err := io.ReadFull(c, response[:]); err != nil {
		return nil, err
	}
	if int32(byteorder.LEUint32(response[:4])) != 0 {
		return nil, syscall.EIO
	}
	conf := &dnsConfig{ndots: 1, timeout: 5 * time.Second, attempts: 2}
	if timeout := int32(byteorder.LEUint32(response[4:8])); timeout > 0 {
		conf.timeout = time.Duration(timeout) * time.Millisecond
	}
	if attempts := byteorder.LEUint32(response[8:12]); attempts > 0 && attempts <= 5 {
		conf.attempts = int(attempts)
	}
	for i := 0; i < 5; i++ {
		b := response[16+i*51 : 16+(i+1)*51]
		end := bytealg.IndexByte(b, 0)
		if end <= 0 {
			continue
		}
		if addr, err := netip.ParseAddr(string(b[:end])); err == nil {
			conf.servers = append(conf.servers, JoinHostPort(addr.String(), "53"))
		}
	}
	if len(conf.servers) == 0 {
		return nil, syscall.ENOENT
	}
	conf.search = dnsDefaultSearch()
	return conf, nil
}
