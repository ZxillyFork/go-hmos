# Validation record — 2026-10-08

This record distinguishes compilation, Linux-host ABI simulation, and real
OpenHarmony execution. **No OpenHarmony SDK-linked binary or actual device has
been executed for this change.** The port remains experimental and the PR is
a draft.

## Passed locally

- Complete three-stage `src/make.bash` bootstrap on Linux/amd64, using official
  Go 1.27.1 as `GOROOT_BOOTSTRAP` (GOMAXPROCS=4).
  Official bootstrap archive SHA-256:
  `63d339f0da5ab53635a56f2490a7984dfe12dfcff22ad749f63edaf590168445`.
- Focused tests: `internal/platform`, `internal/buildcfg`, `go/build`,
  `cmd/go/internal/imports`, `cmd/go/internal/modindex`, `cmd/internal/obj`,
  `cmd/internal/objabi`, `cmd/go/internal/cfg`, `cmd/go/internal/envcmd`,
  `cmd/go/internal/work`, and `cmd/go/internal/toolchain`.
- Independent review ran the API compatibility check with the new downstream
  API baseline, ARM64/AMD64 code-generation tests, and the
  `TestScript/build_openharmony` command-driver test. The script verifies both
  missing-cgo rejection and fail-closed toolchain auto-switching.
- Linux runtime signal/GODEBUG, runtime/cgo and CPU-profile regression tests;
  Windows/amd64 runtime test cross-compilation; Linux/arm64 and Darwin/arm64
  and amd64 runtime cross-compilation.
- `GOOS=openharmony` runtime **Go/assembly** compilation for ARM64 and AMD64.
  This does not compile/link against the OHOS SDK C library.
- Shell syntax checks for SDK download/build helpers. Missing required SDK
  inputs fail rather than falling back to a host compiler.

## Linux-host ABI simulations (not OHOS validation)

Compiled `GOOS=openharmony GOARCH=amd64 CGO_ENABLED=1` with **host GCC/glibc**,
using `-ldflags=-extldflags=-fuse-ld=bfd` where needed. These are deliberately
not described as SDK builds:

- c-shared output contains real `R_X86_64_TLSDESC` and no initial-exec TLS flag.
- `dlopen` fixture: pre-load environment/GODEBUG, 100 callbacks from four
  foreign pthreads, goroutines, GC, timers, recovered nil faults, and existing
  reserved-signal handler preservation passed.
- Network checks were explicitly omitted with `--no-network`: this execution
  environment rejects netlink. An **unmodified official Go** `net.Interfaces`
  probe also returns `operation not permitted`. The first full loader run
  failed on that check; it was not reclassified as a network pass.
- The inherited 39-bit ARM64 heap ceiling was removed after review found no
  platform-wide ABI guarantee. Upstream 48-bit handling and a regression
  assertion are retained; both target runtime packages were recompiled.
- Target runtime-policy/synchronous-fault tests and the explicit CPU-profile
  rejection test passed under the host loader.
- Target PIE hello executable ran; a target-built native `cmd/go` ran and
  reported `GOOS=openharmony`, `GOHOSTOS=openharmony`, and
  `go version go1.27.1 openharmony/amd64` without GOOS/GOARCH environment overrides.
- Netgo and netcgo compile checks passed. Packed timezone tests passed for a
  valid entry, exact-name matching, reversed/oversized index bounds, malformed
  entry size, and oversized payload rejection.

These checks exercise code paths, not the OHOS customized libc, actual SDK
headers, CPU ABI on ARM64, sandbox policy, or a signed application lifecycle.

## Blocked / never run

- Official public SDK 6.1 package and checksum requests returned a 195-byte
  `Site Unavailable` HTML response from the official distribution site in this
  environment. That is **not** a checksum or an SDK. No substitute or fake
  sysroot was used. Real SDK c-shared/c-archive/PIE and native-toolchain linking
  therefore remain **never run** locally.
- Physical OpenHarmony/HarmonyOS NEXT device, x86_64 full-system emulator,
  signed HAP/N-API integration, native self-bootstrap, and complete target
  runtime/stdlib suites: **never run**.
- Target DNS/network-id behavior, certificate-service integration, system
  timezone-parameter synchronization, application permissions/signing,
  foreground/background lifecycle, and memory-pressure stress: **not verified**.
- AMD64 TLS pseudo-instruction's interior PUSH/CALL/POP lacks independently
  represented PCSP metadata. Existing host tests are not an asynchronous
  unwind proof. See `doc/openharmony.md` for this and runtime restrictions.

## Reproduce the real next validation stage

1. Obtain the official public SDK and review its license.
2. Set `OHOS_NDK_HOME` to its native directory and run
   `GOARCH=arm64 BUILD_NATIVE_TOOLS=1 bash misc/openharmony/build.sh`, then AMD64.
3. The opt-in **OpenHarmony port → Run workflow → sdk=true** CI job runs the
   same checksum-verified SDK compilation/link/ELF checks. Its artifacts are
   named `openharmony-compile-only`. Default host CI does not run SDK tests.
4. Sign/deploy as required by the selected OS/device and run the documented
   loader and standard-library tests. Only those results can establish
   actual target execution. Retain OS/API/SDK/build identifiers in the report.
