#!/usr/bin/env bash
# HandSSHake installer
#
#   curl -fsSL https://raw.githubusercontent.com/orchid0610/handsshake/main/install.sh | bash
#
# Options (when piping, put them after "bash -s --"):
#   --version vX.Y.Z   install that release instead of the latest
#   --uninstall        remove HandSSHake (your hosts and settings are kept)
#   --dry-run          show what would happen and change nothing
#   -h, --help         show this help
#
# For testing: HANDSSHAKE_METHOD=tarball forces the tarball install on Debian/Ubuntu.
set -euo pipefail

REPO="orchid0610/handsshake"
BASE="${HANDSSHAKE_BASE_URL:-https://github.com/${REPO}/releases}"

VERSION="latest"
UNINSTALL=0
DRY=0
OS=""
ARCH=""
TMP=""

say()  { printf '%s\n' "$*"; }
step() { printf '==> %s\n' "$*"; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
HandSSHake installer

Usage: install.sh [--version vX.Y.Z] [--uninstall] [--dry-run]

  --version vX.Y.Z   install that release instead of the latest
  --uninstall        remove HandSSHake (your hosts and settings are kept)
  --dry-run          show what would happen and change nothing
  -h, --help         show this help

One-liner:
  curl -fsSL https://raw.githubusercontent.com/orchid0610/handsshake/main/install.sh | bash
With options:
  curl -fsSL https://raw.githubusercontent.com/orchid0610/handsshake/main/install.sh | bash -s -- --version v1.3.0
EOF
}

# run CMD...: execute it, or only print it with --dry-run.
run() {
  if [ "$DRY" = 1 ]; then
    say "[dry-run] $*"
  else
    "$@"
  fi
}

as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    die "this step needs root: run the installer as root, or install sudo."
  fi
}

detect() {
  OS="$(uname -s)"
  case "$(uname -m)" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) die "unsupported CPU: $(uname -m)" ;;
  esac
}

asset_url() {
  if [ "$VERSION" = "latest" ]; then
    echo "${BASE}/latest/download/$1"
  else
    echo "${BASE}/download/${VERSION}/$1"
  fi
}

fetch() { # fetch URL DEST
  curl -fsSL --proto-redir '=https' --retry 3 --connect-timeout 15 -o "$2" "$1" \
    || die "could not download $1"
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    die "need sha256sum or shasum to verify the download"
  fi
}

# verify FILE: compare it with the hash listed in SHA256SUMS.txt.
verify() {
  local file name expected actual
  file="$1"
  name="$(basename "$file")"
  expected="$(awk -v n="$name" '$2 == n || $2 == "*" n { print $1 }' "$TMP/SHA256SUMS.txt")"
  [ -n "$expected" ] || die "$name is not listed in SHA256SUMS.txt"
  actual="$(sha256_of "$file")"
  [ "$expected" = "$actual" ] || die "checksum mismatch for $name (expected $expected, got $actual). Nothing was installed."
}

# download_verified ASSET: download it into $TMP and check its checksum.
download_verified() {
  if [ "$DRY" = 1 ]; then
    say "[dry-run] would download and verify $(asset_url "$1")"
    return
  fi
  step "Downloading $1"
  fetch "$(asset_url "$1")" "$TMP/$1"
  if [ ! -f "$TMP/SHA256SUMS.txt" ]; then
    fetch "$(asset_url SHA256SUMS.txt)" "$TMP/SHA256SUMS.txt"
  fi
  verify "$TMP/$1"
  say "    checksum OK"
}

is_debian_like() {
  [ -r /etc/os-release ] && command -v apt-get >/dev/null 2>&1 || return 1
  ( . /etc/os-release; case " ${ID:-} ${ID_LIKE:-} " in *" debian "*|*" ubuntu "*) exit 0 ;; *) exit 1 ;; esac )
}

install_linux() {
  [ "$ARCH" = "amd64" ] || die "HandSSHake is only built for x86_64 Linux so far (this machine is $(uname -m))."

  if [ "${HANDSSHAKE_METHOD:-}" != "tarball" ] && is_debian_like; then
    download_verified handsshake-linux-amd64.deb
    step "Installing the package (needs sudo)"
    run as_root apt-get install -y "$TMP/handsshake-linux-amd64.deb"
    say ""
    say "HandSSHake is installed. Start it from your application menu, or run: handsshake"
    return
  fi

  # Any other distro: unpack the binary into ~/.local/bin.
  download_verified handsshake-linux-amd64.tar.gz
  local dir="$HOME/.local/bin"
  step "Installing to $dir"
  run mkdir -p "$TMP/unpacked" "$dir"
  run tar -xzf "$TMP/handsshake-linux-amd64.tar.gz" -C "$TMP/unpacked"
  run install -m 755 "$TMP/unpacked/handsshake" "$dir/handsshake"
  say ""
  say "HandSSHake is installed at $dir/handsshake"
  case ":$PATH:" in
    *":$dir:"*) ;;
    *) say "Note: $dir is not on your PATH yet. Add this to your shell profile:"
       say "  export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
  esac
  say "It needs GTK 3 and WebKitGTK 4.1 (names vary: e.g. Fedora 'gtk3 webkit2gtk4.1', Arch 'gtk3 webkit2gtk-4.1')."
}

