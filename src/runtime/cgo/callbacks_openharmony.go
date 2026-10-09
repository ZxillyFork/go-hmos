// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build openharmony

package cgo

import _ "unsafe" // for go:linkname

//go:cgo_import_static x_cgo_get_environ
//go:linkname x_cgo_get_environ x_cgo_get_environ
//go:linkname _cgo_get_environ _cgo_get_environ
var x_cgo_get_environ byte
var _cgo_get_environ = &x_cgo_get_environ

//go:cgo_import_static x_cgo_sigprocmask
//go:linkname x_cgo_sigprocmask x_cgo_sigprocmask
//go:linkname _cgo_sigprocmask _cgo_sigprocmask
var x_cgo_sigprocmask byte
var _cgo_sigprocmask = &x_cgo_sigprocmask
