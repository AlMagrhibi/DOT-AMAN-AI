# Cloud validation and benchmark evidence

Measured on Codex Cloud (Linux x86_64, Debian, GNU C/C++ 14.2.0, Go 1.27.3). Environment exposed 5 logical CPUs and approximately 33 GiB memory; these are not the target i5-1145G7/16GB. llama.cpp built with four jobs, portable CPU flags, at `c7fc5094179bf8a9c73c18593db714f8da41a402` (0.7.0-dev). The real-model figures below describe Cloud only.

## Retrieval

Run: `bin/dotamanai-codex -config configs/example.toml evaluate testdata/evaluation`.

Separate corpus/query JSON: 20 synthetic security/Linux documents and 60 labeled Arabic/English queries. FTS5 BM25, top 5, one relevant document per query. The unchanged evaluation after the natural-question retrieval fix has Recall@5 = **60/60 = 1.0** and fixed-denominator Precision@5 = **0.2** (only one relevant label exists). Citation provenance accuracy = **1.0** for retrieved fixture URLs and digests; this is not generated-answer faithfulness. The original 58/60 regression remains in its historical report; current evidence is [JSON](docs/validation/retrieval-resumed.json).

This small synthetic set tests lexical retrieval and normalization. It cannot establish general RAG quality, semantic understanding or production security coverage.

## Executed checks

- `./scripts/check.sh`: gofmt, vet, tests with fresh execution, build; passed for implemented components.
- `go test -race -tags sqlite_fts5 -count=1 ./...`: passed (repeat after final changes recorded in validation summary).
- Five-second fuzz runs with two workers: tool arguments 53,261 executions; structured suggestions 69,909; Arabic normalization 90,006; all passed. Fuzzing is bounded evidence, not exhaustive proof.
- `python3 tests/pty_smoke.py`: actual PTY, Arabic input/stream output, stats, exit; CLI JSON response, process restart/session export, individual message edit/delete, local import and Arabic search passed. Local HTTP fixture is explicitly synthetic; no claim of model inference. Clipboard substitute and resize unit tests passed; actual Wayland unrun.
- Live OWASP SQL Injection Prevention source import and `prepared statements` retrieval passed. Document content digest `0c00c87970eceb404ec4061a30b0fbe3390d870244f3d0802465e20c39178bbb` at fetch time. Source is CC-BY-SA-4.0; live content is stored only locally, not copied into Git fixtures.
- `go mod verify`: all downloaded modules verified. `govulncheck` v1.9.0 completed with **No vulnerabilities found** after updating the reachable dependency findings. Full output: [govulncheck.txt](docs/validation/govulncheck.txt).
- After the knowledge/memory/tool batch, `go test -race -tags sqlite_fts5 -count=1 ./...` passed. Bounded fuzz runs also passed: `FuzzArguments` 68,773 executions, `FuzzSuggestions` 101,298 executions, and `FuzzNormalize` 105,511 executions, each for five seconds with five workers. The current `scripts/check.sh` gate passed after the same changes.
- An Arch Linux `base` container was pulled at digest `sha256:52dfbd5afb23a220d5ee92dafcb4c788d96c026ed04179890c23adef4c897fa0`; pacman 7.1, systemd/journalctl 262, ip, ps and df command availability were checked. Evidence: [arch-container.json](docs/validation/arch-container.json). This is a command-availability check, not a booted systemd or CachyOS acceptance run.
- The current real Qwen3-4B run after the data-boundary change passed every declared check with one repetition: p50 5.772s, p95 7.621s, 7.914 end-to-end output tok/s, sampled peak llama-server RSS 5,258,907,648 bytes, CPU average 388% of one core. Evidence: [real-model-4b-postboundary.json](docs/validation/real-model-4b-postboundary.json). This remains Cloud-only and small-sample evidence.
- A later revalidation after the tool-schema and source-transport hardening was retained as a failure rather than overwritten: Arabic/English and streaming cancellation passed, but the current Cloud host's JSON-schema requests exceeded the 120-second provider timeout, so downstream RAG/memory/tool checks were not executable in that run. Evidence: [real-model-4b-toolsafe-failed.json](docs/validation/real-model-4b-toolsafe-failed.json). This is a host-performance variance, not a passing claim.
- A same-day current-host `llama-bench` rerun measured 9.12 prompt tok/s and 0.64 generation tok/s at four threads on an Intel Xeon Platinum 8573C Cloud host; the earlier 42.23/14.04 result was on a different AMD EPYC Cloud host. Evidence: [llama-bench-4b-current-host.json](docs/validation/llama-bench-4b-current-host.json). Neither result predicts the target i5-1145G7.
- Re-running the real Nomic embedding server after the same change passed 12/12 independent semantic cases, including 2/2 Arabic queries; Recall@5 was 1.0 with mean retrieval latency 32.62 ms. Evidence: [semantic-real-postboundary.json](docs/validation/semantic-real-postboundary.json). The live CISA evaluator also passed again with 1,754 records, conditional `not-modified`, and CVE search: [cisa-real-postboundary.json](docs/validation/cisa-real-postboundary.json).
- After completing SourceRegistry metadata migration, the live CISA evaluation passed again with the expanded registry and preserved conditional/last-good behavior: [cisa-real-sourceregistry.json](docs/validation/cisa-real-sourceregistry.json).
- `tests/offline_contract.py` ran the built CLI against a loopback fixture while HTTP/HTTPS/ALL proxy variables pointed to an unusable address; `strace` observed 23 connects and zero non-loopback IPv4 connects. Evidence: [offline-contract.json](docs/validation/offline-contract.json).
- A fresh HTTPS clone of the current pushed code commit `27c0c0a70fb9f6730f767d475b7e6fad4c95435d` passed `go mod verify`, `scripts/check.sh`, a clean tagged build and the PTY/CLI fixture, including message edit/delete. Evidence: [clean-clone-final.json](docs/validation/clean-clone-final.json). The earlier clean-clone report remains for historical comparison.