install_macos() {
  # The release is a universal binary, so amd64 and arm64 are both fine here.
  command -v ditto >/dev/null 2>&1 || die "this installer needs 'ditto' (included with macOS)"

  download_verified handsshake-macos-universal.zip

  # Prefer /Applications (writable for admin users, drwxrwxr-x root:admin).
  # Fall back to ~/Applications for non-admin accounts instead of demanding sudo
  # for a per-user GUI app.
  local dest_dir
  if [ -w "/Applications" ]; then
    dest_dir="/Applications"
  else
    dest_dir="$HOME/Applications"
  fi
  local app_dest="$dest_dir/HandSSHake.app"

  step "Installing to $app_dest"
  run mkdir -p "$TMP/unpacked" "$dest_dir"

  # ditto (not unzip) is the right tool for .app bundles: it preserves
  # resource forks, metadata and the bundle layout codesign expects.
  # The zip was made with `ditto -c -k --keepParent`, so the unpacked
  # root contains HandSSHake.app directly.
  run ditto -x -k "$TMP/handsshake-macos-universal.zip" "$TMP/unpacked"

  if [ "$DRY" != 1 ] && [ ! -d "$TMP/unpacked/HandSSHake.app" ]; then
    die "the download did not contain HandSSHake.app — the release asset may be corrupt."
  fi

  # Replace any previous copy.
  if [ "$DRY" = 1 ]; then
    say "[dry-run] rm -rf $app_dest"
  elif [ -d "$app_dest" ]; then
    rm -rf "$app_dest"
  fi
  run ditto "$TMP/unpacked/HandSSHake.app" "$app_dest"

  # The app is only ad-hoc signed (see the release workflow), so a copy
  # downloaded by a browser/curl carries quarantine and Gatekeeper would
  # block the first launch. Strip it from our own verified download.
  if [ "$DRY" = 1 ]; then
    say "[dry-run] xattr -dr com.apple.quarantine $app_dest"
  else
    xattr -dr com.apple.quarantine "$app_dest" 2>/dev/null || true
  fi

  say ""
  say "HandSSHake is installed at $app_dest"
  say "Open it from Launchpad / $dest_dir, or run: open \"$app_dest\""
  if [ "$dest_dir" != "/Applications" ]; then
    say "Note: installed per-user because /Applications is not writable for this account."
  fi
}

uninstall() {
  local removed=0
  case "$OS" in
    Linux)
      if command -v dpkg >/dev/null 2>&1 && dpkg -s handsshake >/dev/null 2>&1; then
        step "Removing the package (needs sudo)"
        run as_root apt-get remove -y handsshake
        removed=1
      fi
      if [ -f "$HOME/.local/bin/handsshake" ]; then
        step "Removing $HOME/.local/bin/handsshake"
        run rm -f "$HOME/.local/bin/handsshake"
        removed=1
      fi
      ;;
    Darwin)
      local dir
      for dir in "/Applications" "$HOME/Applications"; do
        if [ -d "$dir/HandSSHake.app" ]; then
          step "Removing $dir/HandSSHake.app"
          run rm -rf "$dir/HandSSHake.app"
          removed=1
        fi
      done
      ;;
    *) die "uninstall is not supported on $OS yet" ;;
  esac
  [ "$removed" = 1 ] || say "HandSSHake does not seem to be installed."
  say "Your hosts, keys and settings were left alone. To remove them too, delete:"
  case "$OS" in
    Darwin) say "  $HOME/Library/Application Support/handsshake" ;;
    *)      say "  $HOME/.config/handsshake" ;;
  esac
}

main() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --version)
        [ $# -ge 2 ] || die "--version needs a value, like v1.3.0"
        VERSION="$2"
        shift 2 ;;
      --uninstall) UNINSTALL=1; shift ;;
      --dry-run)   DRY=1; shift ;;
      -h|--help)   usage; exit 0 ;;
      *) die "unknown option: $1 (try --help)" ;;
    esac
  done
  case "$VERSION" in latest|v*) ;; *) VERSION="v$VERSION" ;; esac

  for c in curl tar awk mktemp; do
    command -v "$c" >/dev/null 2>&1 || die "this installer needs '$c'"
  done
  detect
  TMP="$(mktemp -d)"
  chmod 755 "$TMP" # so apt's sandbox user can read the downloaded package
  trap 'rm -rf "$TMP"' EXIT

  if [ "$UNINSTALL" = 1 ]; then
    uninstall
    return
  fi

  step "HandSSHake ${VERSION} for ${OS} ${ARCH}"
  case "$OS" in
    Linux)  install_linux ;;
    Darwin) install_macos ;;
    *)      die "unsupported system: $OS" ;;
  esac
}

# Everything runs from here, so a half-downloaded script can't run half a program.
main "$@"
