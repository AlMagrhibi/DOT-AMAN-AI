# Dependencies

- Go 1.27.3, BSD-3-Clause. Cloud archive: go1.27.3.linux-amd64.tar.gz, SHA-256 `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`, verified against official go.dev release metadata.
- github.com/mattn/go-sqlite3 v1.14.32, MIT; bundled SQLite is public domain. Requires CGO and C compiler; compile with `-tags sqlite_fts5`. Selected after modernc distribution redirected to a network-denied host. No TLS/checksum checks were bypassed.
- github.com/pelletier/go-toml/v2 v2.2.4, MIT; strict TOML configuration.

Go module checksums are in go.sum. No cloud AI SDK.
