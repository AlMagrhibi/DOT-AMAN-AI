# Dot Aman AI requirements traceability

This matrix is derived from the attached `Aman AI Codex.md` (the checked-in copy is `docs/SPECIFICATION.md`, identical at the time of review). Status describes evidence in the current checkout; a function stub, an interface, or a fixture-only test is not counted as implementation. `Cloud Verified` and `Target Unverified` are kept separate.

| ID | Requirement from Aman.md | Implementation and evidence | Status | Acceptance still required |
|---|---|---|---|---|
| R-001 | Develop on Cloud, push the public GitHub repository continuously | `/workspace/DOT-AMAN-AI`, `origin/main`, pushed commits; setup draft | Complete | None for source delivery |
| R-002 | Public repository excludes secrets, databases, logs and GGUF weights | `.gitignore`, model locks and validation metadata; `git ls-files` audit | Complete | Recheck on every release |
| R-003 | Go is the core language; local-only inference and storage | `go.mod`, `internal/*`, loopback-only `internal/model/model.go`, SQLite storage | Complete | Target-device run |
| R-004 | Local llama.cpp provider: health, model info, complete, stream, cancel, usage | `internal/model/model.go`; model tests; `docs/validation/real-model-4b-final.json` | Complete for tested scope | Broader model matrix |
| R-005 | Structured JSON Schema tool suggestions and validation before execution | `HTTP.Suggest`, `ParseSuggestions`, ordered schema properties; model/app tests and real 4B 6/6 | Complete for tested scope | Adversarial model corpus |
| R-006 | Context budget reserves system rules, current prompt, history, RAG and answer | `internal/contextmanager/context.go`, `app.Run` | Partial | Tokenizer-backed accounting and long-context acceptance |
| R-007 | Session, episodic, project and experiment memory with IDs, source, review, correction, deletion and export | `internal/storage`, `internal/memory`, TUI/CLI commands and tests | Partial | Automatic promotion/retrieval and long-conversation acceptance |
| R-008 | Automatic bounded hierarchical summaries while retaining originals | `memory.Compress` and `SummaryContext`; explicit CLI only | Partial | Automatic trigger in conversation path and restart/quality tests |
| R-009 | Knowledge SourceRegistry with URL, license, sync state and error history | `knowledge.New`, `Sources`, `sync_history` | Partial | Full registry metadata and exposed error history |
| R-010 | CISA KEV, OWASP and additional security sources with manual sync | `knowledge.Sync` supports CISA/OWASP; live OWASP evidence | Partial | CISA live/fallback, source coverage and conditional sync |
| R-011 | Atomic versioned documents/chunks, provenance, deduplication and safe replacement | `knowledge.put`, `document_versions`, FTS5 chunks and tests | Partial | Relationship metadata, update diff and failure recovery tests |
| R-012 | SQLite FTS5/BM25 Arabic/English lexical search and CVE/CWE lookup | `knowledge.Search`, normalization, retrieval fixtures 60/60 | Complete for lexical scope | Independent broader corpus |
| R-013 | Local embeddings and semantic retrieval alongside lexical retrieval | No embedding index/provider in checkout | Missing | Implement offline embedding model/index, hybrid ranker and benchmark |
| R-014 | RAG context, citations, faithfulness and missing-evidence behavior | `app.Run` references and citations; synthetic real-model RAG check | Partial | Citation/faithfulness corpus and unsupported-claim tests |
| R-015 | Import TXT/MD/JSON/CSV/LOG/TOML/source files with limits and provenance | `knowledge.Import` | Complete for listed text formats | More encoding/error fixtures |
| R-016 | PDF text extraction and DOCX extraction/fingerprints/chunking | `knowledge.Import` rejects both extensions | Missing | Implement and test safe PDF/DOCX ingestion |
| R-017 | Ingestion treats documents/logs/tool output as untrusted data | `contextmanager` prefixes and app system rules; injection tests partial | Partial | End-to-end prompt-injection corpus |
| R-018 | Manual source updates with HTTPS allowlist, redirects/SSRF/DNS rebinding, limits and last-known-good | fixed URLs, redirect rejection and bounded reads | Partial | Conditional validators, last-good retention and network failure tests |
| R-019 | Fixed ToolRegistry, argument schemas, time/output bounds, process groups, audit fail-closed | `internal/tools/tools.go`, unit/security tests | Complete for read-only tools | Mutating authorized Arch tools and broader audit tests |
| R-020 | Analysis, Confirm and Auto modes with max steps/runtime/retries/resource budgets | `internal/app/app.go`, TUI mode; 4-call/90s bound | Partial | Retry policy, resource budget and authorized mutation workflow |
| R-021 | Arch/CachyOS diagnostics and authorized repair with evidence/rollback | Read-only `services`, `packages`, `journal`, `network` commands | Partial | Arch container validation and explicit safe repair tools |
| R-022 | Compact OpenCode-like TUI: scroll, Markdown, streaming, sessions, shortcuts, cancellation | `internal/tui/tui.go`, PTY fixture and unit tests | Partial | Real-model PTY, long output/resize and target terminal acceptance |
| R-023 | Real performance line: tokens, rate, latency, CPU/RAM/GPU and `/stats` | `metrics`, TUI `Stats`, real server wrapper and llama-bench reports | Complete for Cloud measurement | i5/KDE target measurements |
| R-024 | Copy last result with actual clipboard behavior | `WaylandClipboard`, `/copy`, mock clipboard test | Partial | KDE Wayland `wl-copy` acceptance |
| R-025 | Arabic Markdown/JSON cybersecurity reports with evidence, CVE/CWE, CVSS/EPSS/KEV and recommendations | `internal/reporting/report.go` emits basic MD/JSON | Partial | Structured security report schema and quality tests |
| R-026 | PDF report export when required by the final format | No PDF exporter | Missing | Implement only with a tested local renderer or document explicit scope decision |
| R-027 | Experiment log with plans, sanitized args, validation, outcomes, retries and lessons | `experiments`, audit and `Outcome` persistence | Partial | Retry/lesson fields and report integration |
| R-028 | Offline chat, memory, RAG, tools and reports with loopback only | loopback provider and local DB/tests | Partial | Network-isolation integration test |
| R-029 | Evaluation suite for performance, JSON, tools, Arabic/English, security, RAG, memory, TUI and offline | `cmd/dotaman-eval`, fixtures, race/PTY/govuln reports | Partial | Independent long/PDF/DOCX/semantic/adversarial suites |
| R-030 | Fuzz, race, build, vet, module and vulnerability gates | `scripts/check.sh`, Go race/fuzz seeds, `govulncheck.txt` | Complete for current code | Re-run after each implementation batch |
| R-031 | Clean clone builds and documented CachyOS setup | `LOCAL_SETUP.md`, scripts, reproducibility evidence | Partial | Fresh clone after new features and Arch container |
| R-032 | Final target-device distinction and no unsupported claims | `KNOWN_ISSUES.md`, benchmark scope labels | Complete as documentation | Execute only tests requiring the user's hardware when available |

## Classification policy

`Complete` means the requirement is implemented and has a relevant test or direct evidence in the repository. `Partial` means a meaningful subset exists but the stated requirement is broader. `Missing` means no production implementation exists. `Unverified` is reserved for implemented behavior whose required environment could not be run. A row may move to Complete only after its acceptance evidence is committed; changing wording or reducing the test set does not qualify.

The first implementation queue is R-013, R-016, R-018/R-010, R-008/R-007, R-020/R-021, R-025/R-026, and R-022/R-024, followed by independent acceptance suites R-014/R-017/R-028/R-029 and a clean-clone/Arch run.
