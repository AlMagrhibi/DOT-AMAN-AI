# التشغيل المحلي على CachyOS / Arch Linux

التطبيق قابل للبناء والتقييم. شُغّل `llama-server` في Cloud على Qwen3-4B Q4_K_M الرسمي بعد التحقق من الحجم والبصمة والترخيص؛ الأوزان لا تُخزّن في المستودع. نتائج Cloud لا تمثل سرعة أو ذاكرة جهازك المحلي، ولا تجعل النموذج مرجعًا أمنيًا مستقلًا.

## البناء

```sh
sudo pacman -S --needed go base-devel git cmake ninja curl
# Go 1.27.3 is pinned in go.mod. Check `go version` before building.
git clone https://github.com/AlMagrhibi/DOT-AMAN-AI.git
cd DOT-AMAN-AI
./scripts/check.sh
mkdir -p bin
go build -tags sqlite_fts5 -trimpath -o bin/dotamanai-codex ./cmd/dotamanai-codex
install -Dm755 bin/dotamanai-codex "$HOME/.local/bin/dotamanai-codex"
```

SQLite uses CGO and bundled SQLite: GCC is needed at build time. Runtime does not need Python or a cloud account. Python is used only by the optional development PTY integration test.

## llama.cpp وGGUF

```sh
./scripts/build-llama.sh "$HOME/.cache/dotamanai-codex/llama.cpp"
```

Pinned llama.cpp commit: `c7fc5094179bf8a9c73c18593db714f8da41a402`, MIT. CPU build uses four build jobs and no embedded Web UI. The build and real model startup were verified in Cloud.

Recommended Cloud/reference candidate: official Qwen3-4B Q4_K_M, locked in `configs/model-lock-4b.json`. The smaller Qwen3-0.6B and Qwen3-1.7B locks remain available for comparison. Check the publisher model card/license and exact filename, immutable revision, and LFS SHA-256 before every local download. The model files stay outside Git.

Optional explicit download (use the exact publisher URL, revision and digest from a model-lock file):

```sh
./scripts/download-model.sh 'https://huggingface.co/Qwen/MODEL/resolve/REVISION/FILE.gguf' "$HOME/.local/share/dotamanai-codex/models/model.gguf" PUBLISHER_SHA256
```

The script verifies SHA-256 and retains HTTPS verification. It never runs automatically. The helper defaults to a 2 GiB limit; set `MODEL_DOWNLOAD_MAX_BYTES` explicitly for a reviewed larger file such as Qwen3-4B (2.50 GB decimal). Keep the model card/license with your local model.

```sh
export LLAMA_SERVER="$HOME/.cache/dotamanai-codex/llama.cpp/build/bin/llama-server"
export MODEL_PATH="$HOME/.local/share/dotamanai-codex/models/Qwen3-4B-Q4_K_M.gguf"
# Terminal 1; keep it running:
./scripts/start-model.sh
# Terminal 2:
curl --noproxy '*' --fail http://127.0.0.1:8081/health
dotamanai-codex doctor
dotamanai-codex
```

Default inference: numeric loopback `127.0.0.1:8081`; proxies and redirects are disabled for inference. Model service binds loopback. No auto download, sync, cloud AI or telemetry. Processes must restart after machine restart. Try context 2048 and 4096 on the target; do not extrapolate Cloud performance.

Optional semantic retrieval uses a second local llama-server in embedding mode. Download and verify the Nomic lock in `configs/embedding-model-lock.json`, then run `scripts/start-embedding.sh` with `LLAMA_SERVER` and `EMBEDDING_MODEL_PATH`; set `embedding_endpoint = "http://127.0.0.1:8082"` in the config and run `reindex-embeddings`. The embedding server is loopback-only and receives no external or cloud requests.

## الإعدادات والبيانات

Without an explicit config the application uses `$XDG_DATA_HOME/dotamanai-codex`, falling back to `~/.local/share/dotamanai-codex`. Data directory mode is 0700 and database mode 0600. For a config, copy `configs/example.toml` to `~/.config/dotamanai-codex/config.toml`, replace **both absolute directories** with your own paths, and pass `-config PATH`. Config is intentionally loaded only when explicitly specified. Cache location for llama build above is `~/.cache/dotamanai-codex`.

`root` is the sole file-tool scope; select a narrow directory. File tools confine symlinks, refuse special files and bound outputs. Diagnostic tools are read-only. No privileged repair or arbitrary shell command is implemented. Analysis is default; Confirm asks before each tool; Auto remains restricted to validated registered tools. CLI Confirm declines tools (use the TUI for interactive confirmation).

