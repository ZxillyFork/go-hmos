// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package arm64

import (
	"bytes"
	"cmd/internal/obj"
	"fmt"
	"internal/buildcfg"
	"internal/testenv"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func runAssembler(t *testing.T, srcdata string) []byte {
	dir := t.TempDir()
	defer os.RemoveAll(dir)
	srcfile := filepath.Join(dir, "testdata.s")
	outfile := filepath.Join(dir, "testdata.o")
	os.WriteFile(srcfile, []byte(srcdata), 0644)
	cmd := testenv.Command(t, testenv.GoToolPath(t), "tool", "asm", "-S", "-o", outfile, srcfile)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm64")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("The build failed: %v, output:\n%s", err, out)
	}
	return out
}

func TestSplitImm24uScaled(t *testing.T) {
	tests := []struct {
		v       int32
		shift   int
		wantErr bool
		wantHi  int32
		wantLo  int32
	}{
		{
			v:      0,
			shift:  0,
			wantHi: 0,
			wantLo: 0,
		},
		{
			v:      0x1001,
			shift:  0,
			wantHi: 0x2,
			wantLo: 0xfff,
		},
		{
			v:      0xffffff,
			shift:  0,
			wantHi: 0xfff000,
			wantLo: 0xfff,
		},
		{
			v:       0xffffff,
			shift:   1,
			wantErr: true,
		},
		{
			v:      0xfe,
			shift:  1,
			wantHi: 0x0,
			wantLo: 0x7f,
		},
		{
			v:      0x10fe,
			shift:  1,
			wantHi: 0x0,
			wantLo: 0x87f,
		},
		{
			v:      0x2002,
			shift:  1,
			wantHi: 0x4,
			wantLo: 0xfff,
		},
		{
			v:      0xfffffe,
			shift:  1,
			wantHi: 0xffe000,
			wantLo: 0xfff,
		},
		{
			v:      0x1000ffe,
			shift:  1,
			wantHi: 0xfff000,
			wantLo: 0xfff,
		},
		{
			v:       0x1001000,
			shift:   1,
			wantErr: true,
		},
		{
			v:       0x1000001,
			shift:   1,
			wantErr: true,
		},
		{
			v:       0xfffffe,
			shift:   2,
			wantErr: true,
		},
		{
			v:      0x4004,
			shift:  2,
			wantHi: 0x8,
			wantLo: 0xfff,
		},
		{
			v:      0xfffffc,
			shift:  2,
			wantHi: 0xffc000,
			wantLo: 0xfff,
		},
		{
			v:      0x1002ffc,
			shift:  2,
			wantHi: 0xfff000,
			wantLo: 0xfff,
		},
		{
			v:       0x1003000,
			shift:   2,
			wantErr: true,
		},
		{
			v:       0xfffffe,
			shift:   3,
			wantErr: true,
		},
		{
			v:      0x8008,
			shift:  3,
			wantHi: 0x10,
			wantLo: 0xfff,
		},
		{
			v:      0xfffff8,
			shift:  3,
			wantHi: 0xff8000,
			wantLo: 0xfff,
		},
		{
			v:      0x1006ff8,
			shift:  3,
			wantHi: 0xfff000,
			wantLo: 0xfff,
		},
		{
			v:       0x1007000,
			shift:   3,
			wantErr: true,
		},
		// Unshifted hi cases - hi <= 0xfff fits directly
		{
			v:      7,
			shift:  3,
			wantHi: 7,
			wantLo: 0,
		},
		{
			v:      0x8ff7,
			shift:  3,
			wantHi: 0xfff,
			wantLo: 0xfff,
		},
		{
			v:      0x7ff8,
			shift:  3,
			wantHi: 0,
			wantLo: 0xfff,
		},
		{
			v:      0xfff,
			shift:  1,
			wantHi: 1,
			wantLo: 0x7ff,
		},
		{
			v:      0xfff,
			shift:  2,
			wantHi: 3,
			wantLo: 0x3ff,
		},
		{
			v:      0xfff,
			shift:  3,
			wantHi: 7,
			wantLo: 0x1ff,
		},
		{
			v:      0x1ffe,
			shift:  2,
			wantHi: 2,
			wantLo: 0x7ff,
		},
		{
			v:      0x1ffe,
			shift:  3,
			wantHi: 6,
			wantLo: 0x3ff,
		},
		{
			v:      0x1fff,
			shift:  1,
			wantHi: 1,
			wantLo: 0xfff,
		},
		{
			v:      0x1fff,
			shift:  2,
			wantHi: 3,
			wantLo: 0x7ff,
		},
		{
			v:      0x1fff,
			shift:  3,
			wantHi: 7,
			wantLo: 0x3ff,
		},
		{
			v:      0x1001,
			shift:  1,
			wantHi: 1,
			wantLo: 0x800,
		},
		{
			v:      0x1001,
			shift:  2,
			wantHi: 1,
			wantLo: 0x400,
		},
		{
			v:      0x1001,
			shift:  3,
			wantHi: 1,
			wantLo: 0x200,
		},
		{
			v:      0x1000,
			shift:  0,
			wantHi: 0x1,
			wantLo: 0xfff,
		},
		{
			v:      0x8000,
			shift:  3,
			wantHi: 0x8,
			wantLo: 0xfff,
		},
		{
			v:      0xfff,
			shift:  0,
			wantHi: 0,
			wantLo: 0xfff,
		},
		{
			v:      0x1ffe,
			shift:  1,
			wantHi: 0,
			wantLo: 0xfff,
		},
		{
			v:      0x3ffc,
			shift:  2,
			wantHi: 0,
			wantLo: 0xfff,
		},
		{
			v:      0x10fef,
			shift:  4,
			wantHi: 0xfff,
			wantLo: 0xfff,
		},
	}
	for _, test := range tests {
		hi, lo, err := splitImm24uScaled(test.v, test.shift)
		switch {
		case err == nil && test.wantErr:
			t.Errorf("splitImm24uScaled(%v, %v) succeeded, want error", test.v, test.shift)
		case err != nil && !test.wantErr:
			t.Errorf("splitImm24uScaled(%v, %v) failed: %v", test.v, test.shift, err)
		case !test.wantErr:
			if got, want := hi, test.wantHi; got != want {
				t.Errorf("splitImm24uScaled(%x, %x) - got hi %x, want %x", test.v, test.shift, got, want)
			}
			if got, want := lo, test.wantLo; got != want {
				t.Errorf("splitImm24uScaled(%x, %x) - got lo %x, want %x", test.v, test.shift, got, want)
			}
		}
	}
	for shift := 0; shift <= 3; shift++ {
		for v := int32(0); v < 0xfff000+0xfff<<shift; v = v + 1<<shift {
			hi, lo, err := splitImm24uScaled(v, shift)
			if err != nil {
				t.Fatalf("splitImm24uScaled(%x, %x) failed: %v", v, shift, err)
			}
			if hi+lo<<shift != v {
				t.Fatalf("splitImm24uScaled(%x, %x) = (%x, %x) is incorrect", v, shift, hi, lo)
			}
		}
	}

	// Test the unshifted hi range specifically, including unaligned values.
	// This exercises values where the unshifted path may be used.
	for shift := 0; shift <= 3; shift++ {
		maxUnshifted := int32(0xfff + 0xfff<<shift)
		for v := int32(0); v <= maxUnshifted; v++ {
			hi, lo, err := splitImm24uScaled(v, shift)
			if err != nil {
				t.Fatalf("splitImm24uScaled(%x, %x) failed: %v", v, shift, err)
			}
			if hi+lo<<shift != v {
				t.Fatalf("splitImm24uScaled(%x, %x) = (%x, %x) is incorrect", v, shift, hi, lo)
			}
		}
	}
}

