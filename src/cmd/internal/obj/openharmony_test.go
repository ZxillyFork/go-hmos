// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package obj_test

import (
	"internal/testenv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenHarmonyTLSDescriptors(t *testing.T) {
	testenv.MustHaveGoBuild(t)
	for _, arch := range []string{"arm64", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			dir := t.TempDir()
			source := "#define TLSBSS 256\nGLOBL tlsvar(SB), TLSBSS, $8\nTEXT tlsload(SB), 4, $0-0\n"
			want := "R_ARM64_TLS_GD"
			if arch == "arm64" {
				source += "MOVD tlsvar(SB), R0\nRET\n"
			} else {
				source += "MOVQ TLS, CX\nMOVQ 0(CX)(TLS*1), AX\nRET\n"
				want = "R_AMD64_TLS_GD"
			}
			file := filepath.Join(dir, "tls.s")
			if err := os.WriteFile(file, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := testenv.Command(t, testenv.GoToolPath(t), "tool", "asm", "-shared", "-tls=GD", "-S", "-o", filepath.Join(dir, "tls.o"), file)
			cmd.Env = append(cmd.Environ(), "GOOS=openharmony", "GOARCH="+arch)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("assembler: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), want) {
				t.Fatalf("missing %s:\n%s", want, out)
			}
		})
	}
}
