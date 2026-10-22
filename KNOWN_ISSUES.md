# Known issues and outstanding validation

## Current Cloud blockers (resumed evaluation)

- Official Hugging Face model metadata and pinned Apache-2.0 license are accessible. The GGUF transfer redirects to **us.aws.cdn.hf.co**, which returned a proxy CONNECT `403 Forbidden`. This newly observed exact destination was added to the configuration draft; it needs review/save/publication. The user reported publication of the earlier settings; the current API observation remains stale/unknown, so actual service responses are used as evidence.
- No GGUF bytes passed integrity verification and no real model started. Chat/Schema/Streaming/model-backed RAG and memory/performance results remain **unrun**. `configs/model-lock.json` pins publisher revision, expected size/digest and license; it does not claim a completed download.
- The previous Go vulnerability network blocker is resolved. Old scanner v1.1.4 crashed against Go 1.27; v1.9.0 and dependency fixes now produce **No vulnerabilities found** in the detailed source scan. Evidence: docs/validation/govulncheck.txt.
- Real-model evaluator and resource runner are compiled/syntax checked, but their real-GGUF execution remains pending. They never fall back to fixtures and preserve failed checks.

## Functional limitations

- Model reasoning, Arabic answer quality, safe tool selection under adversarial content and real structured-generation success rate are unmeasured. Programmatic validators remain mandatory regardless of model quality.
- Registered tools are read-only: file read/list, uname, Go version, services, installed Arch packages, local network and recent journal. Privileged repairs, arbitrary commands and remote penetration testing are not implemented.
- CISA connector is implemented; a live attempt returned HTTP 403 from the CISA origin access-control layer (HTTPS tunnel succeeded). OWASP live sync is validated. Source registration is not evidence that every listed reference has been fetched. ETag/Last-Modified conditional sync and richer source relationships are not implemented. Document/feed replacement is transactional; old content survives a failed transaction.
- RAG prefers exact multi-term matches, then uses ranked lexical terms with basic function-word filtering if none match. Semantic paraphrases/synonyms can still miss. The unchanged 60-case synthetic evaluation now retrieves 60/60, which is narrow and not a production security benchmark. Retrieved source links are labeled retrieved references, not proof that all generated claims are supported.
- Summarization is explicit (`summarize SESSION_ID`), not an automatic background process; two levels and original message ranges are implemented. It has no real-model quality evaluation. Mandatory prompt overflow returns an error rather than dropping security instructions.
- Reports are local Markdown/JSON. PDF/DOCX/text-PDF import and PDF export are not implemented. No embedding model exists.
- TUI uses maintained Bubble Tea v1 rather than v2; behavior is tested with compatible Bubbles/Lip Gloss/Glamour versions. Real terminal bidi rendering and KDE Wayland clipboard need target verification. `/copy` reports missing wl-copy rather than pretending success.
- CPU/RSS metrics describe the Go frontend only; RSS is process lifetime peak. llama-server resource use and target GPU are not inferred; GPU shows N/A. CLI flags must precede subcommands; config file use is explicit.
- The current environment is Debian, not CachyOS. Actual systemd/pacman behavior, privileged authentication, target-device performance and real external-network isolation remain acceptance checks.
- GitHub Actions configuration is committed; a remote workflow run has not been observed. Local tests are the current evidence.

No feature in this list should be silently marked complete. Continue development after the concrete model/network prerequisite is resolved.
