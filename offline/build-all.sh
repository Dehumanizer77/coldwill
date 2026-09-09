#!/usr/bin/env bash
# build-all.sh - builds inh-offline for every platform an heir might use.
# Output goes to dist/ (gitignored). Run it from the offline/ directory, or
# through release.sh, which builds and then publishes.
#
# Go cross-compiles on its own, no extra toolchain needed: the tool is pure Go
# with no cgo. Only the binary for this machine can be tested here; the others
# have to be tried on the target system, which belongs in the yearly checkup.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

# Go is often installed outside PATH (a tarball unpacked into ~/.local/go, say),
# so look in the usual places before giving up. Override with GO=/path/to/go.
GO="${GO:-$(command -v go || true)}"
if [ -z "$GO" ]; then
  for c in "$HOME/.local/go/bin/go" /usr/local/go/bin/go /usr/lib/go/bin/go; do
    [ -x "$c" ] && { GO="$c"; break; }
  done
fi
if [ -z "$GO" ]; then
  echo "go not found. Install it from https://go.dev/dl/ (unpack the tarball" >&2
  echo "into ~/.local/go), or point this script at it: GO=/path/to/go $0" >&2
  exit 1
fi
echo "Building with $("$GO" version), $GO"
echo

OUT=dist
rm -rf "$OUT" && mkdir -p "$OUT"

# GOOS GOARCH extension description
TARGETS=(
  "linux   amd64 ''    Linux (ordinary computer)"
  "windows amd64 .exe  Windows"
  "darwin  arm64 ''    macOS (Apple Silicon, M1 and newer)"
  "darwin  amd64 ''    macOS (older, Intel)"
)

for t in "${TARGETS[@]}"; do
  read -r goos goarch ext _ <<<"$t"
  [ "$ext" = "''" ] && ext=""
  name="inh-offline-$goos-$goarch$ext"
  GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 \
    "$GO" build -trimpath -ldflags="-s -w" -o "$OUT/$name" .
  printf '  %-34s %6s KB\n' "$name" "$(( $(stat -c%s "$OUT/$name") / 1024 ))"
done

( cd "$OUT" && sha256sum * > SHA256SUMS )
echo
echo "Done, in $(pwd)/$OUT."
echo "release.sh uploads these; for a USB stick, copy the whole directory."