// TestLarge generates a very large file to verify that large
// program builds successfully, in particular, too-far
// conditional branches are fixed, and also verify that the
// instruction's pc can be correctly aligned even when branches
// need to be fixed.
func TestLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip in short mode")
	}
	testenv.MustHaveGoBuild(t)

	// generate a very large function
	buf := bytes.NewBuffer(make([]byte, 0, 7000000))
	fmt.Fprintln(buf, "TEXT f(SB),0,$0-0")
	fmt.Fprintln(buf, "TBZ $5, R0, label")
	fmt.Fprintln(buf, "CBZ R0, label")
	fmt.Fprintln(buf, "BEQ label")
	fmt.Fprintln(buf, "PCALIGN $128")
	fmt.Fprintln(buf, "MOVD $3, R3")
	for i := 0; i < 1<<19; i++ {
		fmt.Fprintln(buf, "MOVD R0, R1")
	}
	fmt.Fprintln(buf, "label:")
	fmt.Fprintln(buf, "RET")

	// assemble generated file
	out := runAssembler(t, buf.String())

	pattern := `0x0080\s00128\s\(.*\)\tMOVD\t\$3,\sR3`
	matched, err := regexp.MatchString(pattern, string(out))

	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("The alignment is not correct: %t\n", matched)
	}
}

