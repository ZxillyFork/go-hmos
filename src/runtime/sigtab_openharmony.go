// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package runtime

// OpenHarmony reserves signals 1-34, and API 19 also uses 35-45.
// Preserve their dispositions except for synchronous faults required by Go
// and SIGPIPE. These handlers always go through libc's signal chain. Do not
// repurpose SIGURG or SIGPROF, or alter the kernel's fault signal numbers.
// os/signal does not expose reserved signals as application notifications.
var sigtable = [...]sigTabT{
	/* 0 */ {0, "SIGNONE: no trap"},
	/* 1 */ {0, "SIGHUP: terminal line hangup"},
	/* 2 */ {0, "SIGINT: interrupt"},
	/* 3 */ {0, "SIGQUIT: quit"},
	/* 4 */ {_SigThrow + _SigUnblock, "SIGILL: illegal instruction"},
	/* 5 */ {_SigThrow + _SigUnblock, "SIGTRAP: trace trap"},
	/* 6 */ {0, "SIGABRT: abort"},
	/* 7 */ {_SigPanic + _SigUnblock, "SIGBUS: bus error"},
	/* 8 */ {_SigPanic + _SigUnblock, "SIGFPE: floating-point exception"},
	/* 9 */ {0, "SIGKILL: kill"},
	/* 10 */ {0, "SIGUSR1: user-defined signal 1"},
	/* 11 */ {_SigPanic + _SigUnblock, "SIGSEGV: segmentation violation"},
	/* 12 */ {0, "SIGUSR2: user-defined signal 2"},
	/* 13 */ {_SigUnblock, "SIGPIPE: write to broken pipe"},
	/* 14 */ {0, "SIGALRM: alarm clock"},
	/* 15 */ {0, "SIGTERM: termination"},
	/* 16 */ {_SigThrow + _SigUnblock, "SIGSTKFLT: stack fault"},
	/* 17 */ {0, "SIGCHLD: child status has changed"},
	/* 18 */ {0, "SIGCONT: continue"},
	/* 19 */ {0, "SIGSTOP: stop, unblockable"},
	/* 20 */ {0, "SIGTSTP: keyboard stop"},
	/* 21 */ {0, "SIGTTIN: background read from tty"},
	/* 22 */ {0, "SIGTTOU: background write to tty"},
	/* 23 */ {0, "SIGURG: urgent condition on socket"},
	/* 24 */ {0, "SIGXCPU: cpu limit exceeded"},
	/* 25 */ {0, "SIGXFSZ: file size limit exceeded"},
	/* 26 */ {0, "SIGVTALRM: virtual alarm clock"},
	/* 27 */ {0, "SIGPROF: profiling alarm clock"},
	/* 28 */ {0, "SIGWINCH: window size change"},
	/* 29 */ {0, "SIGIO: i/o now possible"},
	/* 30 */ {0, "SIGPWR: power failure restart"},
	/* 31 */ {0, "SIGSYS: bad system call"},
	/* 32 */ {0, "signal 32"}, /* SIGCANCEL; see issue 6997 */
	/* 33 */ {0, "signal 33"}, /* SIGSETXID; see issues 3871, 9400, 12498 */
	/* 34 */ {0, "signal 34"}, /* musl SIGSYNCCALL; see issue 39343 */
	/* 35 */ {0, "signal 35"},
	/* 36 */ {0, "signal 36"},
	/* 37 */ {0, "signal 37"},
	/* 38 */ {0, "signal 38"},
	/* 39 */ {0, "signal 39"},
	/* 40 */ {0, "signal 40"},
	/* 41 */ {0, "signal 41"},
	/* 42 */ {0, "signal 42"},
	/* 43 */ {0, "signal 43"},
	/* 44 */ {0, "signal 44"},
	/* 45 */ {0, "signal 45"},
	/* 46 */ {_SigNotify, "signal 46"},
	/* 47 */ {_SigNotify, "signal 47"},
	/* 48 */ {_SigNotify, "signal 48"},
	/* 49 */ {_SigNotify, "signal 49"},
	/* 50 */ {_SigNotify, "signal 50"},
	/* 51 */ {_SigNotify, "signal 51"},
	/* 52 */ {_SigNotify, "signal 52"},
	/* 53 */ {_SigNotify, "signal 53"},
	/* 54 */ {_SigNotify, "signal 54"},
	/* 55 */ {_SigNotify, "signal 55"},
	/* 56 */ {_SigNotify, "signal 56"},
	/* 57 */ {_SigNotify, "signal 57"},
	/* 58 */ {_SigNotify, "signal 58"},
	/* 59 */ {_SigNotify, "signal 59"},
	/* 60 */ {_SigNotify, "signal 60"},
	/* 61 */ {_SigNotify, "signal 61"},
	/* 62 */ {_SigNotify, "signal 62"},
	/* 63 */ {_SigNotify, "signal 63"},
	/* 64 */ {_SigNotify, "signal 64"},
}
