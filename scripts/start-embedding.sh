#!/bin/sh
set -eu
: "${LLAMA_SERVER:?set LLAMA_SERVER to the built llama-server executable}"
: "${EMBEDDING_MODEL_PATH:?set EMBEDDING_MODEL_PATH to a verified local embedding GGUF}"
test -f "$EMBEDDING_MODEL_PATH"
exec "$LLAMA_SERVER" --model "$EMBEDDING_MODEL_PATH" --host 127.0.0.1 --port "${EMBEDDING_PORT:-8082}" --ctx-size "${EMBEDDING_CONTEXT:-2048}" --threads "${EMBEDDING_THREADS:-4}" --parallel 1 --embedding --pooling mean
