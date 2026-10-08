# Experimental OpenHarmony port (Go 1.27)

This is a **downstream experimental port**, not a supported upstream Go target
or a certification of compatibility with commercial HarmonyOS NEXT devices.
Read the verification limits below before deploying it.

## Baseline and provenance

- Upstream: `golang/go`, `release-branch.go1.27` at
  `68fa7699a27d745f90bf8630202e1a415d1b0769` (2026-10-02).
- Latest stable at the 2026-10-08 check: **Go 1.27.1**, tag
  `862c888e612ac346c7c4d99c9392bdfd265f33b0`.
  The PR targets the current release-branch head, including fixes after the tag.
- Porting reference: [OpenHarmony SIG Go](https://gitcode.com/openharmony-sig/ohos_golang_go),
  and its Go 1.26.5 forward-port
  [star4277/ohos-go](https://github.com/star4277/ohos-go/tree/19667c7abfc332f848b5bd1f293e4aeb0e9991a9).
  The adapted delta is against upstream Go 1.26.5
  `c19862e5f8415b4f24b189d065ed739517c548ba`.
- The reference code carries the Go BSD license. Existing copyright notices,
  `LICENSE`, and `PATENTS` are retained. No GPL application code was copied.
  Unrelated mode changes and deleted binary test fixtures from the reference
  were deliberately excluded. Go shared/plugin machinery was not imported.

## Target and identity contract

- Build targets: `GOOS=openharmony GOARCH=arm64` and `GOARCH=amd64`.
  ARM64 is the intended physical-device architecture; AMD64 is intended for
  x86_64 OpenHarmony environments/emulators and remains experimental.
- The `openharmony`, `linux`, and `unix` build tags all match. `_linux.go`
  files are inherited unless explicitly excluded, as with Go's Android port.
- **For compatibility with the prior port, `runtime.GOOS` remains `"linux"`.**
  `runtime.IsOpenharmony` is true only on this target. `go env GOOS`,
  `GOHOSTOS`, native tool selection, and `go version` identify OpenHarmony.
  The extra API is recorded separately in `api/openharmony.txt`.
- Cgo and external linking are required, including for otherwise pure-Go
  executables. `CGO_ENABLED=0` fails with a diagnostic; it does not produce a
  misleading Linux binary. Executables default to PIE.
- Implemented build modes: ordinary executable/PIE, c-shared, c-archive,
  and Go package archives. Go plugin/shared modes, race/MSan/ASan, and
  standalone no-cgo programs are not advertised as supported.
- Automatic toolchain switching is rejected for native or cross-target OHOS
  builds. Stock Go downloads do not contain this port. Use `GOTOOLCHAIN=local`
  and rebuild the port when a dependency requires a newer Go version.

## Why a Linux rename is insufficient

1. **Dynamic TLS:** both architectures emit ELF TLSDESC relocations for Go TLS
   in c-shared/c-archive libraries. Musl cannot resolve a dlopen library's
   initial-exec TLS to a newly allocated dynamic TLS definition. The port uses
   real ELF TLS symbols, not a hard-coded private musl pthread/TCB slot.
   The upstream 48-bit heap-address handling is retained; the reference
   port's unverified 39-bit ARM64 ceiling is not carried forward.
   The Go 1.27 ARM64 instruction-case collision and relocation numbering are
   reconciled with upstream. Other platforms retain their prior TLS model.
2. **Musl constructors:** `.init_array` supplies no portable argc/argv contract.
   OHOS constructors zero these arguments and use Go 1.27's `libInit` path;
   libc environment capture and the early GODEBUG scan do not allocate before
   `mallocinit`. Arguments/auxv use procfs with existing fallback behavior.
3. **Signals:** OHOS libc has a signal chain. Registration and normal mask
   operations go through libc, rather than bypassing it with raw Linux
   registration. The fork-child reset path still uses the raw syscall because
   entering libc locks after fork is unsafe.
4. **Platform services:** libc DNS is preferred for OHOS's network-id/resolver
   integration; interface discovery uses `getifaddrs` with balanced cleanup,
   including with `-tags=netgo`. System roots use `/etc/security/certificates`;
   packed tzdata is read from `/etc/zoneinfo/tzdata`, with the standard
   OHOS TZif directory layouts as fallbacks. The native Go linker's
   optional fallocate optimization is disabled so a sandbox cannot kill it
   for an unsupported syscall; existing portable file growth is used.

## Build and inspect with the official SDK

Bootstrap on a supported host with a compatible official Go installation:

```sh
cd src
GOROOT_BOOTSTRAP=/path/to/official/go ./make.bash
cd ..
export GOTOOLCHAIN=local
export OHOS_NDK_HOME=/path/to/official/sdk/native
GOARCH=arm64 bash misc/openharmony/build.sh
GOARCH=amd64 bash misc/openharmony/build.sh
```

The native SDK directory must contain `llvm/bin/clang`, `llvm/bin/llvm-readelf`
and `sysroot`. Compiler triples are `aarch64-linux-ohos` and
`x86_64-linux-ohos`, **not** `aarch64-linux-gnu`/Android triples.
The helper verifies `__OHOS__`, headers, actual cross-links, c-shared/c-archive,
PIE, netgo file selection, TLS relocations, and the SDK dynamic interpreter:
`/lib/ld-musl-aarch64.so.1` or `/lib/ld-musl-x86_64.so.1`.
It also cross-compiles selected standard-library tests. A passed build is
explicitly reported as **not device execution**.

`misc/openharmony/download-sdk.sh` is an optional checksum-verifying helper for
public OpenHarmony 6.1 SDK 6.1.0.31 (API 23). Set `SDK_DOWNLOAD_DIR` to a scratch
folder. This is not Huawei's separately distributed HarmonyOS NEXT SDK.
Review the SDK's licenses before use. No login, signing key, or SDK is bundled.

To compile a native toolchain's commands, set `BUILD_NATIVE_TOOLS=1` when running
`build.sh`. The produced tools still require the source tree, standard Go
installation layout, a usable native C compiler, executable permissions,
platform-required signing, and a device policy allowing process execution.
This option only cross-compiles them; it does not prove native self-hosting.
A signed N-API bridge/HAP is normally required to call a Go c-shared library
from ArkTS. Creating or signing that app is outside this compiler change.

## Conservative runtime restrictions

Official NDK guidance reserves signals 1–34 and describes system use of 35–45.
This port does not repurpose SIGURG/SIGPROF or claim all Linux signals are free:

- Asynchronous preemption is disabled. Cooperative safe points remain, but
  a tight non-cooperative loop can delay scheduling/GC. Test real workloads.
- `runtime/pprof.StartCPUProfile` returns an explicit unsupported error.
- `os/signal` does not subscribe to or ignore reserved signals 1–45.
  Existing host dispositions are preserved, except libc-chained synchronous
  fault handling needed for Go panics and SIGPIPE behavior.
- This is a deliberately conservative downstream policy. The documentation's
  reservation language is not evidence that musl rejects every sigaction call.
- Do not call `dlclose` on a live Go runtime. An OHOS loader may actually unload
  code, unlike assumptions made by some Linux applications.
- Set C environment variables before the first `dlopen`. Concurrent C
  `setenv`/`unsetenv` while the runtime reads `environ` remains unsafe; this
  patch does not introduce a universal process-wide environment lock.
- `time.Local` uses explicit `TZ` or `/etc/localtime`, then falls back to UTC.
  It does not yet subscribe to the platform timezone parameter/service; set
  `TZ` before loading the library or use an explicit `time.Location`.
- Netgo DNS is opt-in and does not reproduce OHOS network-id/VPN resolver
  policy. User/app certificate policy is not obtained from the platform
  certificate service; configure application trust explicitly when needed.
- Linux syscall inheritance is not a sandbox guarantee. Process creation,
  sockets, multicast/netlink, procfs, mprotect, executable mappings, signing,
  and application permissions must be verified in the actual target app.
  Phone apps cannot be assumed to run arbitrary executables or a Go compiler.
- AMD64 TLS code-generation smoke tests are not a proof of every asynchronous
  unwind path. No device-level unwind/stress certification is claimed.

## Device test procedure

After platform-required signing and authorized deployment into a suitable
OpenHarmony test environment, run from a writable/executable test directory:

```sh
./hello
./loader ./libgo_hmos_test.so
./runtime.test -test.run 'TestOpenHarmony(RuntimePolicy|SynchronousFault)$'
./runtime_pprof.test -test.run '^TestOpenHarmonyCPUProfileUnsupported$'
```

The loader sets the environment before dlopen, invokes Go from four foreign
pthreads (100 callbacks), and checks goroutines, GC, timers, nil-fault recovery,
and preservation of existing reserved handlers without installing or raising
reserved signals. By default it also repeats interface discovery 100 times.
`--no-network` explicitly omits that last check for restricted host simulations;
it must be reported as omitted and must not be used to claim network support.
Run broader runtime/stdlib tests and a signed HAP with its actual permissions,
DNS/network changes, background/foreground lifecycle, repeated callbacks,
synchronous C/Go faults, and memory-pressure stress before production use.

## Verification status and earlier testing

See [the validation record](../misc/openharmony/VALIDATION.md) for exact local
results, failed/blocked stages, and what has never run. CI's default job checks
host bootstrap, focused tests, API, and cross-compiled Go/assembly. The manual
SDK job is opt-in because it downloads roughly 2.3 GB; neither job is a device
or a commercial NEXT certification.

Earlier evidence, not results of this PR:

- SIG's older port contains c-shared/TLS and platform tests. Its later native
  HiShell work adds fallocate/signing adaptations; full test logs are not
  established by the existence of a patch.
- [star4277 upgrade notes](https://github.com/star4277/ohos-go/blob/19667c7abfc332f848b5bd1f293e4aeb0e9991a9/docs/go-upgrade-guide.md)
  report bootstrap and highlight the pre-allocator GODEBUG crash regression.
- [jgowdy's musl fix](https://github.com/jgowdy/go/commit/1a087d05b5cf9573876b18812d8d5516f16bbe57)
  reports ARM64 Alpine/Ubuntu testing and includes a dlopen fixture. Generic
  Alpine testing alone does not validate OHOS's customized musl or sandbox.
- [FlClash HarmonyOS notes](https://github.com/shenyingjun5/FlClash-HarmonyOS/blob/harmony/docs/harmonyos.md)
  contain device/emulator integration evidence with a pinned different Go
  fork. Application-level evidence is useful but is not a full Go test suite.

Primary platform references:
[NDK scope and signal reservations](https://github.com/openharmony/docs/blob/master/en/application-dev/napi/c-cpp-overview.md),
[OHOS musl extensions](https://github.com/openharmony/docs/blob/master/en/application-dev/reference/native-lib/musl.md),
[SDK release](https://github.com/openharmony/docs/blob/master/zh-cn/release-notes/OpenHarmony-v6.1-release.md),
[Go release downloads](https://go.dev/dl/?mode=json).

Additional ABI review item: the AMD64 assembler currently emits PUSH/CALL/POP
inside the TLS pseudo-instruction's machine-code expansion. Those interior
stack-pointer changes are not represented as separate `Prog.SpAdj`/PCSP
entries. Disabling Go async preemption/CPU profiling reduces exposure but does
not prove asynchronous external-unwinder correctness. This requires an
explicit-instruction lowering and target signal-stack/unwind stress tests
before removing the experimental label.
