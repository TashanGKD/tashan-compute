#!/bin/sh
set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
release_file="$script_dir/../release.json"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  install-cli.sh --check' \
    '  install-cli.sh --install [--version <semver>]' \
    '' \
    'Installs tcompute in the current user account without sudo.' \
    'With no arguments this command only prints help.'
}

fail() {
  printf 'install-cli: %s\n' "$1" >&2
  exit 1
}

json_string() {
  key=$1
  value=$(sed -n "s/^[[:space:]]*\"$key\": \"\([^\"]*\)\",\{0,1\}$/\1/p" "$release_file" | head -n 1)
  [ -n "$value" ] || fail "missing $key in release metadata"
  printf '%s\n' "$value"
}

is_semver() {
  printf '%s\n' "$1" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+(\.[0-9A-Za-z]+)*)?$'
}

detect_platform() {
  case "$(uname -s)" in
    Darwin) operating_system=darwin ;;
    Linux) operating_system=linux ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
  esac
  case "$(uname -m)" in
    arm64 | aarch64) architecture=arm64 ;;
    x86_64 | amd64) architecture=x64 ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
  printf '%s-%s\n' "$operating_system" "$architecture"
}

hash_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    fail 'neither shasum nor sha256sum is available'
  fi
}

replace_symlink() {
  link_value=$1
  link_path=$2
  next_link="$link_path.next-$$"
  ln -s "$link_value" "$next_link"
  case "$(uname -s)" in
    Linux) mv -Tf "$next_link" "$link_path" ;;
    Darwin) mv -fh "$next_link" "$link_path" ;;
    *) rm -f -- "$next_link"; fail 'unsupported operating system for activation' ;;
  esac
}

mode=help
version=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --check | --install)
      [ "$mode" = help ] || fail 'choose exactly one mode'
      mode=${1#--}
      shift
      ;;
    --version)
      [ "$#" -ge 2 ] || fail '--version requires a value'
      version=$2
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *) fail "unknown argument: $1" ;;
  esac
done

if [ "$mode" = help ]; then
  [ -z "$version" ] || fail '--version requires --install'
  usage
  exit 0
fi

pinned_version=$(json_string version)
repository=$(json_string repository)
version=${version:-$pinned_version}
is_semver "$version" || fail 'version must be semver'

if [ "${TCOMPUTE_INSTALL_TESTING:-}" = 1 ]; then
  platform=${TCOMPUTE_INSTALL_PLATFORM:-$(detect_platform)}
  release_base_url=${TCOMPUTE_RELEASE_BASE_URL:-"https://github.com/$repository/releases/download/v$version"}
  curl_options='-fL'
else
  platform=$(detect_platform)
  release_base_url="https://compute.tashan.chat/cli/v$version"
  curl_options='-fL --proto =https --tlsv1.2'
fi

case "$platform" in
  darwin-arm64 | darwin-x64 | linux-x64) ;;
  *) fail "unsupported platform: $platform" ;;
esac

