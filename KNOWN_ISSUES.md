# Known issues and outstanding validation

## Current Cloud status

- The official Qwen3-4B Q4_K_M GGUF was downloaded from the pinned Hugging Face revision, verified at 2,497,280,256 bytes with SHA-256 `7485fe6f11af29433bc51cab58009521f205840f5b4ae3a32fa7f92e8534fdf5`, and its Apache-2.0 license digest was verified. The weights remain outside Git.
- `llama-server` ran on the real GGUF with loopback binding. The final evaluator passed 6/6 Arabic/English samples, 6/6 structured tool-schema selections, streaming cancellation, synthetic RAG grounding, session restart, reviewed memory recall, and one real `system_info` Auto-mode call. Evidence: `docs/validation/real-model-4b-final.json`.
- The 0.6B and 1.7B historical reports retain real failures and are not deleted. They show why model size, deterministic temperature, ordered schema properties, and explicit Arabic terminology context matter. The passing 4B smoke suite is not a claim of global model quality.
- `govulncheck` v1.9.0 now reports **No vulnerabilities found** after dependency remediation. Evidence: `docs/validation/govulncheck.txt`.

## Functional limitations

- Model reasoning, Arabic answer quality outside the small rubric, safe tool selection under adversarial content and broad structured-generation success rate remain unmeasured. Programmatic validators remain mandatory regardless of model quality.
- Registered tools include read-only diagnostics plus constrained `service_restart` and `package_install`; arbitrary shell, unbounded package expressions, privileged authentication automation and remote penetration testing are not implemented.
- CISA primary and official GitHub mirror sync were exercised live: 1,754 records imported, the second request returned `not-modified`, and CVE search succeeded. OWASP live sync remains validated. Source registration is not evidence that every listed reference has been bulk-fetched. Document/feed replacement is transactional; old content survives a failed transaction.
- RAG combines FTS5/BM25 with optional local Nomic embeddings. The unchanged 60-case lexical fixture and independent 12-query real semantic fixture pass their declared Recall@5 thresholds; they remain narrow and do not prove generated-answer faithfulness. Retrieved links are provenance, not proof of every claim.
- Summaries trigger automatically once a session exceeds the bounded threshold; explicit `summarize SESSION_ID` remains available. Two levels and original message ranges are retained. Real-model summary quality is not yet independently scored. Mandatory prompt overflow returns an error rather than dropping security instructions.
- Reports are local Arabic Markdown plus structured `.security.json` and legacy `.json`. PDF report rendering is conditional in Aman.md and remains unimplemented; PDF/DOCX text ingestion is implemented and tested. Scanned/image-only PDFs remain unsupported.
- TUI uses maintained Bubble Tea v1 rather than v2; behavior is tested with compatible Bubbles/Lip Gloss/Glamour versions. Real terminal bidi rendering and KDE Wayland clipboard need target verification. `/copy` reports missing wl-copy rather than pretending success.
- The real 4B report samples llama-server CPU/RSS every 100 ms; it measured about 9.30 end-to-end output tok/s, p50 5.723s, p95 7.157s and sampled peak RSS 5.21GB on Cloud. `llama-bench` separately measured 14.04 generation tok/s at four threads. These values do not predict the i5-1145G7 target. The Go frontend also reports its own CPU/heap/peak RSS; GPU remains N/A. CLI flags must precede subcommands; config file use is explicit.
- The current environment is Debian, not CachyOS. Actual systemd/pacman behavior, privileged authentication, target-device performance and real external-network isolation remain acceptance checks.
- GitHub Actions configuration is committed; a remote workflow run has not been observed. Local tests are the current evidence.

No feature in this list should be silently marked complete. The remaining items require a final run on the user's CachyOS machine: i5-1145G7 throughput/RSS at contexts 2048/4096, KDE/Wayland clipboard, RTL/bidi terminal rendering, Arch `systemd`/`pacman`/`journalctl` behavior, and real external-network isolation. Cloud evidence does not substitute for those target-device checks.
