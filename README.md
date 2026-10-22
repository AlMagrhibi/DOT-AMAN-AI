# Dot Aman AI — Codex Edition

مساعد محلي لأمن المعلومات وإدارة Linux، بنواة Go وواجهة Terminal مضغوطة. التطوير والاختبارات على Codex Cloud، والتشغيل النهائي على CachyOS/Arch Linux دون Cloud AI أو Telemetry.

**الحالة: نسخة تطوير قابلة للبناء، والتسليم الكامل لم يكتمل.** اختبارات النواة والـHTTP provider وSQLite وRAG والذاكرة والأدوات والـTUI تعمل. llama.cpp بُني، لكن تشغيل GGUF حقيقي متوقف على سماح الشبكة بتنزيل النموذج؛ لا توجد نتائج استدلال حقيقي أو أداء جهاز المستخدم.

```sh
./scripts/check.sh
mkdir -p bin
go build -tags sqlite_fts5 -trimpath -o bin/dotamanai-codex ./cmd/dotamanai-codex
# After starting a verified local llama-server:
./bin/dotamanai-codex doctor
./bin/dotamanai-codex
```

Requires Go 1.27.3 and a C compiler for SQLite. See [local setup](LOCAL_SETUP.md) for model/configuration/Arch instructions and [project map](PROJECT_MAP.md) for exact completion status.

Implemented: strict TOML config; loopback-only inference and SSE; persistent sessions; reviewed memory and two-level source-linked summaries; local FTS5/BM25 with Arabic normalization and citations; explicit file import and OWASP/CISA sync connectors; fixed read-only tools with validation/audit/cancellation; Analysis/Confirm/Auto modes; Markdown TUI, sessions, search, copy command and local Markdown/JSON reports. Files and knowledge never act as executable instructions.

Cloud tests include real SQLite, actual `uname`/process cancellation, a real PTY with a clearly labeled local HTTP fixture, and a live OWASP import. Narrow synthetic retrieval evaluation: Recall@5 58/60 (96.7%), **not** a broad answer-quality result. Details: [benchmarks](BENCHMARKS.md), [known issues](KNOWN_ISSUES.md), [implementation report](CODEX_IMPLEMENTATION_REPORT.md).

[Specification](docs/SPECIFICATION.md) · [Architecture](ARCHITECTURE.md) · [Threat model](SECURITY.md) · [Dependencies](DEPENDENCIES.md).
