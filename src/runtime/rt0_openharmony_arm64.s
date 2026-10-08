// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

TEXT _rt0_arm64_openharmony(SB),NOSPLIT,$0
	JMP	_rt0_arm64(SB)

// Musl invokes constructors without argc/argv. Do not pass indeterminate
// C argument registers to the runtime. libpreinit captures libc's environ;
// goargs and sysargs obtain argv and auxv from procfs when it is available.
TEXT _rt0_arm64_openharmony_lib(SB),NOSPLIT,$0
	MOVD	ZR, R0
	MOVD	ZR, R1
	JMP	_rt0_arm64_lib(SB)
