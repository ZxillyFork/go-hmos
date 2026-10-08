// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build arm64 && openharmony

package cpu

func osInit() {
	// OpenHarmony uses the Linux ELF HWCAP ABI.
	hwcapInit("openharmony")
}
