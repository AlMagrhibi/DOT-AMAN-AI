# Threat model

Assets: local files, credentials, conversation history, model configuration and execution authority. Threats: prompt injection in documents/logs, malicious model suggestions, shell injection, traversal/symlink escapes, oversized outputs, SSRF and supply-chain compromise.

Controls: registered tools only; typed strict arguments; executable/argv separation; confined file access; time/output limits and cancellation; audit rejection before execution; local-only inference with no redirects/proxy; private application directories; no telemetry; manual HTTPS source sync with fixed destinations, same-host redirects, conditional validators, size validation and public-IP DNS pinning when direct. Codex Cloud's explicit egress proxy is honored only for those fixed built-in destinations. Mutating tools are limited to explicitly named systemd service restart and package names, and remain subject to mode/approval. Never treat retrieved text or tool observations as system instructions.

Default mode: analysis. Confirm requires explicit user action; auto still validates every tool. Root execution is not required. No secrets or actual user databases belong in Git.

Target-device Wayland clipboard, CPU performance and Arch service behavior require separate acceptance checks.
