#!/bin/sh
set -eu
: "${LLAMA_SERVER:?set LLAMA_SERVER to the built llama-server executable}"
: "${MODEL_PATH:?set MODEL_PATH to an independently verified local GGUF}"
test -f "$MODEL_PATH"
# exec preserves signal delivery and does not start or contact a cloud service.
exec "$LLAMA_SERVER" --model "$MODEL_PATH" --host 127.0.0.1 --port 8081 --ctx-size "${MODEL_CONTEXT:-4096}" --threads "${MODEL_THREADS:-4}" --parallel 1
