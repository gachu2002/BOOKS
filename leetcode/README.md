# LeetCode CLI

This folder holds Go solutions and a tiny CLI to list and run registered questions.

## Commands

```bash
go run ./cmd/leet list
go run ./cmd/leet show remove-element
go run ./cmd/leet run remove-element
```

## Add a new question

1. Add the solution file under the right category package.
2. Add a runner in `registry.go` with sample cases.
3. Run it through the CLI.

The CLI is intentionally simple. It exists to answer one question quickly:

`what problem am I working on, and can I run a known sample for it?`
