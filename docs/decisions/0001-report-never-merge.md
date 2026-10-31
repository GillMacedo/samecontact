# 0001: Report duplicates and never merge them

**Decision:** the tool reports groups with evidence. It has no delete or merge action, and `--out` only adds columns to a new file.

**Why:** merging two different customers loses data and is hard to notice, while checking a few extra rows is cheap.

**Alternatives:** automatic merging above a score, which would hide its mistakes.
