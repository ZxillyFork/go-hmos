// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#ifdef GOOS_android
#define TLS_linux
#define TLSG_IS_VARIABLE
#endif
#ifdef GOOS_linux
#define TLS_linux
#endif
#ifdef GOOS_openharmony
#define TLS_linux
#endif
#ifdef TLS_linux
#define MRS_TPIDR_R0 WORD $0xd53bd040 // MRS TPIDR_EL0, R0
#endif

#ifdef GOOS_darwin
#define TLS_darwin
#endif
#ifdef GOOS_ios
#define TLS_darwin
#endif
#ifdef TLS_darwin
#define TLSG_IS_VARIABLE
#define MRS_TPIDR_R0 WORD $0xd53bd060 // MRS TPIDRRO_EL0, R0
#endif

#ifdef GOOS_freebsd
#define MRS_TPIDR_R0 WORD $0xd53bd040 // MRS TPIDR_EL0, R0
#endif

#ifdef GOOS_netbsd
#define MRS_TPIDR_R0 WORD $0xd53bd040 // MRS TPIDRRO_EL0, R0
#endif

#ifdef GOOS_openbsd
#define MRS_TPIDR_R0 WORD $0xd53bd040 // MRS TPIDR_EL0, R0
#endif

#ifdef GOOS_windows
#define TLS_windows
#endif
#ifdef TLS_windows
#define TLSG_IS_VARIABLE
#define MRS_TPIDR_R0 MOVD R18_PLATFORM, R0
#endif

// Define something that will break the build if
// the GOOS is unknown.
#ifndef MRS_TPIDR_R0
#define MRS_TPIDR_R0 unknown_TLS_implementation_in_tls_arm64_h
#endif

#ifdef GOOS_openharmony
#define MRS_TPIDR_R27 WORD $0xd53bd05b // MRS TPIDR_EL0, R27

#ifdef TLS_GD
// TLSDESC returns the offset in R0 and preserves the other registers.
// Save LR atomically at SP+0, as expected by the unwinder. The 32-byte
// frame protects the caller's saved FP below its SP from the C resolver.
// load_g/save_g may only clobber R0 and R27.
#define LOAD_TLS_G_R0 \
    MOVD.W LR, -32(RSP) \
    MOVD runtime·tls_g(SB), R0 \
    MOVD.P 32(RSP), LR
#else
#define LOAD_TLS_G_R0 MOVD runtime·tls_g(SB), R0
#endif
#endif
