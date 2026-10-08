// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package runtime_test

import (
	"runtime"
	"testing"
)

func TestOpenHarmonyRuntimePolicy(t *testing.T) {
	if !runtime.IsOpenharmony {
		t.Fatal("OpenHarmony identity is missing")
	}
	if runtime.GOOS != "linux" {
		t.Fatalf("Linux runtime compatibility identity changed: %q", runtime.GOOS)
	}
	if runtime.OpenHarmonyHeapAddrBits != 48 {
		t.Fatalf("heap address ceiling = %d, want upstream 48-bit handling", runtime.OpenHarmonyHeapAddrBits)
	}
	if runtime.PreemptMSupported {
		t.Fatal("asynchronous preemption must not take over reserved SIGURG")
	}
	reserved, application, faults := runtime.OpenHarmonySignalPolicy()
	if !reserved || !application || !faults {
		t.Fatalf("signal policy: reserved=%v application=%v synchronous faults=%v", reserved, application, faults)
	}
}

func TestOpenHarmonySynchronousFault(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil pointer fault did not become a Go panic")
		}
	}()
	var p *int
	_ = *p
}
