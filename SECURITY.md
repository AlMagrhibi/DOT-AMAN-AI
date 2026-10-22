# Threat model

Assets: local files, credentials, conversation history, model configuration and execution authority. Threats: prompt injection in documents/logs, malicious model suggestions, shell injection, traversal/symlink escapes, oversized outputs, SSRF and supply-chain compromise.

Controls: registered tools only; typed strict arguments; executable/argv separation; confined file access; time/output limits and cancellation; audit rejection before execution; local-only inference with no redirects/proxy; private application directories; no telemetry; manual HTTPS source sync with destination and size validation. Never treat retrieved text as system instructions.

Default mode: analysis. Confirm requires explicit user action; auto still validates every tool. Root execution is not required. No secrets or actual user databases belong in Git.

Target-device Wayland clipboard, CPU performance and Arch service behavior require separate acceptance checks.
