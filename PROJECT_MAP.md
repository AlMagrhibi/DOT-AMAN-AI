# Project map

| Phase | State | Evidence |
|---|---|---|
| 0 Foundation | Complete | e7fb02a pushed to main; official Go archive digest verified |
| 1 Go core | Complete | Config validation, SQLite WAL/FTS5, persistence/isolation/edit/delete tests; check.sh and doctor passed |
| 2 Tools | Complete | Fixed registry, audit fail-closed, process-group cancellation, output caps, root confinement; 4 tests + fuzz seed passed |
| 3 Provider | Complete | Local-only HTTP, health/model info, chat/complete, schema suggestions, SSE, cancellation/redirect tests |
| 4 llama.cpp | Deferred | Not implemented yet |
| 5 Knowledge | Partial | FTS5 BM25, Arabic normalization, provenance, versioned document replacement/import/search tested; live feed and evaluation pending |
| 6 Memory/context | Partial | Reviewed memory correction/deletion/isolation and source-linked summary storage tested; automatic hierarchical compression pending |
| 7 Agent | Partial | Analysis/Confirm/Auto orchestration, 4-step/90-second limits, RAG/context integration, restart and real uname tests passed; quality evaluation pending |
| 8 TUI | Partial | Bubble Tea streaming, sessions/mode/memory/search commands, confirmation, copy mock, Unicode/resize and real PTY passed; full Markdown and Wayland pending |
| 9 Reports/Arch | Partial | Markdown/JSON export and read-only Arch diagnostics implemented; Arch target not available in Cloud |
| 10 Hardening | Deferred | Not implemented yet |
| 11 Handoff | Deferred | Not implemented yet |
