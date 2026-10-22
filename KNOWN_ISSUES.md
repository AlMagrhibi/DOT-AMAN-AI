# Known issues and outstanding validation

## Current Cloud status

- The official Qwen3-4B Q4_K_M GGUF was downloaded from the pinned Hugging Face revision, verified at 2,497,280,256 bytes with SHA-256 `7485fe6f11af29433bc51cab58009521f205840f5b4ae3a32fa7f92e8534fdf5`, and its Apache-2.0 license digest was verified. The weights remain outside Git.
- `llama-server` ran on the real GGUF with loopback binding. The final evaluator passed 6/6 Arabic/English samples, 6/6 structured tool-schema selections, streaming cancellation, synthetic RAG grounding, session restart, reviewed memory recall, and one real `system_info` Auto-mode call. Evidence: `docs/validation/real-model-4b-final.json`.
- The 0.6B and 1.7B historical reports retain real failures and are not deleted. They show why model size, deterministic temperature, ordered schema properties, and explicit Arabic terminology context matter. The passing 4B smoke suite is not a claim of global model quality.
- `govulncheck` v1.9.0 now reports **No vulnerabilities found** after dependency remediation. Evidence: `docs/validation/govulncheck.txt`.

## Functional limitations

- Model reasoning, Arabic answer quality outside the small rubric, safe tool selection under adversarial content and broad structured-generation success rate remain unmeasured. Programmatic validators remain mandatory regardless of model quality.
- Registered tools are read-only: file read/list, uname, Go version, services, installed Arch packages, local network and recent journal. Privileged repairs, arbitrary commands and remote penetration testing are not implemented.
- CISA connector is implemented; a live attempt returned HTTP 403 from the CISA origin access-control layer (HTTPS tunnel succeeded). OWASP live sync is validated. Source registration is not evidence that every listed reference has been fetched. ETag/Last-Modified conditional sync and richer source relationships are not implemented. Document/feed replacement is transactional; old content survives a failed transaction.
- RAG prefers exact multi-term matches, then uses ranked lexical terms with basic function-word filtering if none match. Semantic paraphrases/synonyms can still miss. The unchanged 60-case synthetic evaluation now retrieves 60/60, which is narrow and not a production security benchmark. Retrieved source links are labeled retrieved references, not proof that all generated claims are supported.
- Summarization is explicit (`summarize SESSION_ID`), not an automatic background process; two levels and original message ranges are implemented. It has no real-model quality evaluation. Mandatory prompt overflow returns an error rather than dropping security instructions.
- Reports are local Markdown/JSON. PDF/DOCX/text-PDF import and PDF export are not implemented. No embedding model exists.
- TUI uses maintained Bubble Tea v1 rather than v2; behavior is tested with compatible Bubbles/Lip Gloss/Glamour versions. Real terminal bidi rendering and KDE Wayland clipboard need target verification. `/copy` reports missing wl-copy rather than pretending success.
- The real 4B report samples llama-server CPU/RSS every 100 ms; it measured about 9.30 end-to-end output tok/s, p50 5.723s, p95 7.157s and sampled peak RSS 5.21GB on Cloud. `llama-bench` separately measured 14.04 generation tok/s at four threads. These values do not predict the i5-1145G7 target. The Go frontend also reports its own CPU/heap/peak RSS; GPU remains N/A. CLI flags must precede subcommands; config file use is explicit.
- The current environment is Debian, not CachyOS. Actual systemd/pacman behavior, privileged authentication, target-device performance and real external-network isolation remain acceptance checks.
- GitHub Actions configuration is committed; a remote workflow run has not been observed. Local tests are the current evidence.

No feature in this list should be silently marked complete. The remaining items require a final run on the user's CachyOS machine: i5-1145G7 throughput/RSS at contexts 2048/4096, KDE/Wayland clipboard, RTL/bidi terminal rendering, Arch `systemd`/`pacman`/`journalctl` behavior, and real external-network isolation. Cloud evidence does not substitute for those target-device checks.