```sh
dotamanai-codex -json chat 'اشرح مبدأ أقل امتياز'
dotamanai-codex sessions
dotamanai-codex -session SESSION_ID chat 'نكمل'
dotamanai-codex export-session SESSION_ID
dotamanai-codex summarize SESSION_ID
dotamanai-codex experiments SESSION_ID
dotamanai-codex import ./explicit-file.md
dotamanai-codex search 'صلاحيات الملفات'
dotamanai-codex sources
# Explicit network activity; respect source licenses and attribution:
dotamanai-codex sync owasp
dotamanai-codex sync cisa-kev
```

Flags precede the command. Imported files are data, never commands. Plain text/Markdown/JSON/CSV/log/TOML/Go/shell text import is supported up to 2 MiB; text-based PDF and DOCX extraction is bounded to 16 MiB and tested. Scanned/image-only PDFs are rejected. Search uses lexical FTS5/BM25 and, when configured, local Nomic embeddings with Arabic normalization. Semantic indexing is explicit through `reindex-embeddings`; there is no automatic external download.

TUI: `/help`, `/new`, `/sessions`, `/resume ID`, `/delete-session ID`, `/mode analysis|confirm|auto`, `/search WORDS`, `/import PATH`, `/sources`, `/remember TEXT`, `/facts`, `/correct ID TEXT`, `/forget ID`, `/experiments`, `/export`, `/stats`, `/copy`. PgUp/PgDown scroll, Ctrl+C cancels a request, Ctrl+Q quits. Terminal-dependent Arabic bidi behavior remains a target acceptance check. `/copy` requires local `wl-copy` (`wl-clipboard` package); it copies only on an explicit command. A clipboard substitute and PTY interaction are Cloud-tested; actual KDE/Wayland is not.

`/remember` explicitly promotes user-provided memory with provenance. Reviewed project memory is available across sessions; episodic memory remains session-scoped. Summaries are labeled unverified, retain original message ranges, and are invalidated after source edits/deletions. Compression triggers automatically for long sessions and is also available through `summarize`; it does not train the model. Application history reads at most 64 recent messages for generation. Original messages remain stored and exportable. `/export` writes legacy JSON, structured `.security.json`, and Arabic Markdown. When a local Unicode TrueType font such as `ttf-dejavu` is installed, it also writes a PDF; set `DOTAMAN_PDF_FONT` to an explicit font path when needed. Without a font, the other formats remain successful and PDF is reported unavailable rather than silently producing broken Arabic glyphs.

## اختبارات القبول على الجهاز

1. Validate real GGUF health, Arabic/English output, schema suggestions and interruption.
2. Measure TTFT, throughput, latency and total llama-server RSS at 2048/4096 context on i5-1145G7; Cloud figures cannot establish these.
3. Test KDE Wayland copy, mixed Arabic/code rendering, resize, long sessions and confirm/deny.
4. Verify pacman, systemctl, journalctl and ip diagnostics on Arch; Cloud is Debian without a running systemd manager.
5. Disconnect external networking while retaining loopback: chat, stored search, sessions, tools and report exports must work. Do not invoke manual sync while offline.
6. Inspect `KNOWN_ISSUES.md`; Cloud real-model and vulnerability checks are recorded, but rerun them after changing the model or llama.cpp before relying on a local release.

## Reproduce the real Cloud evaluation

Development-only, after obtaining the exact official model in `configs/model-lock-4b.json`. `govulncheck v1.9.0` is the supported scanner for the pinned Go toolchain.

```sh
export LLAMA_SERVER=/workspace/llama.cpp/build/bin/llama-server
export MODEL_PATH=/workspace/models/Qwen3-4B-Q4_K_M.gguf
python3 tests/run_real_model.py --model-lock configs/model-lock-4b.json --runs 3 --output docs/validation/real-model-4b-final.json
```

The wrapper verifies the full model digest/size, refuses an occupied port, starts/stops only its own server, runs actual Go provider/app evaluation and preserves failed outcomes. Reports contain only synthetic RAG/memory fixture data; language and tool calls use the real GGUF. The final Cloud evidence is `docs/validation/real-model-4b-final.json`; it does not predict target-device performance. The standalone `go run -tags sqlite_fts5 ./cmd/dotaman-eval` can target a separately started numeric-loopback server; it has no mock fallback.
