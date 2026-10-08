#!/usr/bin/env bash
# Copyright 2026 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
set -euo pipefail
# Explicit opt-in helper for the official public OpenHarmony SDK. It does not
# obtain Huawei's commercial HarmonyOS NEXT SDK, accept click-through terms,
# or treat an unavailable download as a passing test.
: "${SDK_DOWNLOAD_DIR:?Set SDK_DOWNLOAD_DIR to a scratch directory}"
mkdir -p "$SDK_DOWNLOAD_DIR"
cd "$SDK_DOWNLOAD_DIR"
base=https://repo.huaweicloud.com/openharmony/os/6.1-Release
archive=ohos-sdk-windows_linux-public.tar.gz
curl --fail --location --retry 2 "$base/$archive.sha256" -o "$archive.sha256"
digest=$(grep -Eo '[0-9a-fA-F]{64}' "$archive.sha256" | head -1)
[[ ${#digest} == 64 ]] || { echo 'official checksum unavailable or invalid' >&2; exit 1; }
curl --fail --location --retry 2 "$base/$archive" -o "$archive"
printf '%s  %s\n' "$digest" "$archive" | sha256sum --check -
# Extract only the Linux native package, not the Windows tools or emulator.
member=$(tar -tzf "$archive" | grep -E '(^|/)linux/native.*\.zip$' | head -1)
[[ -n "$member" ]] || { echo 'Linux native SDK zip not found in verified archive' >&2; exit 1; }
tar -xzf "$archive" "$member"
mkdir -p extracted
unzip -q "$member" -d extracted
native=$(find "$PWD/extracted" -type f -path '*/llvm/bin/clang' -print -quit)
[[ -n "$native" ]] || { echo 'SDK compiler missing after extraction' >&2; exit 1; }
printf '%s\n' "${native%/llvm/bin/clang}" > native-path.txt
printf 'Verified SDK extracted. Review its licenses before use. Native directory: %s\n' "$(cat native-path.txt)"