// Issue 20348.
func TestNoRet(t *testing.T) {
	runAssembler(t, "TEXT ·stub(SB),$0-0\nNOP\n")
}

// TestPCALIGN verifies the correctness of the PCALIGN by checking if the
// code can be aligned to the alignment value.
func TestPCALIGN(t *testing.T) {
	testenv.MustHaveGoBuild(t)

	code1 := "TEXT ·foo(SB),$0-0\nMOVD $0, R0\nPCALIGN $8\nMOVD $1, R1\nRET\n"
	code2 := "TEXT ·foo(SB),$0-0\nMOVD $0, R0\nPCALIGN $16\nMOVD $2, R2\nRET\n"
	// If the output contains this pattern, the pc-offset of "MOVD $1, R1" is 8 bytes aligned.
	out1 := `0x0008\s00008\s\(.*\)\tMOVD\t\$1,\sR1`
	// If the output contains this pattern, the pc-offset of "MOVD $2, R2" is 16 bytes aligned.
	out2 := `0x0010\s00016\s\(.*\)\tMOVD\t\$2,\sR2`
	var testCases = []struct {
		name string
		code string
		out  string
	}{
		{"8-byte alignment", code1, out1},
		{"16-byte alignment", code2, out2},
	}

	for _, test := range testCases {
		out := runAssembler(t, test.code)
		matched, err := regexp.MatchString(test.out, string(out))
		if err != nil {
			t.Fatal(err)
		}
		if !matched {
			t.Errorf("The %s testing failed!\ninput: %s\noutput: %s\n", test.name, test.code, out)
		}
	}
}

