#!/usr/bin/env sh
set -eu

# cifuse installer
#
# Downloads a cifuse binary release from GitHub, verifies its SHA-256 checksum,
# and installs it to $HOME/.local/bin (or $CIFUSE_INSTALL_DIR).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Polymerthcedric/cifuse/main/scripts/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/Polymerthcedric/cifuse/main/scripts/install.sh | bash -s -- -v v0.2.0
#   curl -fsSL https://raw.githubusercontent.com/Polymerthcedric/cifuse/main/scripts/install.sh | CIFUSE_INSTALL_DIR=$HOME/.local/bin bash

repo="Polymerthcedric/cifuse"
version="${CIFUSE_VERSION:-latest}"
install_dir="${CIFUSE_INSTALL_DIR:-$HOME/.local/bin}"

while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version)
      version="$2"
      shift 2
      ;;
    --install-dir)
      install_dir="$2"
      shift 2
      ;;
    *)
      echo "install.sh: unknown option: $1" >&2
      exit 2
      ;;
  esac
done

if [ "$version" = "latest" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  if [ -z "$version" ]; then
    echo "install.sh: could not resolve the latest release" >&2
    exit 1
  fi
fi

os="$(uname -s)"
case "$os" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *) echo "install.sh: unsupported OS: $os" >&2; exit 1 ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "install.sh: unsupported architecture: $arch" >&2; exit 1 ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$1" | awk '{print $1}'; }
else
  sha256() { shasum -a 256 "$1" | awk '{print $1}'; }
fi

archive="cifuse_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$version"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "$base_url/checksums.txt" -o "$tmp/checksums.txt"
curl -fsSL "$base_url/$archive" -o "$tmp/$archive"

expected="$(awk -v a="$archive" '$2 == a {print $1}' "$tmp/checksums.txt" | head -n1)"
if [ -z "$expected" ]; then
  echo "install.sh: no checksum found for $archive" >&2
  exit 1
fi
actual="$(sha256 "$tmp/$archive")"
if [ "$actual" != "$expected" ]; then
  echo "install.sh: checksum mismatch for $archive" >&2
  echo "expected $expected" >&2
  echo "got      $actual" >&2
  exit 1
fi

mkdir -p "$install_dir"
tar -xzf "$tmp/$archive" -C "$tmp"
install -m 0755 "$tmp/cifuse" "$install_dir/cifuse"

echo "installed cifuse $version to $install_dir/cifuse"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "note: $install_dir is not on your PATH" >&2 ;;
esac