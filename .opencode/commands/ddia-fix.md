---
description: Audit and improve one official DDIA chapter
agent: ddia-maintainer
subtask: true
---
Load the `ddia-review` skill and follow it exactly.

Audit official DDIA chapter `$ARGUMENTS` against the `learning-marketplace` project.

If the chapter grades below `A`:
- make the smallest useful change needed to improve learning value
- prefer improving existing real project paths over docs-only changes
- tests are allowed only when they clarify existing real project behavior
- do not add standalone chapter labs, synthetic demo packages, or lab-only endpoints
- avoid unrelated cleanup

If the chapter already grades `A` or better:
- do not change product code

Requirements:
- Always refresh `@learning-marketplace/docs/ddia-status.md` and `@learning-marketplace/docs/ddia-map.md` after the audit.
- If you change product code, run the smallest relevant verification commands.
- If `$ARGUMENTS` is `all`, work chapter-by-chapter across official DDIA chapters `1` to `12` and stop only when every failing chapter is fixed or clearly blocked.
- Read enough of the requested chapter first from `@distributed-systems/Designing Data-Intensive Applications, 2nd Edition (Martin Kleppmann, Chris Riccomini) (z-library.sk, 1lib.sk, z-lib.sk).pdf`.
- Return a markdown table only.

Return columns:
- `Ch`
- `Title`
- `Before`
- `After`
- `Changed`
- `Remaining Weak`
- `Verify`
