# DDIA Map

Read this first when using the project with `Designing Data-Intensive Applications`.

This is the exact map from the book to this repo.

## Keep only these docs open

1. `docs/ddia-map.md`
2. `docs/system.md`
3. `docs/invariants.md`

## How to use this project while reading the book

For each chapter:

1. Read the chapter in the book.
2. Open the matching section below.
3. Read the listed files.
4. Run the listed command or endpoint.
5. Ask:
   - what is the invariant?
   - what can fail?
   - what trade-off did this project choose?

## Chapter 1

Status: `strong`

Read:
- `README.md`
- `docs/system.md`
- `internal/app/app.go`
- `internal/config/config.go`

Look for:
- source of truth vs derived systems
- why the project starts as a modular monolith
- what parts are intentionally delayed

## Chapter 2

Status: `partial`

Read:
- `docs/system.md`
- `docs/invariants.md`

Look for:
- what must be correct immediately
- what may be eventually consistent
- what the system optimizes for: correctness first on money and seats

## Chapter 3

Status: `strong`

Read:
- `db/migrations/001_initial_schema.sql`
- `internal/store/types.go`
- `internal/store/users.go`
- `internal/store/products.go`
- `internal/store/cohorts.go`
- `internal/store/promo_codes.go`

Look for:
- relational modeling
- keys, foreign keys, unique constraints
- why JSON metadata is not used for core invariants

## Chapter 4

Status: `strong`

Read:
- `db/migrations/001_initial_schema.sql`
- CRUD list methods in `internal/store/*.go`
- `internal/httpapi/users.go`
- `internal/httpapi/products.go`
- `internal/httpapi/cohorts.go`
- `internal/httpapi/promo_codes.go`

Run:
- `go run ./cmd/api`

Look for:
- pagination
- list query ordering
- indexes supporting OLTP reads
- how query shape and index shape fit together

## Chapter 5

Status: `partial`

Read:
- request/response structs in `internal/httpapi/`
- `internal/store/checkout.go`
- `internal/projector/user_library.go`

Look for:
- payload shapes
- outbox event payloads
- how replay depends on stable event meaning

## Chapter 6

Status: `strong`

Read:
- `internal/store/entitlements.go`
- `internal/httpapi/entitlements.go`
- `cmd/api/main.go`
- replica config in `internal/config/config.go`

Run:
- `docker compose -f docker-compose.replication.yml up -d`
- `go run ./cmd/api`
- `GET /v1/users/{userID}/entitlements?consistency=strong`
- `GET /v1/users/{userID}/entitlements?consistency=eventual`

Look for:
- strong vs eventual reads
- stale replica reads
- read-after-write problems

## Chapter 7

Status: `weak`

Read:
- `docs/system.md`
- `docs/invariants.md`

Look for:
- likely hot spots: cohorts, orders, outbox events
- what would need sharding later

Note:
- true distributed sharding is not implemented yet

## Chapter 8

Status: `strong`

Read:
- `internal/store/checkout.go`
- `internal/store/errors.go`
- `internal/store/checkout_integration_test.go`
- `docs/invariants.md`

Run:
- `go test ./internal/store -run Checkout`

Look for:
- transaction boundary
- `SELECT ... FOR UPDATE`
- idempotency key handling
- oversell prevention
- promo exhaustion race

This is one of the best parts of the repo.

## Chapter 9

Status: `strong`

Read:
- `internal/httpapi/entitlements.go`
- `internal/projector/user_library.go`
- `internal/httpapi/coordination.go`

Look for:
- stale reads
- lagging projections
- zombie worker shape of failure

## Chapter 10

Status: `strong`

Read:
- `internal/coordination/lease.go`
- `internal/store/protected_counter.go`
- `internal/store/protected_counter_integration_test.go`
- `internal/httpapi/coordination.go`

Run:
- `docker compose -f docker-compose.coordination.yml up -d`
- `go run ./cmd/api`

Look for:
- leases
- fencing tokens
- stale holder rejection
- why a lease alone is not enough

## Chapter 11

Status: `strong`

Read:
- `internal/analytics/reports.go`
- `internal/analytics/reports_test.go`
- `cmd/batch-reports/main.go`
- `internal/httpapi/reports.go`

Run:
- `go run ./cmd/batch-reports`
- `GET /v1/reports/daily-revenue`
- `GET /v1/reports/cohort-fill`

Look for:
- rebuildable batch reports
- operational vs analytical data
- why reports are derived, not hand-maintained

## Chapter 12

Status: `strong`

Read:
- `internal/projector/user_library.go`
- `internal/projector/user_library_test.go`
- `cmd/projector/main.go`
- `internal/store/projection.go`

Run:
- `PROJECTOR_MODE=once go run ./cmd/projector`
- `PROJECTOR_MODE=rebuild go run ./cmd/projector`

Look for:
- outbox-driven projection
- replay and rebuild
- idempotent derived-state updates

## Chapter 13

Status: `strong`

Read:
- `internal/httpapi/library.go`
- `internal/store/projection.go`
- `internal/store/entitlements.go`
- `internal/projector/user_library.go`

Run:
- `GET /v1/users/{userID}/library?source=truth`
- `GET /v1/users/{userID}/library?source=projection`

Look for:
- source of truth vs projection
- why derived state can lag and still be valid
- rebuildable state as a design choice

## Chapter 14

Status: `weak`

Read:
- `docs/system.md`
- `docs/invariants.md`

Look for:
- what is missing: retention, deletion, PII handling, privacy policy

Note:
- this chapter is not implemented strongly yet

## Best order in this repo

If you want the fastest path to useful learning:

1. `Ch. 1-4`
2. `Ch. 8`
3. `Ch. 6`
4. `Ch. 9-10`
5. `Ch. 11-13`

## Fast commands

- run API: `go run ./cmd/api`
- run tests: `go test ./...`
- run projector once: `PROJECTOR_MODE=once go run ./cmd/projector`
- rebuild projector: `PROJECTOR_MODE=rebuild go run ./cmd/projector`
- rebuild batch reports: `go run ./cmd/batch-reports`

## Fast endpoints

- health: `GET /healthz`
- metrics: `GET /metrics`
- checkout: `POST /v1/checkouts/live-cohorts`
- entitlements: `GET /v1/users/{userID}/entitlements?consistency=strong|eventual`
- library: `GET /v1/users/{userID}/library?source=truth|projection`
- daily revenue: `GET /v1/reports/daily-revenue`
- cohort fill: `GET /v1/reports/cohort-fill`
- lease lab: `POST /v1/lease-lab/counters/{resource}/acquire`
