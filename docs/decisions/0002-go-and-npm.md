# 0002: Go binary, distributed through npm

**Decision:** written in Go, and published to npm as a launcher plus one binary package per platform. Dependencies are vendored.

**Why:** the maintainer prefers Go, and a binary has no runtime requirement. npm provides the `npx` install MCP clients expect, and a package type the MCP Registry accepts.

**Trade-off:** `npx` itself needs Node, and each release publishes six packages.

**Alternatives:** Docker images, which can't easily read local files.
