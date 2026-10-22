# تقرير تنفيذ Dot Aman AI على Codex Cloud

المستودع: https://github.com/AlMagrhibi/DOT-AMAN-AI
الفرع: `main`
آخر Commit موثق وقت التحقق البرمجي: `b61936a86c77b6e98c6e66f10da1a1adfb74b3cf`.

التنفيذ والاختبارات تمت داخل Cloud. الأوزان GGUF وقواعد البيانات والسجلات لا تدخل Git. كل أرقام الأداء أدناه تخص Cloud ولا تمثل جهاز CachyOS/i5-1145G7.

## المكونات المنفذة

- Go core بإعداد TOML صارم، SQLite WAL/FTS5، جلسات ورسائل أصلية، migrations، memory project/episodic، تصحيح وحذف وتصدير.
- Provider محلي إلى llama.cpp عبر loopback رقمي فقط، مع health/model info/JSON Schema/stream/cancel/usage وحدود الطلب والاستجابة.
- Knowledge SourceRegistry ببيانات URL/fallback/format/license/max bytes/bytes/ETag/Last-Modified/status/error history، ومصادر CVE/NVD/EPSS/OSV/Go vuln/CWE/CAPEC/ATT&CK/OWASP/Arch/kernel المسجلة.
- RAG هجين: FTS5/BM25 وتطبيع عربي مع embeddings محلية Nomic، provenance/SHA-256، نسخ وثائق ذرية، واستيراد TXT/MD/JSON/CSV/LOG/TOML/Go/Shell وPDF النصي وDOCX.
- CISA KEV primary + official mirror وOWASP sync يدوي مشروط، مع last-known-good، ETag/Last-Modified، حدود 8 MiB، redirects مضبوطة وسجل مزامنة.
- Agent بثلاثة أوضاع، `max_steps/max_runtime_seconds/max_retries`، أدوات تشخيص ثابتة وأداتا إصلاح مصرح بهما فقط (`service_restart`, `package_install`) مع argv validation/audit/cancellation.
- TUI Bubble Tea مضغوطة: streaming، جلسات، Markdown، إلغاء، resize، `/stats`، `/copy`، `/export`، وإشعار عند غياب دليل RAG.
- تقارير عربية Markdown وlegacy JSON وstructured `.security.json`، وPDF عند وجود Unicode TrueType محلي؛ غياب الخط يفشل PDF صراحةً ولا يفسد MD/JSON.
- Experiment log يتضمن plan/attempts/validated/lessons مع بقاء النصوص غير موثوقة.

## أدلة Cloud الفعلية

- نموذج المحادثة: Qwen3-4B Q4_K_M من `Qwen/Qwen3-4B-GGUF`، revision `bc640142c66e1fdd12af0bd68f40445458f3869b`، Apache-2.0، SHA-256 `7485fe6f11af29433bc51cab58009521f205840f5b4ae3a32fa7f92e8534fdf5`، الحجم 2,497,280,256 بايت. الوزن خارج Git.
- llama.cpp: commit `c7fc5094179bf8a9c73c18593db714f8da41a402`، CPU-only Cloud.
- أحدث Qwen بعد فحص data-boundary: كل checks نجحت، p50 `5.772s`، p95 `7.621s`، `7.914` end-to-end tok/s، peak RSS مقاس `5,258,907,648` بايت، CPU متوسط `388%` من نواة واحدة. الدليل: `docs/validation/real-model-4b-postboundary.json`.
- أُعيد تشغيل التقييم بعد إصلاح مخطط الأدوات ونقل مصادر المعرفة. على مضيف Cloud مختلف أبطأ، نجحت العربية/الإنجليزية وStreaming cancellation، لكن طلبات JSON Schema تجاوزت مهلة 120 ثانية؛ تم حفظ الفشل صراحةً في `docs/validation/real-model-4b-toolsafe-failed.json` ولم يُستبدل به الدليل الناجح. قياس `llama-bench` لنفس المضيف هو `9.12` prompt tok/s و`0.64` generation tok/s (`docs/validation/llama-bench-4b-current-host.json`).
- نموذج embeddings: Nomic `nomic-embed-text-v1.5.Q4_K_M.gguf`، Apache-2.0، metadata lock وSHA في `configs/embedding-model-lock.json`; التقييم الحقيقي 12/12، Recall@5 `1.0`، العربية 2/2، mean latency `32.62ms`: `docs/validation/semantic-real-postboundary.json`.
- CISA الحقيقي: 1,754 سجلًا، CVE search، ثم طلب مشروط `not-modified` مع last-good: `docs/validation/cisa-real-sourceregistry.json`.
- Arch container: توفر pacman/systemd/journalctl/ip/ps/df فقط؛ لم تُشغّل systemd PID1 أو صلاحيات privileged: `docs/validation/arch-container.json`.
- Offline: `strace` أثناء CLI مع proxies خارجية معطلة؛ 23 اتصالًا و0 non-loopback IPv4: `docs/validation/offline-contract.json`.
- Fresh clone من `b61936a` و`go mod verify` و`check.sh` وbuild وPTY موثقة في `docs/validation/clean-clone-4ce71ee.json`.

## بوابات الجودة

`gofmt`, `go vet`, `go test -tags sqlite_fts5`, `go test -race`, build، PTY، و`go mod verify` نجحت. Fuzz الحالي نجح لـarguments/suggestions/Arabic normalization بأكثر من 68k/101k/105k حالة (خمس ثوانٍ لكل هدف). `govulncheck@v1.9.0 ./...` أعاد `No vulnerabilities found`.

## الحالة والحدود

كل متطلبات Cloud البرمجية في `REQUIREMENTS_TRACEABILITY.md` مصنفة Complete مع دليل. لا يعني ذلك جودة نموذج عامة أو faithfulness بشريًا على كل corpus. لا تزال اختبارات Target Unverified: KDE Wayland clipboard، RTL/bidi بصريًا، Arch/CachyOS systemd/pacman/journal في جلسة حقيقية، المصادقة privileged، throughput/RSS على i5-1145G7، وعزل الشبكة الفيزيائي. PDF Arabic shaping يحتاج فحص viewer محلي.

التشغيل المحلي الكامل موثق في `LOCAL_SETUP.md`. يتطلب التشغيل نموذجًا محليًا يتحقق المستخدم من بصمته، ولا يعتمد على ملفات Cloud أو حساب Cloud.
