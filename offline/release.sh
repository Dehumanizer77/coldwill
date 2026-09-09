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
if gh release view "$TAG" >/dev/null 2>&1; then
  gh release upload "$TAG" dist/* --clobber
else
  gh release create "$TAG" dist/* \
    --title "inh-offline $TAG" \
    --notes "Offline setup and recovery tool.

Download the file for your computer:

| Computer | File |
|---|---|
| Windows | \`inh-offline-windows-amd64.exe\` |
| Mac (2020 onwards) | \`inh-offline-darwin-arm64\` |
| Mac (older, Intel) | \`inh-offline-darwin-amd64\` |
| Linux | \`inh-offline-linux-amd64\` |

\`SHA256SUMS\` lists the checksums. The binaries are unsigned, so Windows and
macOS will warn about an unidentified developer; the runbook explains how to get
past that.

Built locally from this tag, not by CI."
fi

# Print the address the runbook actually sends people to, read from the repo
# this was published into rather than hardcoded to one account.
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
echo
echo "Done. The runbook's address now serves this build:"
echo "  https://github.com/${REPO:-<username>/<repo>}/releases/latest"
