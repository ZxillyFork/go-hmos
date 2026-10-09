// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

#include <pthread.h>
#include <signal.h>
#include <stddef.h>
#include <stdint.h>

extern char **environ;

// Called on the C stack before the Go allocator is initialized. A musl
// constructor has no argc/argv/envp arguments. Capture the actual libc
// environment, including changes made before dlopen, without allocating.
void
x_cgo_get_environ(char ***envp) {
	*envp = environ;
}

// OpenHarmony's libc filters masks through its signal chain. Do not bypass
// it with the Linux rt_sigprocmask syscall during normal runtime operation.
// The runtime's kernel sigset is 64 bits; libc's sigset_t is not.
int32_t
x_cgo_sigprocmask(intptr_t how, const uint64_t *new, uint64_t *old) {
	sigset_t set, oset;
	int ret;
	unsigned int i;

	if (new != NULL) {
		sigemptyset(&set);
		for (i = 0; i < 64; i++) {
			if (*new & ((uint64_t)1 << i)) {
				sigaddset(&set, i + 1);
			}
		}
	}
	ret = pthread_sigmask((int)how, new != NULL ? &set : NULL,
		old != NULL ? &oset : NULL);
	if (ret == 0 && old != NULL) {
		*old = 0;
		for (i = 0; i < 64; i++) {
			if (sigismember(&oset, i + 1) == 1) {
				*old |= (uint64_t)1 << i;
			}
		}
	}
	return ret;
}
