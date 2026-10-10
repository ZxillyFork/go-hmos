// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package syscall_test

import (
	"context"
	"internal/testenv"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// OpenHarmony uses a real-time signal for asynchronous preemption. Unlike
// SIGURG, its default disposition terminates the process. It must not remain
// pending when exec replaces the Go signal handler with the default handler.
func TestOpenHarmonyExecPreemption(t *testing.T) {
	const helper = "GO_TEST_OPENHARMONY_EXEC_PREEMPTION"
	if mode := os.Getenv(helper); mode != "" {
		runtime.LockOSThread()
		for i := 0; i < 10; i++ {
			// Keep this thread busy long enough for sysmon to request
			// preemption, including around unsuccessful exec calls.
			deadline := time.Now().Add(20 * time.Millisecond)
			for time.Now().Before(deadline) {
			}
			if mode == "failed-exec" {
				err := syscall.Exec("/does-not-exist/go-exec-preemption", []string{"missing"}, os.Environ())
				if err != syscall.ENOENT {
					t.Fatalf("failed Exec: got %v, want ENOENT", err)
				}
			}
			if mode == "thread-exit" {
				done := make(chan struct{})
				go func() {
					runtime.LockOSThread()
					deadline := time.Now().Add(20 * time.Millisecond)
					for time.Now().Before(deadline) {
					}
					close(done)
					// Exiting a locked goroutine destroys its thread. Any
					// undelivered preemption must leave the pending count.
				}()
				<-done
			}
		}
		// Use a non-Go executable: another Go runtime could install its
		// handler before the stray signal arrives and hide the failure.
		if err := syscall.Exec("/system/bin/sh", []string{"sh", "-c", "exit 0"}, os.Environ()); err != nil {
			t.Fatal(err)
		}
		return
	}

	testenv.MustHaveExec(t)
	if _, err := os.Stat("/system/bin/sh"); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"exec", "failed-exec", "thread-exit"} {
		t.Run(mode, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				cmd := exec.CommandContext(ctx, testenv.Executable(t), "-test.run=^TestOpenHarmonyExecPreemption$")
				cmd.Env = append(os.Environ(), helper+"="+mode, "GODEBUG=asyncpreemptoff=0")
				output, err := cmd.CombinedOutput()
				cancel()
				if err != nil {
					t.Fatalf("iteration %d: %v\n%s", i, err, output)
				}
			}
		})
	}
}
