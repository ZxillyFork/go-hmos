// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package pprof

import (
	"bytes"
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

// Heap profiles remain supported even though signal-based CPU profiling is not.
// The debug text path appends MaxRSS through the platform-specific rusage code.
func TestOpenHarmonyHeapProfileDebug(t *testing.T) {
	var buf bytes.Buffer
	if err := Lookup("heap").WriteTo(&buf, 1); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "# MaxRSS = ") {
		t.Fatal("heap profile lacks Linux-ABI MaxRSS output")
	}
}
