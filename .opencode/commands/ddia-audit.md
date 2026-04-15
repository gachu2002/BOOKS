---
description: Audit one official DDIA chapter against learning-marketplace
agent: ddia-auditor
subtask: true
---
Load the `ddia-review` skill and follow it exactly.

Audit official DDIA chapter `$ARGUMENTS` against the `learning-marketplace` project.

Grade the chapter against the workflow's `A` passing target.

Requirements:
- Always refresh `@learning-marketplace/docs/ddia-status.md` and `@learning-marketplace/docs/ddia-map.md` in place.
- If `$ARGUMENTS` is `all`, audit every official DDIA chapter from `1` to `12`.
- If a chapter row already exists, refresh it rather than duplicating it.
- Do not change product code.
- Read enough of the requested chapter first from `@distributed-systems/Designing Data-Intensive Applications, 2nd Edition (Martin Kleppmann, Chris Riccomini) (z-library.sk, 1lib.sk, z-lib.sk).pdf`.
- Inspect project code first, supporting docs second, real tests/endpoints/commands third.
- Keep the main DDIA path strict and project-only.
- Do not count standalone chapter labs, synthetic demo packages, or lab-only endpoints in the main mapping.
- Return a markdown table only.

Return columns:
- `Ch`
- `Title`
- `Before`
- `After`
- `Verdict`
- `Learn Now?`
- `Strongest`
- `Weakest`
- `Updated`
