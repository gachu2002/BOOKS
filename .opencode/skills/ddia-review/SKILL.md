---
name: ddia-review
description: Review how well learning-marketplace covers an official DDIA chapter and keep the DDIA workflow tables current.
compatibility: opencode
---

## Scope

Use the official numbered chapters of `Designing Data-Intensive Applications`.

Local book source:

| Source | Path |
| --- | --- |
| DDIA PDF | `distributed-systems/Designing Data-Intensive Applications, 2nd Edition (Martin Kleppmann, Chris Riccomini) (z-library.sk, 1lib.sk, z-lib.sk).pdf` |

Use this local book file as the default chapter source during `ddia-audit` and `ddia-fix`.
Do not rely on memory or web summaries when the local file is available.

| Ch | Title |
| --- | --- |
| 1 | Reliable, Scalable, and Maintainable Applications |
| 2 | Data Models and Query Languages |
| 3 | Storage and Retrieval |
| 4 | Encoding and Evolution |
| 5 | Replication |
| 6 | Partitioning |
| 7 | Transactions |
| 8 | The Trouble with Distributed Systems |
| 9 | Consistency and Consensus |
| 10 | Batch Processing |
| 11 | Stream Processing |
| 12 | The Future of Data Systems |

Do not invent chapters `13` or `14`.
Do not use custom chapter numbering.

## Workflow Files

Only two DDIA workflow docs exist.

| File | Purpose |
| --- | --- |
| `learning-marketplace/docs/ddia-status.md` | audit truth |
| `learning-marketplace/docs/ddia-map.md` | project-to-book map |

Keep both files table-only.

`learning-marketplace/docs/ddia-status.md` columns must stay:

| Ch | Title | Target | Grade | Verdict | Learn Now? | Strongest Evidence | Weakest Gap | Smallest Next Fix | Verify | Updated |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |

`learning-marketplace/docs/ddia-map.md` columns must stay:

| Ch | Title | Related? | DDIA Parts | Project Parts | Read | Run/Inspect | Follow | Limits |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |

## Command Modes

| Command | Behavior |
| --- | --- |
| `ddia-status` | read-only; show rows from `docs/ddia-status.md` only |
| `ddia-map` | read-only; show rows from `docs/ddia-map.md` only |
| `ddia-audit` | audit one official DDIA chapter; refresh both workflow docs; do not change product code |
| `ddia-fix` | audit first; if the chapter grades below `A`, make the smallest useful improvement; refresh both workflow docs |

Read-only commands must not inspect code, the book, or supporting docs.

## Audit Standard

The passing target for this workflow is `A`.

This workflow is strict and project-first.

Do not pass a chapter on docs alone.
Do not pass a chapter because of standalone chapter labs, synthetic demo packages, or lab-only endpoints.

The main DDIA learning path must use the real project as it exists today.

A chapter is good enough only if its important concepts are embodied strongly enough in the real repo that a learner can:

1. map the book concept to exact project files
2. understand the implementation shape
3. see at least some real constraints, trade-offs, or failure modes
4. run or inspect relevant real commands, tests, or endpoints when appropriate

If the real project does not teach a chapter well yet, mark it `partial`, `weak`, or `missing` honestly.
Do not invent coverage to make the grade pass.

## Audit Workflow

For each requested chapter:

1. Read enough of the requested chapter from the local DDIA PDF in `distributed-systems/` to identify its main concepts and subsections.
2. Inspect project code first.
3. Inspect supporting docs second:
   - `learning-marketplace/docs/system.md`
   - `learning-marketplace/docs/invariants.md`
4. Use real tests, commands, and endpoints as runnable evidence.
5. Exclude standalone chapter labs, synthetic demo packages, and lab-only endpoints from the main DDIA map and from grading.
6. Decide which concepts are `must learn`, `nice to learn`, and `skip for now` for this repo.
7. Score the chapter with the rubric below.
8. Refresh the matching row in `learning-marketplace/docs/ddia-status.md`.
9. Refresh the matching row in `learning-marketplace/docs/ddia-map.md`.
10. If running in fix mode and the chapter grades below `A`, improve the smallest useful thing in an existing real project path, verify it, then rerun the audit.

If an existing row reflects stale or wrong chapter numbering, correct it instead of carrying it forward blindly.

## Evidence Strength

| Level | Meaning |
| --- | --- |
| `strong` | real code exists; runnable evidence exists; trade-off or failure mode is visible |
| `partial` | some code exists, but runnable or failure evidence is thin |
| `weak` | mostly docs or very indirect code evidence |
| `missing` | not really represented |

Treat these as excluded from main-path evidence unless the user explicitly asks for an appendix or supplemental mode:

- chapter-numbered labs such as `cmd/ch*lab`
- synthetic teaching packages such as `internal/*/lab.go`
- lab-only endpoints such as `/lease-lab/*`
- tests that exercise only synthetic demo code rather than real project flows

## Scoring Rubric

Concept weights:

| Kind | Weight |
| --- | --- |
| `must learn` | `3` |
| `nice to learn` | `1` |
| `skip for now` | `0` |

Evidence scores:

| Level | Score |
| --- | --- |
| `strong` | `1.00` |
| `partial` | `0.60` |
| `weak` | `0.25` |
| `missing` | `0.00` |

Weighted score: `sum(weight * score) / sum(weight)`

Grade mapping:

| Grade | Threshold |
| --- | --- |
| `A` | `0.90+` |
| `A-` | `0.85+` |
| `B+` | `0.78+` |
| `B` | `0.70+` |
| `C+` | `0.62+` |
| `C` | `0.55+` |
| `not good enough` | `< 0.55` |

Use status verdicts:

| Verdict | Meaning |
| --- | --- |
| `pass` | chapter meets or beats the `A` target |
| `needs work` | chapter was audited but is below the `A` target |
| `not audited` | chapter has not been audited yet |

## Fix Mode

If running in fix mode and the chapter grades below `A`:

1. improve the smallest useful thing
2. improve an existing real project path before touching docs
3. tests are allowed only when they make an existing real project behavior easier to learn
4. do not add standalone chapter labs, synthetic teaching packages, or lab-only endpoints
5. if the real project still does not cover the chapter strongly, leave the chapter below `A` and document the gap honestly
6. avoid unrelated cleanup
7. run the smallest relevant verification commands
8. rerun the audit
9. refresh both workflow tables with the new result

## Output Style

Return markdown tables only.

`ddia-audit` rows must include:

| Ch | Title | Before | After | Verdict | Learn Now? | Strongest | Weakest | Updated |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |

`ddia-fix` rows must include:

| Ch | Title | Before | After | Changed | Remaining Weak | Verify |
| --- | --- | --- | --- | --- | --- | --- |

Base directory for this skill: file:///home/nguyenlt/repos/BOOK/.opencode/skills/ddia-review
