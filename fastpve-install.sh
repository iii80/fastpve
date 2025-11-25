#!/bin/bash

set -e

DOWNLOAD_URL="https://fw0.koolcenter.com/binary/fastpve/FastPVE"
MAX_CACHE_MINS=10
if [ -d "/root" ]; then
  TEMP_DIR="/root"
else
  TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'fastpve')
fi
CACHE_FILE="$TEMP_DIR/FastPVE"
#CACHE_FILE="./FastPVE"

cleanup() {
    rm -f "/tmp/fastpve-install.sh"
}
trap cleanup EXIT

check_local_version() {
    local binary_path=$1
    if [[ ! -f "$binary_path" ]]; then
        return 1
    fi

    echo "Binary exists: $binary_path"
    local last_modified=$(stat -c %Y "$binary_path" 2>/dev/null)
    local current_time=$(date +%s)
    local time_diff=$(( (current_time - last_modified) / 60 ))

    if [[ $time_diff -ge $MAX_CACHE_MINS ]]; then
        return 1
    fi

    if "$binary_path" version &>/dev/null; then
        return 0
    else
        return 1
    fi
}

if check_local_version "$CACHE_FILE"; then
    "$CACHE_FILE" version
else
    if command -v curl &>/dev/null; then
        curl -L -o "$CACHE_FILE" "$DOWNLOAD_URL"
    elif command -v wget &>/dev/null; then
        wget -O "$CACHE_FILE" "$DOWNLOAD_URL"
    else
        exit 1
    fi

    chmod +x "$CACHE_FILE"
fi

"$CACHE_FILE"
