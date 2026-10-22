# Dependencies

- Go 1.27.3, BSD-3-Clause. Cloud archive: go1.27.3.linux-amd64.tar.gz, SHA-256 `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`, verified against official go.dev release metadata.
- github.com/mattn/go-sqlite3 v1.14.32, MIT; bundled SQLite is public domain. Requires CGO and C compiler; compile with `-tags sqlite_fts5`. Selected after modernc distribution redirected to a network-denied host. No TLS/checksum checks were bypassed.
- github.com/pelletier/go-toml/v2 v2.2.4, MIT; strict TOML configuration.

Go module checksums are in go.sum. No cloud AI SDK.

TUI direct dependencies (full resolved versions in go.mod/go.sum): Bubble Tea v1.3.10, Bubbles v0.21.0, Glamour v0.10.0, Lip Gloss v1.1.1-0.20250404203927-76690c660834, charmbracelet/x/ansi v0.10.1: MIT. The v1 TUI API was chosen for verified compatibility. Glamour adds Goldmark (MIT), Chroma (MIT) and Bluemonday (BSD-3-Clause) transitively; these are used only for local rendering. No remote Markdown fetching occurs.

Cloud-only build helpers: CMake 3.31.6 (BSD-3-Clause), Ninja 1.11.1.4 distribution / Ninja (Apache-2.0), installed from PyPI over verified HTTPS. Python is not a delivered application runtime dependency. llama.cpp is pinned to `c7fc5094179bf8a9c73c18593db714f8da41a402`, MIT, cloned through existing GitHub proxy authentication. GCC 14.2.0 builds C/C++ dependencies.

Go module integrity verified by `go mod verify`. govulncheck v1.1.4 (BSD-3-Clause) is installed for development; live vulnerability DB access is blocked, so no clean vulnerability audit is claimed. Original upstream licenses remain applicable. This public repository does not yet specify a license for the new application code; public visibility alone does not grant a license.
