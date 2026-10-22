#!/bin/sh
# Explicit optional download; never runs during application startup.
set -eu
URL=${1:?usage: download-model.sh HTTPS_QWEN_URL DEST_GGUF EXPECTED_SHA256}
DEST=${2:?destination GGUF required}
SHA=${3:?expected SHA256 from publisher LFS metadata required}
case "$URL" in https://huggingface.co/Qwen/*) ;; *) echo 'only official Qwen HTTPS model URLs supported' >&2;exit 1;; esac
case "$SHA" in *[!0-9a-fA-F]*|'') echo 'invalid SHA256' >&2;exit 1;; esac
test "${#SHA}" = 64
test ! -e "$DEST"
mkdir -p "$(dirname "$DEST")"
PART=$(mktemp "$(dirname "$DEST")/.model.XXXXXX")
trap 'rm -f "$PART"' EXIT HUP INT TERM
curl --proto '=https' --proto-redir '=https' --max-time 1800 --max-filesize 2147483648 -fL "$URL" -o "$PART"
printf '%s  %s\n' "$SHA" "$PART" | sha256sum -c -
chmod 600 "$PART"
mv "$PART" "$DEST"
printf '%s  %s\n' "$SHA" "$DEST" > "$DEST.sha256"
