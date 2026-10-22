# Project map

States describe implemented scope, not an unconditional product acceptance claim. Development is on `main` with continuous confirmed pushes. Real Qwen3-4B GGUF and vulnerability validation were executed in Cloud; target CachyOS acceptance remains separate.

| Phase | State | Evidence / milestone commit |
|---|---|---|
| 0 Foundation | Complete | Specification, architecture, threat model and official Go digest verified; `e7fb02a` |
| 1 Go core | Complete | Strict config, SQLite/WAL/FTS5, session persistence/isolation/edit/delete and migrations; `96dcb35` |
| 2 Tool core | Complete | Fixed read-only registry, validator/audit fail-closed, argv separation, process-group cancellation and file confinement; `de97000` |
| 3 Provider | Complete for the tested Cloud scope | Local HTTP, health/model info, chat/complete, schema suggestions, SSE, cancellation/redirect/proxy tests; real 4B provider and rebuilt CLI smoke passed; evidence `real-model-4b-final.json` and `real-cli-4b.json` |
| 4 llama.cpp/model | Complete for Cloud evaluation | Server/bench built at `c7fc509`; official Qwen3-4B Q4_K_M downloaded and digest/license verified outside Git; real report and independent benchmark in `docs/validation/real-model-4b-final.json` and `docs/validation/llama-bench-4b.json` |
| 5 Knowledge | Complete for implemented scope | SQLite FTS5/BM25 plus local Nomic embeddings, bounded PDF/DOCX extraction, atomic versions, full reference SourceRegistry metadata, CISA primary/mirror conditional sync and OWASP sync; real semantic evidence `semantic-real-postboundary.json`, CISA evidence `cisa-real-postboundary.json` |
| 6 Memory/context | Complete for bounded local scope | Conservative/custom token estimator, cross-session reviewed project memory, isolated episodic memory, automatic hierarchical compression with originals retained and restart/mutation tests |
| 7 Agent | Complete for tested registry scope | Modes, 4-call/90s limits, RAG/context, actual observations, experiment persistence, deterministic schema planning, duplicate suppression, explicit service/package authorized actions with argv validation |
| 8 TUI | Partial | Bubble Tea, Markdown, stream, sessions, memory, search, confirmation, copy substitute; real PTY + Unicode/resize passed at `ac6f6fa`, `174f6ff`; actual Wayland/target bidi pending |
| 9 Reports/Arch | Partial | Arabic structured Markdown/legacy+security JSON and conditional Unicode-font PDF export; read-only systemd/journal/pacman/ip/process/filesystem registry plus constrained service/package repair; target Arch runtime/privileged authentication not validated |
| 10 Hardening | Complete for Cloud scope | gofmt/vet/build, unit/integration/race, bounded fuzz, module integrity and govulncheck v1.9.0 all passed; real 4B evaluator passed its declared smoke set |
| 11 Handoff | Complete for Cloud delivery | Local setup, pinned build/start/download scripts, traceability/report/benchmark documents, fresh-clone evidence at `b61936a`, and continuously pushed source; final target acceptance remains explicitly separate |

Validation evidence: `docs/validation/quality.json`, `retrieval.json`, `pty-cli.json`, `live-owasp.json`. See KNOWN_ISSUES.md for limitations, BENCHMARKS.md for metrics, and CODEX_IMPLEMENTATION_REPORT.md for delivery status. Nothing labeled Partial should be treated as complete.

## Current evidence

- The official Qwen3-0.6B and Qwen3-1.7B metadata, licenses and digests are tracked in separate model-lock files; their failed real runs remain in `docs/validation/` for auditability.
- The official Qwen3-4B Q4_K_M model passed the declared Cloud evaluation after deterministic regular completions, ordered JSON-schema properties, explicit registry guidance, Arabic terminology grounding and duplicate-call suppression. Its locked metadata is `configs/model-lock-4b.json`; the GGUF is never committed.
- `go test -race`, bounded fuzz checks, PTY/CLI fixture checks, module verification and `govulncheck` v1.9.0 are the current hardening evidence. See `BENCHMARKS.md` for exact commands and measured results.
- Cloud completion does not close target-device acceptance. CachyOS/Arch behavior, Wayland clipboard, terminal bidi rendering, i5-1145G7 performance and external-network isolation still require execution on the user's machine.
