# التشغيل المحلي على CachyOS / Arch Linux

التطبيق قيد التحقق؛ لا توجد أوزان نموذج ضمن المستودع. الاستدلال الحقيقي ما زال غير مختبر بسبب حظر تنزيل GGUF في بيئة Cloud. لا تستخدمه كمرجع أمني مستقل دون مراجعة الأدلة.

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

Pinned llama.cpp commit: `c7fc5094179bf8a9c73c18593db714f8da41a402`, MIT. CPU build uses four build jobs and no embedded Web UI. The build is verified in Cloud; real model startup remains pending.

Reference model: official Qwen3-1.7B Q4_K_M; Qwen3-0.6B Q4_K_M may be used for a smaller Cloud smoke test. Check the publisher model card/license and exact filename, immutable revision, and LFS SHA-256 first. No model fingerprint is claimed because weights have not been downloaded.

Optional explicit download (replace placeholders with actual publisher metadata; do not run them literally):

```sh
./scripts/download-model.sh 'https://huggingface.co/Qwen/MODEL/resolve/REVISION/FILE.gguf' "$HOME/.local/share/dotamanai-codex/models/model.gguf" PUBLISHER_SHA256
```

The script verifies SHA-256 and retains HTTPS verification. It never runs automatically. A model up to 2 GiB is allowed by that helper; larger downloads require deliberate review. Keep the model card/license with your local model.

```sh
export LLAMA_SERVER="$HOME/.cache/dotamanai-codex/llama.cpp/build/bin/llama-server"
export MODEL_PATH="$HOME/.local/share/dotamanai-codex/models/model.gguf"
# Terminal 1; keep it running:
./scripts/start-model.sh
# Terminal 2:
curl --noproxy '*' --fail http://127.0.0.1:8081/health
dotamanai-codex doctor
dotamanai-codex
```

Default inference: numeric loopback `127.0.0.1:8081`; proxies and redirects are disabled for inference. Model service binds loopback. No auto download, sync, cloud AI or telemetry. Processes must restart after machine restart. Try context 2048 and 4096 on the target; do not extrapolate Cloud performance.

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

Flags precede the command. Imported files are data, never commands. Plain text/Markdown/JSON/CSV/log/TOML/Go/shell text import is supported up to 2 MiB; PDF/DOCX are not supported. Search is lexical FTS5/BM25, with Arabic letter/diacritic normalization; semantic synonyms need further work. No embedding service exists.

TUI: `/help`, `/new`, `/sessions`, `/resume ID`, `/delete-session ID`, `/mode analysis|confirm|auto`, `/search WORDS`, `/import PATH`, `/sources`, `/remember TEXT`, `/facts`, `/correct ID TEXT`, `/forget ID`, `/experiments`, `/export`, `/stats`, `/copy`. PgUp/PgDown scroll, Ctrl+C cancels a request, Ctrl+Q quits. Terminal-dependent Arabic bidi behavior remains a target acceptance check. `/copy` requires local `wl-copy` (`wl-clipboard` package); it copies only on an explicit command. A clipboard substitute and PTY interaction are Cloud-tested; actual KDE/Wayland is not.

`/remember` explicitly promotes user-provided memory with provenance. Summaries are labeled unverified, retain original message ranges, and are invalidated after source edits/deletions. `summarize` is an explicit bounded local operation; it does not train the model. Application history reads at most 64 recent messages for generation. Original messages remain stored and exportable.

## اختبارات القبول على الجهاز

1. Validate real GGUF health, Arabic/English output, schema suggestions and interruption.
2. Measure TTFT, throughput, latency and total llama-server RSS at 2048/4096 context on i5-1145G7; Cloud figures cannot establish these.
3. Test KDE Wayland copy, mixed Arabic/code rendering, resize, long sessions and confirm/deny.
4. Verify pacman, systemctl, journalctl and ip diagnostics on Arch; Cloud is Debian without a running systemd manager.
5. Disconnect external networking while retaining loopback: chat, stored search, sessions, tools and report exports must work. Do not invoke manual sync while offline.
6. Inspect `KNOWN_ISSUES.md`; complete real-model checks and rerun the current vulnerability scanner before relying on the release.

## Reproduce the real Cloud evaluation

Development-only, after obtaining the exact official model in `configs/model-lock.json` and enabling its download CDN. `govulncheck v1.9.0` is the supported scanner for the pinned Go toolchain.

```sh
export LLAMA_SERVER=/workspace/llama.cpp/build/bin/llama-server
export MODEL_PATH=/workspace/models/Qwen3-0.6B-Q8_0.gguf
python3 tests/run_real_model.py --runs 3 --output docs/validation/real-model.json
```

The wrapper verifies the full model digest/size, refuses an occupied port, starts/stops only its own server, runs actual Go provider/app evaluation and preserves failed outcomes. Reports contain only synthetic test data. The official small model is a Cloud smoke target, not evidence of Qwen1.7B quality or target-device performance. This command is **not yet real-model verified** while CDN access is denied. The standalone `go run -tags sqlite_fts5 ./cmd/dotaman-eval` can target a separately started numeric-loopback server; it has no mock fallback.
