#!/usr/bin/env bash
set -euo pipefail

version="${1:-auto}"
output_root="${2:-.tmp/ladybug-native}"

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/.." && pwd)"
if [[ "$output_root" != /* ]]; then
  output_root="${repo_root}/${output_root}"
fi

resolve_latest_version_tag() {
  local cache_path="${output_root}/latest-release.json"
  local today_utc
  today_utc="$(date -u +%F)"
  if [[ -f "$cache_path" ]]; then
    local cached_date cached_tag
    cached_date="$(sed -n 's/.*"checkedDateUtc"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$cache_path" | head -n 1)"
    cached_tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$cache_path" | head -n 1)"
    if [[ "$cached_date" == "$today_utc" && -n "$cached_tag" ]]; then
      printf '%s\n' "$cached_tag"
      return 0
    fi
  fi

  local response tag checked_at
  response="$(curl -fsSL \
    -H 'Accept: application/vnd.github+json' \
    -H 'User-Agent: avmatrix-go-native-bootstrap' \
    https://api.github.com/repos/LadybugDB/ladybug/releases/latest)"
  tag="$(printf '%s\n' "$response" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [[ -z "$tag" ]]; then
    echo "Could not resolve latest LadybugDB release tag from GitHub." >&2
    exit 1
  fi
  checked_at="$(date -u +%FT%TZ)"
  mkdir -p "$output_root"
  printf '{\n  "tag_name": "%s",\n  "checkedDateUtc": "%s",\n  "checkedAtUtc": "%s"\n}\n' "$tag" "$today_utc" "$checked_at" > "$cache_path"
  printf '%s\n' "$tag"
}

if [[ "$version" == "auto" || -z "$version" ]]; then
  version_tag="$(resolve_latest_version_tag)"
elif [[ "$version" == v* ]]; then
  version_tag="$version"
else
  version_tag="v${version}"
fi
version_number="${version_tag#v}"

os="$(uname -s)"
arch="$(uname -m)"
case "${os}:${arch}" in
  Linux:x86_64)
    platform="linux-x86_64"
    asset="liblbug-linux-x86_64.tar.gz"
    primary_lib="liblbug.so"
    ;;
  Linux:aarch64|Linux:arm64)
    platform="linux-aarch64"
    asset="liblbug-linux-aarch64.tar.gz"
    primary_lib="liblbug.so"
    ;;
  Darwin:x86_64)
    platform="osx-x86_64"
    asset="liblbug-osx-x86_64.tar.gz"
    primary_lib="liblbug.dylib"
    ;;
  Darwin:arm64)
    platform="osx-arm64"
    asset="liblbug-osx-arm64.tar.gz"
    primary_lib="liblbug.dylib"
    ;;
  *)
    echo "LadybugDB native runtime script does not support ${os} ${arch}." >&2
    exit 1
    ;;
esac

native_dir="${output_root}/${version_tag}/${platform}"
if [[ -f "${native_dir}/lbug.h" && -f "${native_dir}/${primary_lib}" ]]; then
  printf '%s\n' "$native_dir"
  exit 0
fi

downloads_dir="${output_root}/downloads"
extract_root="${output_root}/extract"
mkdir -p "$downloads_dir" "$extract_root"

archive_path="${downloads_dir}/${asset%.tar.gz}-${version_number}.tar.gz"
if [[ ! -f "$archive_path" ]]; then
  url="https://github.com/LadybugDB/ladybug/releases/download/${version_tag}/${asset}"
  curl -fsSL "$url" -o "$archive_path"
fi

tmp_extract="${extract_root}/${platform}-${version_number}"
case "$(realpath -m "$tmp_extract")" in
  "$(realpath -m "$output_root")"/*) ;;
  *)
    echo "Refusing to write outside native output root: ${tmp_extract}" >&2
    exit 1
    ;;
esac
rm -rf "$tmp_extract"
mkdir -p "$tmp_extract"
tar -xzf "$archive_path" -C "$tmp_extract"

header_path="$(find "$tmp_extract" -name lbug.h -type f | head -n 1)"
if [[ -z "$header_path" ]]; then
  echo "Downloaded LadybugDB archive did not contain lbug.h." >&2
  exit 1
fi
source_dir="$(dirname "$header_path")"

case "$(realpath -m "$native_dir")" in
  "$(realpath -m "$output_root")"/*) ;;
  *)
    echo "Refusing to write outside native output root: ${native_dir}" >&2
    exit 1
    ;;
esac
rm -rf "$native_dir"
mkdir -p "$native_dir"
cp -a "${source_dir}/." "$native_dir/"

if [[ ! -f "${native_dir}/${primary_lib}" ]]; then
  if [[ "$primary_lib" == "liblbug.so" ]]; then
    soname="$(find "$native_dir" -maxdepth 1 -name 'liblbug.so.*' -type f | sort | tail -n 1)"
  else
    soname="$(find "$native_dir" -maxdepth 1 -name 'liblbug.*.dylib' -type f | sort | tail -n 1)"
  fi
  if [[ -z "$soname" ]]; then
    echo "Downloaded LadybugDB archive did not contain ${primary_lib}." >&2
    exit 1
  fi
  ln -s "$(basename "$soname")" "${native_dir}/${primary_lib}"
fi

if [[ ! -f "${native_dir}/lbug.h" || ! -e "${native_dir}/${primary_lib}" ]]; then
  echo "LadybugDB native runtime is incomplete in ${native_dir}." >&2
  exit 1
fi

printf '%s\n' "$native_dir"
