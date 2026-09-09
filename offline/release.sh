#!/usr/bin/env bash
# release.sh - builds the binaries and publishes them as a GitHub release.
#
#   ./release.sh              # tag from today's date
#   ./release.sh v1.0.0       # explicit tag
#
# The binaries are built HERE, on your machine, and only uploaded. There is no
# CI build on purpose: a release pipeline would mean trusting somebody else's
# runner with the binary that reassembles the key file.
#
# Needs the gh CLI, authenticated (gh auth login, or GH_TOKEN in the
# environment). The runbook sends the heir to the /releases/latest page, so
# publishing a release is what keeps that address working.
#
# Re-running it on an existing tag makes that release mirror dist/: the title
# and notes are rewritten, and any asset this build no longer produces is
# DELETED from the release. That is deliberate. Renaming the tool once left
# both the old and the new binaries sitting in the same release, with notes
# naming the ones the runbook no longer mentions.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

TAG="${1:-tool-$(date +%Y-%m-%d)}"

command -v gh >/dev/null || { echo "gh is not installed" >&2; exit 1; }
gh auth status >/dev/null 2>&1 || {
  echo "gh is not authenticated: run 'gh auth login', or set GH_TOKEN" >&2
  exit 1
}

./build-all.sh

echo
echo "Publishing $TAG"

# The table is built from what is actually in dist/, so a renamed binary cannot
# leave the notes pointing at a filename nobody will find on the page.
label() {
  case "$1" in
    *windows*)      echo "Windows" ;;
    *darwin-arm64*) echo "Mac (2020 onwards)" ;;
    *darwin-amd64*) echo "Mac (older, Intel)" ;;
    *linux*)        echo "Linux" ;;
    *)              echo "Other" ;;
  esac
}
TABLE=""
for pat in '*windows*' '*darwin-arm64*' '*darwin-amd64*' '*linux*'; do
  for f in dist/$pat; do
    [ -e "$f" ] || continue
    b=$(basename "$f")
    TABLE="$TABLE| $(label "$b") | \`$b\` |
"
  done
done

NOTES="Offline setup and recovery tool.

Download the file for your computer:

| Computer | File |
|---|---|
${TABLE}
\`SHA256SUMS\` lists the checksums. The binaries are unsigned, so Windows and
macOS will warn about an unidentified developer; the runbook explains how to get
past that.

Built locally from this tag, not by CI."

if gh release view "$TAG" >/dev/null 2>&1; then
  # Drop whatever this build no longer produces, or an old name stays on the
  # page next to the new one and the heir has to guess.
  for old in $(gh release view "$TAG" --json assets -q '.assets[].name'); do
    if [ ! -e "dist/$old" ]; then
      echo "  removing stale asset $old"
      gh release delete-asset "$TAG" "$old" --yes
    fi
  done
  gh release upload "$TAG" dist/* --clobber
  gh release edit "$TAG" --title "coldwill $TAG" --notes "$NOTES"
else
  gh release create "$TAG" dist/* --title "coldwill $TAG" --notes "$NOTES"
fi

# Print the address the runbook actually sends people to, read from the repo
# this was published into rather than hardcoded to one account.
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
echo
echo "Done. The runbook's address now serves this build:"
echo "  https://github.com/${REPO:-<username>/<repo>}/releases/latest"
