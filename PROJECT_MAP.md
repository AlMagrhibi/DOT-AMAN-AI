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
| 7 Agent | Deferred | Not implemented yet |
| 8 TUI | Deferred | Not implemented yet |
| 9 Reports/Arch | Deferred | Not implemented yet |
| 10 Hardening | Deferred | Not implemented yet |
| 11 Handoff | Deferred | Not implemented yet |
