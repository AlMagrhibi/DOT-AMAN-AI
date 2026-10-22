# Architecture

Terminal UI → application → context + local retrieval → loopback model provider → validated tool registry → bounded execution → audit → observations.

SQLite owns sessions, original messages, reviewed memory, experiment metadata, knowledge provenance, source registry/history and audit records. Transactions preserve last good state. UI never owns policy. Documents, model output and tool observations are untrusted data; neither can register commands or alter configuration. HTTP inference and embeddings accept numeric loopback addresses only, disable proxies and redirects, and bound requests/responses. Knowledge synchronization is a separate explicit operation; CISA/OWASP are the only built-in fetch connectors, while the other references are registered metadata until an explicit connector is added.

Use small interfaces and add packages only when used. Final runtime requires Go-built executable and local llama-server/GGUF, never Python.
