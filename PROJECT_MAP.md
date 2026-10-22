# Project map

States describe implemented scope, not an unconditional product acceptance claim. Development is on `main` with continuous confirmed pushes. Real GGUF and vulnerability validation remain externally blocked.

| Phase | State | Evidence / milestone commit |
|---|---|---|
| 0 Foundation | Complete | Specification, architecture, threat model and official Go digest verified; `e7fb02a` |
| 1 Go core | Complete | Strict config, SQLite/WAL/FTS5, session persistence/isolation/edit/delete and migrations; `96dcb35` |
| 2 Tool core | Complete | Fixed read-only registry, validator/audit fail-closed, argv separation, process-group cancellation and file confinement; `de97000` |
| 3 Provider | Complete (contract) | Local HTTP, health/model info, chat/complete, schema suggestions, SSE, cancellation/redirect/proxy tests; `c4598cf`; real model under phase 4 |
| 4 llama.cpp/model | Partial | Server/bench built at `c7fc509`; real model download denied by network policy; no inference/throughput benchmark |
| 5 Knowledge | Partial | Live OWASP sync/search, CISA connector, atomic versioned FTS5 search; `a69c4a5`, `ac6f6fa`; 58/60 narrow lexical evaluation at `174f6ff`; CISA live/conditional sync/richer source metadata pending |
| 6 Memory/context | Partial | Persistent reviewed facts, source-linked two-level explicit summaries, conservative prompt budgets; `174f6ff`; source mutation invalidation tested; real-model long-dialogue accuracy pending |
| 7 Agent | Partial | Modes, 4-call/90s limits, RAG/context, actual tool observations, experiment success/failure persistence; `ef30006` and hardening follow-up; real model planning quality/recovery pending |
| 8 TUI | Partial | Bubble Tea, Markdown, stream, sessions, memory, search, confirmation, copy substitute; real PTY + Unicode/resize passed at `ac6f6fa`, `174f6ff`; actual Wayland/target bidi pending |
| 9 Reports/Arch | Partial | Private Arabic Markdown/JSON evidence exports, read-only systemd/journal/pacman/ip registry; target Arch runtime/privileged repair not validated or implemented |
| 10 Hardening | Partial | gofmt/vet/build, 32 passing test/subtest/fuzz-seed outcomes with race; no failures/skips; bounded fuzz runs and module integrity passed; vulnerability DB and full real-model quality checks blocked |
| 11 Handoff | Partial | Local setup, pinned build/start/download scripts, saved install/start draft, continuously pushed source; fresh-clone build/tests/PTY and identical binaries verified at `5afeaf0`; source archive provided in chat; full acceptance requires blockers resolved |

Validation evidence: `docs/validation/quality.json`, `retrieval.json`, `pty-cli.json`, `live-owasp.json`. See KNOWN_ISSUES.md for limitations, BENCHMARKS.md for metrics, and CODEX_IMPLEMENTATION_REPORT.md for delivery status. Nothing labeled Partial should be treated as complete.

## Resumed evaluation

- Continued from b54277e without replacing completed components.
- Official Qwen0.6B Q8_0 metadata and Apache-2.0 license reviewed; pinned revision/digest in configs/model-lock.json. CDN redirect us.aws.cdn.hf.co still denied; exact domain added to draft. No weights or real-inference success claimed.
- govulncheck v1.9.0 source scan completed after three dependency fixes: 0 reachable vulnerabilities; non-reachable findings under review. Formatting/vet/unit/build passed after changes.

Dependency follow-up: all reported module/package findings remediated via x/net v0.60.0 and its compatible minimum dependencies. govulncheck -show verbose now reports No vulnerabilities found.
