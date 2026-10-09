// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import "testing"

func TestOpenHarmonySupport(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		if !CgoSupported("openharmony", arch) || !MustLinkExternal("openharmony", arch, true) || !DefaultPIE("openharmony", arch, false, true) {
			t.Errorf("openharmony/%s must support cgo with external PIE linking", arch)
		}
		if MustLinkExternal("openharmony", arch, false) || DefaultPIE("openharmony", arch, false, false) {
			t.Errorf("openharmony/%s must support standalone executables without cgo", arch)
		}
		for _, mode := range []string{"archive", "default", "exe", "pie", "c-shared", "c-archive"} {
			if !BuildModeSupported("gc", mode, "openharmony", arch) {
				t.Errorf("openharmony/%s unexpectedly rejects %s", arch, mode)
			}
		}
		for _, mode := range []string{"shared", "plugin"} {
			if BuildModeSupported("gc", mode, "openharmony", arch) {
				t.Errorf("openharmony/%s must not advertise unverified %s", arch, mode)
			}
		}
		if RaceDetectorSupported("openharmony", arch) || ASanSupported("openharmony", arch) || MSanSupported("openharmony", arch) || InternalLinkPIESupported("openharmony", arch) {
			t.Errorf("openharmony/%s advertises an unsupported mode", arch)
		}
	}
	if CgoSupported("openharmony", "386") || BuildModeSupported("gc", "exe", "openharmony", "arm") {
		t.Error("32-bit OpenHarmony targets must not be advertised")
	}
}