// The OpenHarmony TLS path must not alter existing platforms' register or
// thread-pointer alignment contracts.
func TestRuntimeTLSMacros(t *testing.T) {
	testenv.MustHaveGoBuild(t)
	root := testenv.GOROOT(t)
	assemble := func(t *testing.T, goos, file string, flags ...string) []byte {
		t.Helper()
		dir := t.TempDir()
		// tls_arm64.s includes go_asm.h but uses no generated offsets.
		if err := os.WriteFile(filepath.Join(dir, "go_asm.h"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		cmd := testenv.Command(t, testenv.GoToolPath(t), "tool", "asm", "-S",
			"-I", dir, "-I", filepath.Join(root, "pkg", "include"),
			"-I", filepath.Join(root, "src", "runtime"),
			"-D", "GOOS_"+goos, "-o", filepath.Join(t.TempDir(), "tls.o"), file)
		cmd.Args = append(cmd.Args[:len(cmd.Args)-1], append(flags, file)...)
		cmd.Env = append(cmd.Environ(), "GOOS="+goos, "GOARCH=arm64")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("assemble TLS for %s: %v\n%s", goos, err, out)
		}
		return out
	}
	t.Run("OpenHarmonyFrame", func(t *testing.T) {
		out := assemble(t, "openharmony", filepath.Join(root, "src", "runtime", "tls_arm64.s"), "-shared", "-D", "TLS_GD")
		for _, pattern := range []string{`MOVD.W\s+R30, -32\(RSP\)`, `MOVD.P\s+32\(RSP\), R30`} {
			if len(regexp.MustCompile(pattern).FindAll(out, -1)) != 2 {
				t.Fatalf("load_g and save_g must atomically save/restore LR at SP+0 (%s):\n%s", pattern, out)
			}
		}
		if bytes.Contains(out, []byte("R29")) || bytes.Contains(out, []byte("R25")) {
			t.Fatalf("TLS helpers must preserve frame pointer and callee-saved registers:\n%s", out)
		}
	})
	t.Run("DarwinAlignment", func(t *testing.T) {
		out := assemble(t, "darwin", filepath.Join(root, "src", "runtime", "tls_arm64.s"))
		if !regexp.MustCompile(`AND\s+\$-8, R0(, R0)?`).Match(out) {
			t.Fatalf("Darwin TLS must clear all three low pointer bits:\n%s", out)
		}
	})
	t.Run("LinuxRaceRegisters", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(root, "src", "runtime", "race_arm64.s"))
		if err != nil {
			t.Fatal(err)
		}
		start := bytes.Index(data, []byte("// Darwin may return unaligned thread pointer."))
		end := bytes.Index(data, []byte("// func runtime·raceread"))
		if start < 0 || end <= start {
			t.Fatal("cannot locate race TLS macros")
		}
		source := "#include \"textflag.h\"\n#include \"tls_arm64.h\"\n" + string(data[start:end]) +
			"\nGLOBL runtime·tls_g(SB), TLSBSS, $8\nTEXT raceTLS(SB), NOSPLIT|NOFRAME, $0-0\nload_g\nRET\n"
		file := filepath.Join(t.TempDir(), "race_tls.s")
		if err := os.WriteFile(file, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		out := assemble(t, "linux", file)
		if bytes.Contains(out, []byte("R27")) || !bytes.Contains(out, []byte("R11")) {
			t.Fatalf("Linux race TLS may clobber R0/R11, not callee-saved R27:\n%s", out)
		}
	})
}

func TestTLSOperandClassNames(t *testing.T) {
	if got := DRconv(C_TLS_GD); got != "TLS_GD" {
		t.Errorf("GD operand class = %q, want TLS_GD; regenerate anames7.go", got)
	}
	if got := DRconv(C_NCLASS); got != "NCLASS" {
		t.Errorf("last operand class = %q, want NCLASS", got)
	}
}

// Indexed TLS saves must have the same PCSP accounting as explicit ADD/SUB.
func TestOpenHarmonyTLSStackDelta(t *testing.T) {
	old := buildcfg.GOOS
	buildcfg.GOOS = "openharmony"
	defer func() { buildcfg.GOOS = old }()
	ctxt := obj.Linknew(&Linkarm64)
	ctxt.DiagFunc = func(format string, args ...interface{}) { t.Errorf(format, args...) }
	sym := &obj.LSym{Name: "tlsframe"}
	sym.Set(obj.AttrNoSplit, true)
	sym.Set(obj.AttrNoFrame, true)
	text := &obj.Prog{As: obj.ATEXT, From: obj.Addr{Sym: sym}, To: obj.Addr{Val: int32(0)}}
	push := &obj.Prog{As: AMOVD, Scond: C_XPRE, From: obj.Addr{Type: obj.TYPE_REG, Reg: REGLINK}, To: obj.Addr{Type: obj.TYPE_MEM, Reg: REGSP, Offset: -32}}
	pop := &obj.Prog{As: AMOVD, Scond: C_XPOST, From: obj.Addr{Type: obj.TYPE_MEM, Reg: REGSP, Offset: 32}, To: obj.Addr{Type: obj.TYPE_REG, Reg: REGLINK}}
	text.Link, push.Link, pop.Link = push, pop, &obj.Prog{As: obj.ARET}
	sym.NewFuncInfo().Text = text
	preprocess(ctxt, sym, func() *obj.Prog { return new(obj.Prog) })
	if push.Spadj != 32 || pop.Spadj != -32 {
		t.Fatalf("TLS push/pop Spadj = %d/%d, want 32/-32", push.Spadj, pop.Spadj)
	}
}
