// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package ld

// HarmonyOS application sandboxes may deny fallocate with SIGSYS instead of
// returning ENOSYS. The linker already has a portable file-growth fallback.
func (out *OutBuf) fallocate(size uint64) error {
	return errNoFallocate
}
