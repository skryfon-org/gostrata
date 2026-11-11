#!/usr/bin/env sh
# gostrata installer for Linux and macOS.
#   curl -fsSL https://raw.githubusercontent.com/skryfon-org/gostrata/main/install.sh | sh
# Optional: VERSION=v1.1.1  INSTALL_DIR=/usr/local/bin
set -eu

REPO="skryfon-org/gostrata"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in linux|darwin) ;; *) echo "Unsupported OS: $os (use install.ps1 on Windows)" >&2; exit 1 ;; esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

version="${VERSION:-}"
if [ -z "$version" ]; then
  version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
fi
[ -n "$version" ] || { echo "Could not determine latest version" >&2; exit 1; }
num="${version#v}"

archive="gostrata_${num}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT

echo "Downloading gostrata $version ($os/$arch)..."
curl -fsSL "$base/$archive" -o "$tmp/$archive"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

expected=$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$tmp/$archive" | awk '{print $1}')
else actual=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}'); fi
[ -n "$expected" ] && [ "$expected" = "$actual" ] || { echo "Checksum verification failed" >&2; exit 1; }

tar -xzf "$tmp/$archive" -C "$tmp" gostrata

dir="${INSTALL_DIR:-}"
if [ -z "$dir" ]; then
  if [ -w /usr/local/bin ]; then dir=/usr/local/bin; else dir="$HOME/.local/bin"; fi
fi
mkdir -p "$dir"
install -m 0755 "$tmp/gostrata" "$dir/gostrata"

echo "Installed gostrata to $dir/gostrata"
case ":$PATH:" in *":$dir:"*) ;; *) echo "Add $dir to your PATH:  export PATH=\"\$PATH:$dir\"" ;; esac
"$dir/gostrata" version 2>/dev/null || true
