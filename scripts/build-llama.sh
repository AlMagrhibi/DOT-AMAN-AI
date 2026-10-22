#!/bin/sh
set -eu
DEST=${1:?usage: build-llama.sh /absolute/dependency-cache/llama.cpp}
REV=c7fc5094179bf8a9c73c18593db714f8da41a402
case "$DEST" in /*) ;; *) echo 'absolute path required' >&2;exit 1;; esac
if [ ! -d "$DEST/.git" ]; then
 mkdir -p "$DEST"
 git -C "$DEST" init
 git -C "$DEST" remote add origin https://github.com/ggml-org/llama.cpp.git
 git -C "$DEST" fetch --depth 1 origin "$REV"
 git -C "$DEST" checkout --detach "$REV"
fi
test "$(git -C "$DEST" rev-parse HEAD)" = "$REV"
cmake -S "$DEST" -B "$DEST/build" -G Ninja -DCMAKE_BUILD_TYPE=Release -DGGML_NATIVE=OFF -DLLAMA_CURL=OFF -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_UI=OFF -DLLAMA_USE_PREBUILT_UI=OFF
cmake --build "$DEST/build" --target llama-server llama-bench -j "${BUILD_JOBS:-4}"
"$DEST/build/bin/llama-server" --version
