# gohmos: isolated OpenHarmony Go installer

The fork-specific equivalent of `gotip`: fetch a pinned source commit, build it
on your development computer, and invoke it without replacing your regular Go.
There are currently **no prebuilt go-hmos releases**. The first install takes
several minutes, rather than downloading an already-built toolchain.

## One-command installation

Install [official Go](https://go.dev/dl/) **1.24.6 or later** and
[Git](https://git-scm.com/downloads) first. Linux/macOS also need `bash`.
No host C compiler is required for this installer. Run the same command in
Linux/macOS terminals or Windows PowerShell:

```text
go run github.com/ZxillyFork/go-hmos/gohmos@feature/openharmony-go1.27 install
```

This branch is an experimental PR preview. Go resolves the installer revision
through its normal module download and checksum mechanisms; the installer then
builds the full 40-character source commit embedded in `main.go`, never a moving
source branch. For reproducible installation, replace the branch after `@` with
a reviewed **installer commit SHA** from the PR. That commit must include the
`gohmos/go.mod` file; the earlier source-only commit does not.

The command prints the exact path of the installed `gohmos` launcher. Normally:

```sh
# Linux/macOS
"$HOME/sdk/go-hmos/bin/gohmos" version
"$HOME/sdk/go-hmos/bin/gohmos" tool dist list
```

```powershell
# Windows PowerShell
& "$HOME/sdk/go-hmos/bin/gohmos.exe" version
& "$HOME/sdk/go-hmos/bin/gohmos.exe" tool dist list
```

For a custom location, append `--home "/path/with spaces/go-hmos"` to `install`.
The installed launcher remembers its location by its directory layout; no
permanent environment variables are needed. `GOHMOS_HOME` can override it.
You may add the printed `bin` directory to PATH yourself if desired. Installation
never modifies PATH, replaces a command named `go`, or runs `go env -w`.

## Use and upgrade

`gohmos` forwards ordinary Go arguments, preserving target settings such as
`GOOS`, `GOARCH`, `CGO_ENABLED`, and `CC`. It sets `GOROOT` to the selected fork
and `GOTOOLCHAIN=local` for that process so Go cannot silently switch to an
upstream toolchain that lacks this port. `gohmos root` prints the source and
SDK toolchain directory. `gohmos --version` identifies the installer/source pin;
`gohmos version` identifies the built Go toolchain.

To upgrade, run the one-line install command with a newer reviewed installer
revision. Repeating a revision reuses its completed build; the installed
`gohmos install` (or `download`) selects the source pinned by that launcher.
Existing completed toolchains remain available. To roll back, rerun an earlier
installer revision. No updates happen in the background.

During host bootstrap, target/compiler/Go configuration is isolated, cgo is
disabled for the host build, and build concurrency is limited to four. A source
fetch or build failure leaves the previously selected toolchain usable. New
builds happen in a temporary sibling directory and are selected only after both
OpenHarmony targets are verified. Concurrent installs fail with a lock message.
If a forcibly terminated process leaves an empty `.install-lock` directory,
check that no installer is running, remove that empty directory, then retry.
Unfinished `.build-*` directories can be removed when no installer is running.

The source revision is verified against Git's fetched commit ID. This pins
content; it is not a code-signing or device-compatibility guarantee. Git uses
normal HTTPS and your existing network/proxy configuration. No token is needed
for the public repository, and no credentials are saved by this installer.

## OpenHarmony SDK is still required

Installation produces **host** Go tools. Cross-linking OpenHarmony executables
or native libraries still requires the official SDK's matching Clang and
sysroot, `CGO_ENABLED=1`, and external linking. Read
[the port documentation](../doc/openharmony.md) before building for devices.
This installer does not download the SDK, deploy to a phone, or establish
commercial HarmonyOS NEXT compatibility.

## Verification

```sh
cd gohmos
go test ./...
# Optional actual source bootstrap from a local checkout:
GOHMOS_TEST_REPOSITORY=/absolute/path/to/go-hmos \
GOHMOS_TEST_COMMIT=<full-commit-sha> \
  go test -run '^TestRealInstallation$' -timeout 25m -v
```

The installer CI runs unit tests and source installation on Linux, macOS, and
Windows. The source installation test uses a directory containing spaces,
checks a cached reinstall, and compiles/runs a host smoke program. Host tests
do not substitute for OpenHarmony SDK linking or device execution.
