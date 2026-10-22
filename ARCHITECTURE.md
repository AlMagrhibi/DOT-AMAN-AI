# Architecture

Terminal UI → application → context + local retrieval → loopback model provider → validated tool registry → bounded execution → audit → observations.

SQLite owns sessions, original messages, reviewed memory, knowledge provenance and audit records. Transactions preserve last good state. UI never owns policy. Documents and model output are untrusted data; neither can register commands or alter configuration. HTTP inference accepts numeric loopback addresses only, disables proxies and redirects, and bounds requests and responses. Knowledge synchronization is a separate explicit operation.

Use small interfaces and add packages only when used. Final runtime requires Go-built executable and local llama-server/GGUF, never Python.
