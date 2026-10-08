# OpenHarmony port

This experimental downstream port targets 64-bit OpenHarmony standard systems.
It has not been certified on commercial HarmonyOS NEXT devices. The base is
`release-branch.go1.27`; the official Go release and the port's toolchain
version are distinct.

## Build configuration

The targets are `GOOS=openharmony GOARCH=arm64` and `GOARCH=amd64`. ARM64 is
intended for physical devices; AMD64 is intended for x86_64 systems/emulators.
The `openharmony`, `linux`, and `unix` build tags match. Linux source files are
inherited unless explicitly excluded.

`runtime.GOOS` is `"openharmony"`. The go command's GOOS, GOHOSTOS, native
tool selection, and version output use the same identity. Linux source reuse
is expressed by build constraints; there is no extra public runtime platform
discriminator.

Cgo and external linking are required. Executables default to PIE. Supported
build modes are ordinary executables/PIE, c-shared, c-archive, and Go package
archives. Go plugin/shared modes, no-cgo executables, and race/MSan/ASan are
not supported. Automatic toolchain switching is rejected for OpenHarmony
because official Go downloads do not contain this port.

Bootstrap the host toolchain using `src/make.bash` (or `make.bat` on Windows)
and a supported bootstrap Go version. Set `GOTOOLCHAIN=local`. For target
builds, set `CGO_ENABLED=1` and point `CC` at an OpenHarmony SDK compiler wrapper
that supplies the target triple and sysroot:

- ARM64: `aarch64-linux-ohos`
- AMD64: `x86_64-linux-ohos`
- SDK sysroot: the SDK's `native/sysroot` directory

For example, with `CC` already configured:

```sh
GOOS=openharmony GOARCH=arm64 CGO_ENABLED=1 go build -buildmode=c-shared -o libexample.so ./example
```

The public OpenHarmony SDK and Huawei's commercial HarmonyOS NEXT SDK are
separate distributions. Neither is bundled here. A native Go compiler also
requires a native C compiler, the Go installation layout, platform-required
signing, and a device policy permitting process execution. Producing native
tools does not establish native self-hosting on phone applications.

The downstream version suffix contains `devel` so compiler/assembler/linker
cache identities include their content build IDs. Reusing the unmodified
upstream release identity with a changed toolchain can load incompatible
cached object files.

## Runtime and library behavior

- Both architectures use ELF TLS descriptors for c-shared/c-archive libraries.
  This avoids musl's rejection of initial-exec TLS in a dynamically loaded
  library. No private pthread/TCB offset is used.
- Musl constructors are not assumed to receive argc/argv. Startup uses Go 1.27's
  library initialization path, allocation-free early environment access, and
  procfs arguments/auxv where available. The upstream 48-bit heap-address
  handling is retained instead of assuming every ARM64 device has a 39-bit ABI.
- Signal registration and normal signal masking use libc's signal chain.
  The fork-child reset path retains raw syscalls because libc locks may be
  unsafe after fork.
- The platform reserves signals 1–34 and uses additional signals through 45.
  The port preserves reserved handlers except libc-chained synchronous fault
  handling and SIGPIPE. `os/signal` does not subscribe to or ignore 1–45.
- Asynchronous preemption is disabled rather than taking over SIGURG.
  Cooperative safe points remain; a tight non-cooperative loop can delay
  scheduling or GC. CPU profiling reports unsupported rather than using SIGPROF.
- Libc DNS is preferred for system network-id/resolver integration. Interface
  discovery uses getifaddrs with balanced resource cleanup, including with
  `netgo`. Netgo DNS does not implement the platform's network-id/VPN policy.
- System certificates use `/etc/security/certificates`. Application/user trust
  policy is not obtained from the platform certificate service.
- Timezones support packed `/etc/zoneinfo/tzdata` and the platform's TZif
  directories. `time.Local` uses explicit `TZ` or `/etc/localtime`, then UTC;
  it does not yet track `persist.time.timezone`. Set `TZ` before initialization
  or use an explicit `time.Location`.
- Native linker file growth uses the portable fallback instead of fallocate,
  which may be prohibited by a target sandbox. Applications must provide a
  writable temporary directory and their required OS permissions.

Do not unload a live Go runtime with `dlclose`. Set C environment variables
before the first dlopen; concurrent C setenv/unsetenv during initialization
remains unsafe. Linux syscall compatibility does not establish availability
inside an application sandbox, including process creation, sockets, procfs,
executable memory, signing, and background lifecycle behavior.

TLS resolver calls must preserve the Go frame layout and stack metadata.
Target signal-stack and external-unwinder testing remains necessary; host
code-generation checks alone do not establish device behavior.

## Tests and validation

The port includes build-tag/target/codegen/driver tests, target runtime signal
and fault tests, CPU-profile rejection tests, and timezone parser tests.
`TestOpenHarmonyDynamicTLS` in `cmd/cgo/internal/testcshared` loads a library,
uses 100 callbacks from four foreign pthreads, sets GODEBUG before dlopen to
exercise early initialization, and checks environment, GC, goroutines, timers,
recovered faults, interface discovery, and preservation of existing reserved
handlers. It requires an actual native test environment;
host cross-compilation or a skipped test is not a target runtime pass.

The fixtures in that package's `testdata/openharmony` directory can also be
cross-built for a signed target test. `loader library.so --no-network` explicitly omits
network checks and must not be reported as network validation.

Host bootstrap, focused tests, cross-compilation, SDK linking, and Linux-host
ABI simulations each require results tied to the source revision under test.
They do not establish full target runtime/stdlib coverage, physical-device or
emulator execution, signed HAP/N-API integration, or native bootstrap. Build automation and detailed per-run validation
records are maintained separately in `ZxillyFork/go-hmos-build`; the installer
is a separate project. They are not part of this Go source tree.

## References

The port adapts the BSD-licensed OpenHarmony SIG work and
[star4277's Go 1.26.5 forward-port](https://github.com/star4277/ohos-go/tree/19667c7abfc332f848b5bd1f293e4aeb0e9991a9),
excluding unrelated fixture deletion, mode changes, and unsupported shared/plugin
machinery. Existing Go copyright, LICENSE and PATENTS notices are preserved.

Primary platform references:
[NDK scope and signals](https://github.com/openharmony/docs/blob/master/en/application-dev/napi/c-cpp-overview.md),
[customized musl](https://github.com/openharmony/docs/blob/master/en/application-dev/reference/native-lib/musl.md),
[SDK release](https://github.com/openharmony/docs/blob/master/zh-cn/release-notes/OpenHarmony-v6.1-release.md).
Earlier application and Alpine tests are useful evidence, but do not establish
compatibility of this revision with every OpenHarmony or NEXT environment.
