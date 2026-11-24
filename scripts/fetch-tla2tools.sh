#!/usr/bin/env bash
# Download the pinned tla2tools.jar into the Edda cache and print its path.
#
#   scripts/fetch-tla2tools.sh          use the pin in scripts/tla2tools.txt
#   scripts/fetch-tla2tools.sh <tag>    pin a new release tag, for example v1.8.0
#
# Usage: export EDDA_TLA2TOOLS="$(scripts/fetch-tla2tools.sh)"
set -euo pipefail

pin="$(dirname "$0")/tla2tools.txt"
dest="${XDG_CACHE_HOME:-$HOME/.cache}/edda/tla2tools.jar"

sha256() {
  if command -v sha256sum >/dev/null; then sha256sum "$1"; else shasum -a 256 "$1"; fi | cut -d' ' -f1
}

if [[ $# -eq 1 ]]; then
  # The expected digest comes from GitHub, not from the download itself.
  VERSION="$1"
  api="repos/tlaplus/tlaplus/releases/tags/$VERSION"
  # gh is authenticated. Anonymous curl has a limit of 60 requests each hour.
  if command -v gh >/dev/null; then
    release="$(gh api "$api")"
  else
    release="$(curl -fsSL "https://api.github.com/$api")"
  fi
  digest="$(jq -r '.assets[] | select(.name == "tla2tools.jar") | .digest' <<<"$release")"
  URL="$(jq -r '.assets[] | select(.name == "tla2tools.jar") | .browser_download_url' <<<"$release")"
  if [[ "$digest" != sha256:* ]]; then
    echo "error: GitHub gives no SHA-256 digest for tla2tools.jar in $VERSION. Select a different tag." >&2
    exit 1
  fi
  SHA256="${digest#sha256:}"
else
  # shellcheck source=tla2tools.txt
  . "$pin"
fi

if [[ -f "$dest" && "$(sha256 "$dest")" == "$SHA256" && $# -eq 0 ]]; then
  echo "$dest"
  exit 0
fi

mkdir -p "$(dirname "$dest")"
tmp="$(mktemp "$dest.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
echo "The script downloads tla2tools.jar $VERSION." >&2
curl -fsSL -o "$tmp" "$URL"

actual="$(sha256 "$tmp")"
if [[ "$actual" != "$SHA256" ]]; then
  echo "error: SHA-256 of tla2tools.jar $VERSION is not correct." >&2
  echo "  expected: $SHA256" >&2
  echo "  actual:   $actual" >&2
  echo "The TLA+ team can build a prerelease tag again. To pin the new build, run: $0 $VERSION" >&2
  exit 1
fi
mv "$tmp" "$dest"

if [[ $# -eq 1 ]]; then
  sed -i.bak -e "s|^VERSION=.*|VERSION=$VERSION|" -e "s|^SHA256=.*|SHA256=$SHA256|" -e "s|^URL=.*|URL=$URL|" "$pin"
  rm -f "$pin.bak"
  echo "The script pinned $VERSION in $pin." >&2
fi
echo "$dest"
