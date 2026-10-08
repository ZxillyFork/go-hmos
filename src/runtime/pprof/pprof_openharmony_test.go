// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package pprof

import (
	"io"
	"strings"
	"testing"
)

func TestOpenHarmonyCPUProfileUnsupported(t *testing.T) {
	if err := StartCPUProfile(io.Discard); err == nil || !strings.Contains(err.Error(), "OpenHarmony") {
		StopCPUProfile()
		t.Fatalf("StartCPUProfile = %v, want explicit OpenHarmony unsupported error", err)
	}
	// Failing to start must not leave a writer running or CPU state active.
	StopCPUProfile()
	if cpu.profiling {
		t.Fatal("unsupported CPU profiler remained active")
	}
}