data_home=${XDG_DATA_HOME:-"$HOME/.local/share"}
install_root="$data_home/tcompute"
bin_directory=${TCOMPUTE_BIN_DIR:-"$HOME/.local/bin"}
case "$install_root:$bin_directory" in
  /*:/*) ;;
  *) fail 'install and bin directories must be absolute paths' ;;
esac
[ "$install_root" != /tcompute ] || fail 'refusing unsafe install root'

target="$bin_directory/tcompute"
managed_target="$install_root/current/bin/tcompute"
managed_coder="$install_root/current/bin/coder"
if [ -e "$target" ] || [ -L "$target" ]; then
  if [ ! -L "$target" ] || [ "$(readlink "$target")" != "$managed_target" ]; then
    fail "refusing to replace unmanaged tcompute at $target"
  fi
fi

if [ "$mode" = check ]; then
  [ -L "$target" ] || fail 'tcompute is not installed by this installer'
  installed_version=$($target --version 2>/dev/null) || fail 'installed tcompute failed its version check'
  [ -x "$managed_coder" ] || fail 'bundled Coder CLI is missing'
  "$managed_coder" version >/dev/null 2>&1 || fail 'bundled Coder CLI failed its version check'
  printf 'tcompute %s is installed at %s\n' "$installed_version" "$target"
  exit 0
fi

if [ -L "$target" ]; then
  installed_version=$($target --version 2>/dev/null || true)
  if [ "$installed_version" = "$version" ]; then
    printf 'tcompute %s is already installed at %s\n' "$version" "$target"
    exit 0
  fi
fi

asset="tcompute-v$version-$platform.tar.gz"
top_level=${asset%.tar.gz}
temporary_root=
staging_root=
cleanup() {
  [ -z "$temporary_root" ] || [ ! -d "$temporary_root" ] || rm -rf -- "$temporary_root"
  [ -z "$staging_root" ] || [ ! -d "$staging_root" ] || rm -rf -- "$staging_root"
}
trap cleanup EXIT HUP INT TERM

temporary_root=$(mktemp -d "${TMPDIR:-/tmp}/tcompute-install.XXXXXX")
archive="$temporary_root/$asset"
checksums="$temporary_root/SHA256SUMS"

# shellcheck disable=SC2086
curl $curl_options -o "$checksums" "$release_base_url/SHA256SUMS" >/dev/null 2>&1 || fail 'failed to download SHA256SUMS'
# shellcheck disable=SC2086
curl $curl_options -o "$archive" "$release_base_url/$asset" >/dev/null 2>&1 || fail "failed to download $asset"

expected_hash=$(awk -v file="$asset" '$2 == file { count += 1; hash = $1 } END { if (count == 1) print hash; else exit 1 }' "$checksums") || fail "checksum entry not found for $asset"
printf '%s\n' "$expected_hash" | grep -Eq '^[0-9a-f]{64}$' || fail "invalid checksum for $asset"
[ "$(hash_file "$archive")" = "$expected_hash" ] || fail "checksum verification failed for $asset"

if ! tar -tvzf "$archive" | awk '$1 !~ /^[-d]/ { bad = 1 } END { exit bad }'; then
  fail 'archive links are not allowed'
fi
actual_entries="$temporary_root/actual-entries"
expected_entries="$temporary_root/expected-entries"
tar -tzf "$archive" | LC_ALL=C sort >"$actual_entries"
printf '%s\n' "$top_level/" "$top_level/VERSION" "$top_level/bin/" "$top_level/bin/tcompute" "$top_level/bin/coder" | LC_ALL=C sort >"$expected_entries"
cmp -s "$actual_entries" "$expected_entries" || fail 'invalid archive layout'

mkdir -p "$install_root/versions" "$bin_directory"
version_directory="$install_root/versions/$version"
[ ! -e "$version_directory" ] || fail "version directory already exists but is not active: $version"
staging_root="$install_root/.staging-$version-$$"
mkdir "$staging_root"
tar -xzf "$archive" -C "$staging_root"
candidate="$staging_root/$top_level"
[ -x "$candidate/bin/tcompute" ] || fail 'invalid archive layout'
[ -x "$candidate/bin/coder" ] || fail 'bundled Coder CLI is missing'
[ "$(cat "$candidate/VERSION")" = "$version" ] || fail 'archive version mismatch'
[ "$($candidate/bin/tcompute --version 2>/dev/null || true)" = "$version" ] || fail 'installed CLI smoke test failed'
"$candidate/bin/coder" version >/dev/null 2>&1 || fail 'bundled Coder CLI failed its version check'

mv "$candidate" "$version_directory"
rm -rf -- "$staging_root"
staging_root=
old_current=
if [ -L "$install_root/current" ]; then
  old_current=$(readlink "$install_root/current")
elif [ -e "$install_root/current" ]; then
  fail 'refusing to replace unmanaged current path'
fi
replace_symlink "versions/$version" "$install_root/current"

if [ ! -L "$target" ]; then
  next_target="$bin_directory/.tcompute-$$"
  ln -s "$managed_target" "$next_target"
  mv "$next_target" "$target"
fi

if [ "$($target --version 2>&1 || true)" != "$version" ]; then
  if [ -n "$old_current" ]; then
    replace_symlink "$old_current" "$install_root/current"
  else
    rm -f -- "$install_root/current" "$target"
  fi
  fail 'installed CLI smoke test failed after activation'
fi

printf 'installed tcompute %s at %s\n' "$version" "$target"
