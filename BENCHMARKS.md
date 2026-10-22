# Cloud validation and benchmark evidence

Measured on Codex Cloud (Linux x86_64, Debian, GNU C/C++ 14.2.0, Go 1.27.3). Environment exposed 5 logical CPUs and approximately 33 GiB memory; these are not the target i5-1145G7/16GB. llama.cpp built with four jobs, portable CPU flags, at `c7fc5094179bf8a9c73c18593db714f8da41a402` (0.7.0-dev). No GGUF installed: real inference throughput, TTFT, p50/p95 and server peak RSS **unrun**.

## Retrieval

Run: `bin/dotamanai-codex -config configs/example.toml evaluate testdata/evaluation`.

Separate corpus/query JSON: 20 synthetic security/Linux documents and 60 labeled Arabic/English queries. FTS5 BM25, top 5, one relevant document per query. Recall@5 = number of queries retrieving expected ID / 60 = **58/60 = 0.9667**. Precision@5 uses a fixed denominator 5 per query = **0.1933** (only one relevant label exists). Citation provenance accuracy = **1.0** for retrieved fixture URLs and digests; this is not generated-answer faithfulness. Per-case measured search durations and missing cases: [JSON](docs/validation/retrieval.json).

This small synthetic set tests lexical retrieval and normalization. It cannot establish general RAG quality, semantic understanding or production security coverage. The two misses are retained in results; the benchmark was not redefined after observing them.

## Executed checks

- `./scripts/check.sh`: gofmt, vet, tests with fresh execution, build; passed for implemented components.
- `go test -race -tags sqlite_fts5 -count=1 ./...`: passed (repeat after final changes recorded in validation summary).
- Five-second fuzz runs with two workers: tool arguments 53,261 executions; structured suggestions 69,909; Arabic normalization 90,006; all passed. Fuzzing is bounded evidence, not exhaustive proof.
- `python3 tests/pty_smoke.py`: actual PTY, Arabic input/stream output, stats, exit; CLI JSON response, process restart/session export, local import and Arabic search passed. Local HTTP fixture is explicitly synthetic; no claim of model inference. Clipboard substitute and resize unit tests passed; actual Wayland unrun.
- Live OWASP SQL Injection Prevention source import and `prepared statements` retrieval passed. Document content digest `0c00c87970eceb404ec4061a30b0fbe3390d870244f3d0802465e20c39178bbb` at fetch time. Source is CC-BY-SA-4.0; live content is stored only locally, not copied into Git fixtures.
- `go mod verify`: all downloaded modules verified. `govulncheck` database fetch blocked; vulnerability results unavailable.

Cloud functional checks use real SQLite/WAL/FTS5 and real local processes. Offline verification covers loopback-only transport and local operations; a target-device physical network-isolation test remains required.

## Fresh-clone and reproducible build

Verified pushed revision `5afeaf0a62daa173d6695b4c453c276622ec8787` from a fresh HTTPS clone in a second directory: formatting, vet, tests/build, CLI/PTY fixture and module verification passed; no checkout changes. Two builds with `-tags sqlite_fts5 -trimpath -buildvcs=false` produced identical SHA-256 `b6fa00ffade9f5acf8beee3662307b646da6baf54579e006d4fba7d51ea1f91d` using the same Cloud toolchain, architecture and module cache. This does not establish bit-identical output across different C toolchains or architectures. Details: [reproducibility evidence](docs/validation/reproducibility.json).

## Resumed Cloud evaluation

Continuing from b54277e, with no reinitialization. Current detailed govulncheck v1.9.0 scan is clean after updating vulnerable dependencies; the earlier scanner crash and findings were not counted as passes. Latest full output is docs/validation/govulncheck.txt.

A failing regression proved that requiring every word of a natural-language question could hide a matching reference. Search now retains exact matching first and falls back to ranked lexical content words. **Same 20 documents and same 60 queries**, unchanged expected labels: Recall@5 **60/60 (1.0)**, fixed-denominator Precision@5 **0.2**, citation provenance **1.0**. See docs/validation/retrieval-resumed.json. This remains a small lexical fixture benchmark, not real model answer quality.

New evaluator (`cmd/dotaman-eval`, `tests/run_real_model.py`) is prepared for repeated real Arabic/English SSE, schema validity/selection, cancellation, RAG answer evidence, session restart, reviewed-memory recall and actual agent diagnostics. It records per-request latency/TTFT/usage, nearest-rank p50/p95 and output tokens per end-to-end second. Wrapper samples server CPU/RSS every 100 ms and verifies GGUF against publisher digest before launch. **These real-model measurements have not executed**: the newly observed download CDN remains denied. An absent real server correctly fails the evaluator without writing a success report. Resource-counter parsing was checked on a real local process; this is not an inference benchmark.
