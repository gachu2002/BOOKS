# Learning Marketplace

`learning-marketplace` is a deliberately growing backend system built to study and test the ideas in `Designing Data-Intensive Applications` through real implementation work.

The system starts as a modular monolith in `Go + Postgres`, then gradually adds caching, asynchronous jobs, search, replication, partitioning, analytics, and stream-driven projections.

The product domain is a digital learning marketplace with limited-seat live cohorts.

## Why this system

This domain gives us both soft and hard requirements:

- normal CRUD for products, users, and cohorts
- strict money and inventory constraints during checkout
- eventual-consistent derived systems like search and analytics
- asynchronous delivery like emails, entitlements, and notifications
- real opportunities to induce failures and observe breakage

## Core domain

- `users`
- `products`
- `cohorts`
- `promo_codes`
- `orders`
- `order_items`
- `payments`
- `entitlements`
- `audit_log`
- `outbox_events`

## Key invariants

These are the first business rules we want to preserve as the system grows:

- a user email is unique
- a product slug is unique
- a promo code string is unique
- a payment idempotency key is unique
- an order has valid state transitions
- a user should receive an entitlement at most once for the same purchased item
- a cohort must never sell more seats than its capacity

Some of these are easy to enforce with local constraints. Some require transaction design. Some will later require careful distributed design.

## DDIA roadmap

The project is meant to be built chapter by chapter:

1. `Phase 0-1`: domain model, schema, CRUD, constraints
2. `Phase 2`: indexes, query plans, and read paths
3. `Phase 3`: checkout correctness, idempotency, and transaction isolation
4. `Phase 4`: Redis caching and derived views
5. `Phase 5`: search indexing and outbox-driven projections
6. `Phase 6`: background jobs and retry safety
7. `Phase 7`: Postgres replication and replica lag drills
8. `Phase 8`: partitioning and hot-spot experiments
9. `Phase 9`: coordination, leases, and fencing
10. `Phase 10`: batch analytics and reproducible jobs
11. `Phase 11`: CDC and streaming projections
12. `Phase 12`: failure lab and operational drills

If your goal is learning DDIA with this repo, only use these docs:

- `docs/ddia-map.md`: strict chapter -> real project flow -> code -> command mapping
- `docs/ddia-status.md`: audit truth for which chapters the real project teaches well today
- `docs/system.md`: what the system is and what constraints it has
- `docs/invariants.md`: what must stay correct and what may lag

The main DDIA path is intentionally project-first:

- it excludes standalone chapter labs, synthetic demo packages, and lab-only endpoints
- if a chapter is weak in the real product, the map should say so instead of inventing coverage

Minimal chapter loop:

1. `ddia-status <chapter>`
2. `ddia-map <chapter>`
3. read only the mapped files
4. run only the mapped command or endpoint
5. write `5` lines of notes: source of truth, derived state, trade-off, failure mode, missing coverage

Use `ddia-audit <chapter>` only if the chapter is still `not audited` or the repo changed a lot.
Use `ddia-fix <chapter>` only when you want to improve a real product path.

## Current scope

This first cut includes:

- project scaffolding
- local Docker Compose for Postgres
- Phase 1 CRUD API for users, products, cohorts, and promo codes
- transactional cohort checkout with seat-capacity enforcement and payment idempotency
- integration tests for oversell prevention, idempotent retries, and promo exhaustion races
- optional read-replica wiring plus a stale-read lab for entitlements
- standalone projector plus rebuildable `user_library_projection` derived state
- etcd-backed lease lab plus fenced writes against protected counters
- rebuildable batch analytics for daily revenue and cohort fill
- Prometheus metrics endpoint and request logging middleware
- GitHub Actions CI for build, test, and lint
- Chapter 4 storage/retrieval lab with keyset pagination, search, and query plans
- initial SQL schema for the core business entities
- phase planning docs to guide the next implementations

## Run locally

1. Copy `.env.example` to `.env` if you want custom values.
2. Start Postgres:

```bash
docker compose up -d postgres
```

3. Run the API:

```bash
go run ./cmd/api
```

4. Open the health endpoint:

```text
http://localhost:8080/healthz
```

5. Open the metrics endpoint:

```text
http://localhost:8080/metrics
```

## Project layout

```text
learning-marketplace/
  cmd/api/              # application entrypoint
  db/migrations/        # SQL schema and later migrations
  docs/                 # architecture, phases, invariants, failure drills
  internal/app/         # app wiring
  internal/config/      # environment config
  internal/httpapi/     # HTTP routes and handlers
  scripts/              # helper scripts added as the project grows
```

## Immediate next step

Add projector lag visibility and then decide whether to evolve the outbox into a broker-backed stream for stronger `DDIA Chapter 12` coverage.
