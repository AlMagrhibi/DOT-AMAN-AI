# Dot Aman AI — Codex Edition

مساعد محلي لأمن المعلومات وإدارة Linux، بنواة Go وواجهة Terminal مضغوطة. التطوير والاختبارات على Codex Cloud، والتشغيل النهائي على CachyOS/Arch Linux دون Cloud AI أو Telemetry.

**الحالة: نسخة قابلة للبناء والتقييم داخل Cloud.** شغّلت `llama-server` فعليًا على نموذج Qwen3‑4B الرسمي، وثبّتُّ بصمة الملف والترخيص، ومرّ اختبار التكامل الواقعي العربي/الإنجليزي وJSON Schema والبث والإلغاء وRAG والجلسات والذاكرة ودورة Auto. النتائج Cloud CPU ولا تمثل أداء جهازك المحلي.

```sh
./scripts/check.sh
mkdir -p bin
go build -tags sqlite_fts5 -trimpath -o bin/dotamanai-codex ./cmd/dotamanai-codex
# After starting a verified local llama-server:
./bin/dotamanai-codex doctor
./bin/dotamanai-codex
```

Requires Go 1.27.3 and a C compiler for SQLite. See [local setup](LOCAL_SETUP.md) for model/configuration/Arch instructions and [project map](PROJECT_MAP.md) for exact completion status.

Implemented: strict TOML config; loopback-only inference and SSE; persistent sessions; reviewed project/episodic memory and automatic two-level source-linked summaries; hybrid local FTS5/BM25 plus Nomic embeddings with Arabic normalization and citations; bounded TXT/PDF/DOCX import and OWASP/CISA sync connectors with last-good/conditional updates; fixed diagnostics plus constrained authorized service/package actions with validation/audit/cancellation; Analysis/Confirm/Auto modes with runtime/step/retry limits; Markdown TUI, sessions, search, streaming, cancellation, copy command and Arabic Markdown/JSON/optional PDF reports. Files, tool observations and knowledge never act as executable instructions.

Cloud evidence includes real SQLite, actual `uname`/process cancellation, a real PTY with a clearly labeled local HTTP fixture, live OWASP import, and a real pinned Qwen3‑4B GGUF run. The unchanged synthetic retrieval evaluation is Recall@5 60/60 (100%); this is a narrow lexical fixture benchmark, not broad answer quality. Details: [benchmarks](BENCHMARKS.md), [known issues](KNOWN_ISSUES.md), [implementation report](CODEX_IMPLEMENTATION_REPORT.md).

[Specification](docs/SPECIFICATION.md) · [Architecture](ARCHITECTURE.md) · [Threat model](SECURITY.md) · [Dependencies](DEPENDENCIES.md).
