// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"internal/byteorder"
	"io"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestOpenHarmonyDNSConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		length  int
		status  uint32
		servers []string
		want    []string
	}{
		{"IPv4-and-IPv6", 272, 0, []string{"192.0.2.53", "fe80::53%eth0"}, []string{"192.0.2.53:53", "[fe80::53%eth0]:53"}},
		{"invalid-address", 272, 0, []string{"dns.example", "2001:db8::53"}, []string{"[2001:db8::53]:53"}},
		{"empty", 272, 0, nil, nil},
		{"service-error", 272, 1, []string{"192.0.2.53"}, nil},
		{"truncated", 271, 0, []string{"192.0.2.53"}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "dns")
			listener, err := Listen("unix", path)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				c, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer c.Close()
				c.SetDeadline(time.Now().Add(10 * time.Second))
				var request [12]byte
				if _, err := io.ReadFull(c, request[:]); err != nil {
					done <- err
					return
				}
				if byteorder.LEUint32(request[:4]) != uint32(syscall.Getuid()) || byteorder.LEUint32(request[4:8]) != 1 || byteorder.LEUint32(request[8:]) != 0 {
					done <- syscall.EINVAL
					return
				}
				var response [272]byte
				byteorder.LEPutUint32(response[:4], test.status)
				byteorder.LEPutUint32(response[4:8], 1250)
				byteorder.LEPutUint32(response[8:12], 3)
				for i, server := range test.servers {
					copy(response[16+i*51:16+(i+1)*51], server)
				}
				// A stream read is not guaranteed to return a complete structure.
				for _, b := range response[:test.length] {
					if _, err := c.Write([]byte{b}); err != nil {
						done <- err
						return
					}
				}
				done <- nil
			}()
			conf, err := openharmonyDNSConfig(path)
			if serverErr := <-done; serverErr != nil {
				t.Fatal(serverErr)
			}
			if test.want == nil {
				if err == nil {
					t.Fatalf("accepted invalid response: %+v", conf)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(conf.servers, test.want) || conf.timeout != 1250*time.Millisecond || conf.attempts != 3 {
				t.Fatalf("unexpected DNS configuration: %+v", conf)
			}
		})
	}
}
