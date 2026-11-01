# samecontact

Finds likely duplicate people in contact CSVs. It's a Go MCP server and CLI, with a Claude Code plugin wrapper.

## Architecture
- `internal/contacts/load.go`: reads files and finds columns.
- `internal/contacts/normalize.go`: emails, phones, names and companies.
- `internal/contacts/find.go`: evidence levels and grouping.
- `internal/server`: MCP tools. `main.go`: CLI.
- `npm/bin.js` and `scripts/build-npm.sh`: npm packaging, one binary package per platform.
- `vendor/`: dependencies, committed so tests run with no network.

## Commands

    make test
    make lint    # plugin validation and MCP Inspector
    make demo    # the README example

## Rules
- Never merge or delete anything. The tool reports, and a person decides.
- Weak evidence goes to review, and never into a group.
- Never drop a row silently. Skipped rows are reported.
- Never add network access. `--out` only ever creates a new file.
- Sample data stays obviously made up, using `example.com` addresses and 555 numbers.
- Never write to stdout in server code, because it corrupts JSON-RPC.

## Definition of done
Tests pass, the README example matches, and `ROADMAP.md` is updated if relevant.
