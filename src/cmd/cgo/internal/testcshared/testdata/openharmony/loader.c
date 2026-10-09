// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#define _POSIX_C_SOURCE 200809L
#include <dlfcn.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>

typedef int (*go_check)(int);
static go_check check;
static void *worker(void *arg) {
  (void)arg;
  for (int i = 0; i < 25; i++) {
    int result = check(i);
    if (result) { fprintf(stderr, "GoCheck: %d\n", result); abort(); }
  }
  return NULL;
}
int main(int argc, char **argv) {
  if (argc != 2 && !(argc == 3 && !strcmp(argv[2], "--no-network"))) {
    fprintf(stderr, "usage: %s ./library.so [--no-network]\n", argv[0]); return 2;
  }
  int signals[] = { SIGURG, 35, 43 };
  struct sigaction before[3] = {0}, after[3] = {0};
  // Observe reserved handlers without installing or raising reserved signals.
  for (int i = 0; i < 3; i++) if (sigaction(signals[i], NULL, &before[i])) return 8;
  // Do not mutate C environ after dlopen while the Go runtime is starting.
  if (setenv("OHOS_GO_TEST", "before-dlopen", 1) || setenv("GODEBUG", "cpu.all=off", 1)) return 3;
  void *lib = dlopen(argv[1], RTLD_NOW | RTLD_LOCAL);
  if (!lib) { fprintf(stderr, "dlopen: %s\n", dlerror()); return 4; }
  check = (go_check)dlsym(lib, "GoCheck");
  if (!check) { fprintf(stderr, "dlsym: %s\n", dlerror()); return 5; }
  pthread_t threads[4];
  for (int i = 0; i < 4; i++) if (pthread_create(&threads[i], NULL, worker, NULL)) return 6;
  for (int i = 0; i < 4; i++) if (pthread_join(threads[i], NULL)) return 7;
  for (int i = 0; i < 3; i++) {
    if (sigaction(signals[i], NULL, &after[i])) return 9;
    if (before[i].sa_handler != after[i].sa_handler || before[i].sa_flags != after[i].sa_flags) {
      fputs("reserved handlers changed\n", stderr); return 10;
    }
  }
  if (argc == 2) {
    int (*network)(void) = (int (*)(void))dlsym(lib, "GoNetworkCheck");
    if (!network) return 11;
    for (int i = 0; i < 100; i++) if (network()) return 12;
    puts("PASS: 100 interface-discovery calls");
  } else puts("NOT RUN: network checks (--no-network explicitly selected)");
  puts("PASS: dlopen, environment, 100 foreign-pthread callbacks, GC, goroutines, timers, panic recovery, reserved signal handlers");
  // A Go runtime cannot be unloaded safely. Intentionally do not dlclose.
  return 0;
}
