# Known issues and outstanding validation

## External Cloud blockers

- Hugging Face model download returned proxy `403 Forbidden` under running network revision 1. No GGUF was downloaded, no SHA-256/model license was independently inspected, and no true inference, JSON-suggestion quality, TTFT/tok/s or long-context quality result exists. The reference model is a proposal, not an installed component.
- `govulncheck` v1.1.4 installed, but fetching `https://vuln.go.dev/index/modules.json.gz` returned `Forbidden`. Dependency audit remains unrun, not passed.
- Network additions and install/start instructions are saved as a **draft**, not applied or published. In environment settings review/save them and publish the environment, then resume these checks. Added destinations: huggingface.co, cdn-lfs.huggingface.co, cdn-lfs-us-1.hf.co, cas-bridge.xethub.hf.co, storage.googleapis.com, www.cisa.gov, vuln.go.dev. Newly observed download redirect hosts may require specific additions; do not broaden access blindly.

## Functional limitations

- Model reasoning, Arabic answer quality, safe tool selection under adversarial content and real structured-generation success rate are unmeasured. Programmatic validators remain mandatory regardless of model quality.
- Registered tools are read-only: file read/list, uname, Go version, services, installed Arch packages, local network and recent journal. Privileged repairs, arbitrary commands and remote penetration testing are not implemented.
- CISA connector is implemented but live sync is not validated; OWASP live sync is validated. Source registration is not evidence that every listed reference has been fetched. ETag/Last-Modified conditional sync and richer source relationships are not implemented. Document/feed replacement is transactional; old content survives a failed transaction.
- RAG is lexical AND search; semantic paraphrases/synonyms can miss. The 60-case synthetic evaluation is narrow and not a production security benchmark. Retrieved source links are labeled retrieved references, not proof that all generated claims are supported.
- Summarization is explicit (`summarize SESSION_ID`), not an automatic background process; two levels and original message ranges are implemented. It has no real-model quality evaluation. Mandatory prompt overflow returns an error rather than dropping security instructions.
- Reports are local Markdown/JSON. PDF/DOCX/text-PDF import and PDF export are not implemented. No embedding model exists.
- TUI uses maintained Bubble Tea v1 rather than v2; behavior is tested with compatible Bubbles/Lip Gloss/Glamour versions. Real terminal bidi rendering and KDE Wayland clipboard need target verification. `/copy` reports missing wl-copy rather than pretending success.
- CPU/RSS metrics describe the Go frontend only; RSS is process lifetime peak. llama-server resource use and target GPU are not inferred; GPU shows N/A. CLI flags must precede subcommands; config file use is explicit.
- The current environment is Debian, not CachyOS. Actual systemd/pacman behavior, privileged authentication, target-device performance and real external-network isolation remain acceptance checks.
- GitHub Actions configuration is committed; a remote workflow run has not been observed. Local tests are the current evidence.

No feature in this list should be silently marked complete. Continue development after the concrete model/network prerequisite is resolved.
