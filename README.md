# samecontact: find duplicate customers and contacts in CSV files

[![npm](https://img.shields.io/npm/v/samecontact)](https://www.npmjs.com/package/samecontact) [![CI](https://github.com/GillMacedo/samecontact/actions/workflows/ci.yml/badge.svg)](https://github.com/GillMacedo/samecontact/actions/workflows/ci.yml) [![MIT license](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Clean up a CRM export, customer list or mailing list by finding rows that are the same person, even when the name, email or phone is written differently. For small businesses, sales teams and marketers.**

Most customer lists contain the same person several times: "Robert Jones" and "Bob Jones", or `Ana.Silva+news@` and `ana.silva@`. Duplicates waste mailing costs, annoy customers and overstate your numbers, while a simple dedupe in a spreadsheet misses most of them. `samecontact` compares emails, phone numbers, names and companies, then says how sure it is and why. It never merges anything itself. It's an MCP server and a single Go binary, with no account, no API key and no network access.

## What you can ask your agent

- "Find duplicates in my customer list."
- "How many unique customers do I really have?"
- "Clean up this HubSpot or Salesforce export before I import it."
- "Which contacts in this mailing list are the same person?"

## Install

As a Claude Code plugin, which adds the MCP server and a skill that tells Claude when to use it. You need Node, which runs the server through `npx`:

    claude plugin marketplace add GillMacedo/samecontact
    claude plugin install samecontact@samecontact-marketplace

For other MCP clients:

```json
{
  "mcpServers": {
    "samecontact": {
      "command": "npx",
      "args": ["-y", "samecontact", "serve"]
    }
  }
}
```

The npm install covers macOS and Linux. On Windows, use `go install` for now.

Or build the binary with Go 1.27 or newer:

    go install github.com/GillMacedo/samecontact@latest

## Usage

Try it on the sample file in `examples/`. All the people in it are made up.

    $ go run . find examples/contacts.csv
    SAME PERSON (2)
      rows 3, 4  Ana Silva / Silva, Ana  [same email; same name and company]
      rows 5, 6  Robert Jones / Bob Jones  [same phone and matching name]
    LIKELY: check these (1)
      rows 7, 8  Maria Costa  [same name and company]
    REVIEW: a person should decide (2)
      rows 11, 12  John Smith / John Smith  [same name, different companies]
      rows 13, 14  Lena Fischer / Omar Haddad  [same email, different names: may be a shared address]
    SKIPPED row 16: no name, email or phone
    13 contacts, 3 duplicate groups, 3 extra rows, 2 pairs to review

With an installed copy, run `samecontact find contacts.csv`. Add `--out marked.csv` to write a copy with `dup_group` and `dup_level` columns, which you can sort in a spreadsheet. Your original file is never changed. Add `--json` for machine-readable output.

## How it decides

| Level | Evidence |
|---|---|
| **Same person** | Same email with matching names, same phone with matching names, or an identical row |
| **Likely** | Same name and company, same name and work email domain, or same phone with one name missing |
| **Review** | One email or phone under different names, the same name alone, or a similar name at the same company |

It understands nicknames, initials, accents, name order, small typos, phone formats, Gmail dots and `+tags`. Addresses like `info@` and free domains like Gmail don't count as evidence of the same person.

## MCP tools

| Tool | What it does |
|---|---|
| `contacts_find_duplicates` | Returns groups and review pairs, with a limit for large files |
| `contacts_inspect_csv` | Shows detected columns and row counts, without row contents |

Both are read-only. `contacts_find_duplicates` returns names from your file to the MCP client, so they reach whichever AI model you use. The CLI keeps everything on your machine.

## FAQ

**How do I remove duplicate contacts from a CSV?** Run `samecontact find contacts.csv --out marked.csv`, review the groups, then delete or merge rows in your CRM or spreadsheet. The tool doesn't delete anything.

**Does it work with HubSpot, Salesforce or Mailchimp exports?** It reads CSV and recognizes common column names, and you can name others with `--columns email=Mail,name=Who`. It hasn't been tested on each product's real exports yet.

**Is my customer data uploaded?** The tool has no network code. Through MCP, results go to whichever AI model your client uses.

## Limits

- It finds candidates, and a person should check them before merging anything.
- Name matching is built for Western names and common nicknames, so it will miss some variations in other naming systems.
- Phones compare on their last 10 digits, so two numbers in different countries could collide.
- It reads CSV only, up to 50 MB.
- So far it has been tested on made-up data, not real exports. If yours fails, please open an issue with the column headers, not your data.

## Contributing

Contributions are welcome, from people and from people working with AI agents. See [CONTRIBUTING.md](CONTRIBUTING.md). Good first issues:

- Add column names from more CRMs and languages
- Add more nicknames, including non-English ones
- Add an XLSX reader
- Add a `merge` command that suggests one combined row per group
- Add a plugin icon

### How pull requests work

1. **Open a PR.** Fork the repo, work on a branch, and keep it to one change. Write a title that says what changes for users, because it becomes the changelog line: `feat:` goes under Added and `fix:` under Fixed.
2. **Automatic check.** Code changes need tests, and `make test` must pass. Changes over 30 lines need a linked open issue (`Closes #<number>`), and a PR can have at most 800 changed lines. If something is missing, a comment tells you what to fix.
3. **Review.** Claude reviews each PR automatically. Code PRs also get a security review and are tested in a sandbox. The review then merges the PR or asks for changes. PRs that change CI, agent settings, dependencies or build files go to the maintainer, as does anything the security review doesn't clear.
4. **Release.** A merge that changes what users install becomes a new version within about 20 minutes, with a GitHub Release and, if the project has one, its npm package. Docs and CI changes wait for the next release.

## AI disclosure

This project was largely generated with Claude Code and reviewed by its maintainer.

## About me

I'm Giulliano Macedo, and I build developer tools at GoTech Solutions. Follow along or get in touch:

- GitHub: [GillMacedo](https://github.com/GillMacedo)
- LinkedIn: [giulliano](https://www.linkedin.com/in/giulliano/)
- X: [@giumacedo](https://x.com/giumacedo)

If `samecontact` cleaned up your list, a star helps others find it.

---

MIT licensed.

Built by [Giulliano Macedo](https://github.com/GillMacedo), GoTech Solutions · [LinkedIn](https://www.linkedin.com/in/giulliano/) · [X](https://x.com/giumacedo)