Cloud functional checks use real SQLite/WAL/FTS5 and real local processes. Offline verification covers loopback-only transport and local operations; a target-device physical network-isolation test remains required.

## Fresh-clone and reproducible build

Verified pushed revision `5afeaf0a62daa173d6695b4c453c276622ec8787` from a fresh HTTPS clone in a second directory: formatting, vet, tests/build, CLI/PTY fixture and module verification passed; no checkout changes. Two builds with `-tags sqlite_fts5 -trimpath -buildvcs=false` produced identical SHA-256 `b6fa00ffade9f5acf8beee3662307b646da6baf54579e006d4fba7d51ea1f91d` using the same Cloud toolchain, architecture and module cache. This does not establish bit-identical output across different C toolchains or architectures. Details: [reproducibility evidence](docs/validation/reproducibility.json).

## Resumed Cloud evaluation

Continuing from b54277e, with no reinitialization. Current detailed govulncheck v1.9.0 scan is clean after updating vulnerable dependencies; the earlier scanner crash and findings were not counted as passes. Latest full output is docs/validation/govulncheck.txt.

A failing regression proved that requiring every word of a natural-language question could hide a matching reference. Search now retains exact matching first and falls back to ranked lexical content words. **Same 20 documents and same 60 queries**, unchanged expected labels: Recall@5 **60/60 (1.0)**, fixed-denominator Precision@5 **0.2**, citation provenance **1.0**. See docs/validation/retrieval-resumed.json. This remains a small lexical fixture benchmark, not real model answer quality.

The evaluator (`cmd/dotaman-eval`, `tests/run_real_model.py`) executes repeated real Arabic/English SSE, schema validity/selection, cancellation, RAG answer evidence, session restart, reviewed-memory recall and actual agent diagnostics. It records per-request latency/TTFT/usage, nearest-rank p50/p95 and output tokens per end-to-end second. The wrapper samples server CPU/RSS every 100 ms, verifies the GGUF against publisher digest before launch, and stops only its own server. The final report is [real-model-4b-final.json](docs/validation/real-model-4b-final.json): all checks passed with 4B Q4_K_M, 0.0 temperature, p50 5.723s, p95 7.157s, 9.30 end-to-end output tok/s and sampled peak server RSS 5.21GB. This is a small smoke/evaluation set, not a global quality claim.

The independent pinned `llama-bench` run measured 42.23 prompt tok/s and 14.04 generation tok/s at four CPU threads; raw summary: [llama-bench-4b.json](docs/validation/llama-bench-4b.json). The earlier 0.6B and 1.7B runs remain tracked as failed/diagnostic evidence; they are not overwritten by the passing 4B report. The GGUF files and binaries remain outside Git.

After rebuilding the current CLI binary, `doctor` and a real Arabic `-json chat` completed against the same loopback 4B server; the result is [real-cli-4b.json](docs/validation/real-cli-4b.json). The PTY test remains intentionally labeled as a fixture test, and KDE/Wayland clipboard was not executable in Cloud.
