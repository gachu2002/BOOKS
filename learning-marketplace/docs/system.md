# System Design

## Product concept

The system sells digital learning products:

- self-paced digital products
- limited-seat live cohorts
- bundles and promotions later

The main learning value comes from the live cohort workflow, because it introduces a strict inventory constraint: seats are scarce and must not be oversold.

## Actors

- student buyer
- admin/operator
- payment provider
- background worker
- analytics consumer
- search projection consumer

## Main user flows

### Browse and discover

- user lists products
- user filters products
- user views a product page

Later, this flow will use cache and search.

### Buy a cohort seat

- user starts checkout
- system validates cohort state and promo code
- system creates pending order
- system records payment intent
- system captures payment
- system grants entitlement
- system decrements available cohort capacity
- system emits async events for email and analytics

### Access purchased content

- user lists entitlements
- user opens purchased product
- user downloads assets or joins cohort details

### Admin operations

- create product
- create cohort
- publish product
- issue promo code
- view seat utilization and order summaries

## Boundaries

### System of record

Initially:

- Postgres is the system of record for operational data.

Later:

- Redis becomes a derived cache.
- OpenSearch becomes a derived search index.
- analytics tables become derived OLAP data.
- outbox and CDC streams become the write path for projections.

## Invariants by strength

### Hard invariants

These must hold immediately:

- `users.email` is unique
- `products.slug` is unique
- `promo_codes.code` is unique
- `payments.idempotency_key` is unique
- one order cannot be captured twice
- one entitlement cannot be granted twice for the same purchase

### Hard but transaction-sensitive invariants

These are where we want to learn from DDIA:

- a cohort cannot oversell seats
- a promo code cannot exceed max redemptions
- checkout retries must not duplicate money movement
- reads after successful payment must not expose contradictory order/payment state

### Eventual invariants

These may lag, but must converge:

- search index matches published product state
- cache matches source-of-truth product and cohort state
- analytics counters match source order and payment events
- notification history matches outbox events

## Likely failure scenarios we want to reproduce

- two concurrent users buy the last cohort seat
- same checkout request retries after timeout
- payment succeeds but notification job fails
- Postgres primary is ahead of replica and a critical read hits the replica
- dual-write bug between Postgres and search index
- lagging projection causes stale entitlement state
- hot cohort creates write hot spot under high traffic
- paused worker still thinks it owns a lease and performs duplicate work

## Technical direction

### Initial architecture

- Go HTTP API
- Postgres primary database
- Docker Compose for local development
- modular monolith layout

### Incremental additions

- Redis for cache-aside and derived counters
- outbox table plus worker
- Redpanda or Kafka for durable event transport
- OpenSearch for search
- Postgres replica(s) for lag and read routing experiments
- etcd for lease/leadership experiments
- DuckDB and SQL-based batch analytics first

## Non-goals for the first phase

- microservices
- distributed transactions across different engines
- complex frontend
- cloud deployment
- autoscaling

We want a clean baseline first, then introduce complexity only when it teaches a DDIA concept.
