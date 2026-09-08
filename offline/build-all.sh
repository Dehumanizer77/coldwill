#!/usr/bin/env bash
# Vybuildí inh-offline pre všetky platformy, na ktorých ho môže dedič spúšťať.
# Výstup ide do dist/ (gitignorované). Spúšťa sa z adresára offline/.
#
# Go kríž-kompiluje samo, netreba žiadny ďalší toolchain: nástroj je čisté Go
# bez cgo. Otestovať sa tu dá len tá binárka pre tento stroj, ostatné treba
# vyskúšať na cieľovom systéme (patrí to do ročnej údržby).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

OUT=dist
rm -rf "$OUT" && mkdir -p "$OUT"

# GOOS GOARCH prípona popis
TARGETS=(
  "linux   amd64 ''    Linux (bežný počítač)"
  "linux   arm64 ''    Linux (ARM, napr. Raspberry Pi)"
  "windows amd64 .exe  Windows"
  "darwin  arm64 ''    macOS (Apple Silicon, M1 a novšie)"
  "darwin  amd64 ''    macOS (staršie, Intel)"
)

for t in "${TARGETS[@]}"; do
  read -r goos goarch ext _ <<<"$t"
  [ "$ext" = "''" ] && ext=""
  name="inh-offline-$goos-$goarch$ext"
  GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "$OUT/$name" .
  printf '  %-34s %6s KB\n' "$name" "$(( $(stat -c%s "$OUT/$name") / 1024 ))"
done

( cd "$OUT" && sha256sum * > SHA256SUMS )
echo
echo "Hotovo v $(pwd)/$OUT — na USB kľúč skopíruj celý obsah."
