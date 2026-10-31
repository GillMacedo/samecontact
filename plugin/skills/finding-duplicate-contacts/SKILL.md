---
name: finding-duplicate-contacts
description: Finds rows in a contact, customer, lead or mailing list CSV that are probably the same person. Use when the user wants to dedupe or clean a CRM export or customer list, merge contacts, or asks how many unique customers they have.
---

# Finding duplicate contacts

Call `contacts_find_duplicates` instead of comparing rows yourself, because long lists are easy to get wrong. If it can't find columns, the error says how to name them, and `contacts_inspect_csv` shows what it detected.

## Reporting results
- Lead with the counts: contacts, duplicate groups, extra rows and pairs to review.
- `same` groups have strong evidence, such as one email with matching names.
- `likely` groups need a quick check.
- `review` pairs are for the user to decide. Don't merge them yourself.
- Report `warnings` and `skipped_rows`, so nothing is silently left out.

Results contain people's names. Don't send them anywhere else, and don't delete or merge records in any system without the user's approval.
