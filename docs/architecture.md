# Architecture

    contacts.csv ─> load ─> normalize ─> find ─> CLI and MCP tools

**Loading:** finds the header in the first 20 rows, handles Latin-1, BOMs and common delimiters, and reports rows with nothing to match on.

**Finding:** rows sharing an email, phone, name, company or email domain go in a block, and only rows within a block are compared, so large files stay fast. Oversized blocks are skipped, with a warning for emails and phones, because they're usually placeholders.

**Evidence:** each comparison produces an edge marked same, likely or review. Same and likely edges form groups through union-find. A group is `same` only if same-level edges alone connect all its rows. Review edges never join rows, which stops a weak link from chaining two people together.

**Privacy:** no network code, which the test suite checks. Verification runs with the network off.

**Packaging:** npm installs one small launcher plus the binary for the platform, through optional dependencies, the same approach esbuild uses. The MCP Registry needs a supported package type, and npm fits `npx` users.

See `docs/decisions/`.
