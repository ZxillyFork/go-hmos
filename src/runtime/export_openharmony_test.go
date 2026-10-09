// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package runtime

// OpenHarmonySignalPolicy checks the signal table without registering any
// handlers or sending a signal reserved by the host operating system.
func OpenHarmonySignalPolicy() (reserved, application, faults bool) {
	reserved, application, faults = true, true, true
	for sig := 1; sig <= 45; sig++ {
		if sigtable[sig].flags&_SigNotify != 0 {
			reserved = false
		}
	}
	for sig := 32; sig <= 45; sig++ {
		if sigtable[sig].flags != 0 {
			reserved = false
		}
	}
	for sig := 46; sig < len(sigtable); sig++ {
		if sigtable[sig].flags&_SigNotify == 0 {
			application = false
		}
	}
	for _, sig := range []int{_SIGSEGV, _SIGBUS, _SIGFPE} {
		if sigtable[sig].flags&(_SigPanic|_SigUnblock) != _SigPanic|_SigUnblock {
			faults = false
		}
	}
	return
}

// Keep the upstream ARM64/AMD64 address ceiling. A 39-bit kernel does not
// establish a 39-bit user-space ABI for every OpenHarmony device.
const OpenHarmonyHeapAddrBits = heapAddrBits

// OpenHarmonyMadvDontNeedDefault is read after runtime debug initialization.
func OpenHarmonyMadvDontNeedDefault() bool { return debug.madvdontneed != 0 }
