// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package x509

import "internal/goos"

// Possible certificate files; stop after finding one.
var certFiles = []string{
	"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
	"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
	"/etc/pki/tls/cacert.pem",                           // OpenELEC
	"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	"/etc/ssl/cert.pem",                                 // Alpine Linux
}

// Possible directories with certificate files; all will be read.
var certDirectories = []string{
	"/etc/ssl/certs",     // SLES10/SLES11, https://golang.org/issue/12139
	"/etc/pki/tls/certs", // Fedora/RHEL
}

func init() {
	if goos.IsOpenharmony == 1 {
		// Public system roots only. User/app trust policies require the
		// platform certificate service and must be supplied by the app.
		// The public CA bundle is also accessible from the shell domain,
		// which cannot enumerate the certificate directories.
		certFiles = []string{"/etc/ssl/certs/cacert.pem"}
		certDirectories = []string{"/etc/security/certificates"}
	}
	if goos.IsAndroid == 1 {
		certDirectories = append(certDirectories,
			"/system/etc/security/cacerts",    // Android system roots
			"/data/misc/keychain/certs-added", // User trusted CA folder
		)
	}
}
