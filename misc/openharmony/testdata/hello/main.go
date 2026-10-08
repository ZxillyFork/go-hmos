// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
)

func main() {
	if runtime.GOOS != "linux" || !runtime.IsOpenharmony {
		panic("wrong target identity")
	}
	fmt.Printf("openharmony/%s (runtime.GOOS=%s)\n", runtime.GOARCH, runtime.GOOS)
}
