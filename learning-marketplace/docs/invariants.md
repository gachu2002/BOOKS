# Invariant Matrix

This document turns the project invariants into something actionable.

Each invariant should eventually have:

- a source of truth
- an enforcement mechanism
- a failure drill
- a monitoring or audit signal

## Current invariants

| Invariant | Strength | Current enforcement | Future enforcement / notes | DDIA chapter |
| --- | --- | --- | --- | --- |
| User email is unique | Hard | DB unique constraint on `users.email` | Add API validation and duplicate-creation tests | `Ch. 3`, `Ch. 8` |
| Product slug is unique | Hard | DB unique constraint on `products.slug` | Add publish/update race tests | `Ch. 3`, `Ch. 8` |
| Promo code string is unique | Hard | DB unique constraint on `promo_codes.code` | Add admin conflict tests | `Ch. 3`, `Ch. 8` |
| Payment idempotency key is unique | Hard | DB unique constraint on `payments.idempotency_key` | Add retry-after-timeout tests | `Ch. 5`, `Ch. 8` |
| Order status is valid | Hard | DB check constraint on `orders.status` | Add application-level transition guard table or state machine | `Ch. 3`, `Ch. 8` |
| One order is not captured twice | Hard | Not fully enforced yet | Enforce via idempotency, payment state transitions, and transactional capture logic | `Ch. 8` |
| One entitlement is granted once per purchased item | Hard | Unique constraint on `(user_id, product_id, cohort_id)` | Add async projection/idempotency handling | `Ch. 8`, `Ch. 12` |
| Cohort seats are never oversold | Hard, transaction-sensitive | Not enforced yet | Enforce in checkout transaction with locking or serializable isolation | `Ch. 8` |
| Promo redemptions stay within configured limit | Hard, transaction-sensitive | Not enforced yet beyond schema shape | Enforce in redemption transaction with concurrency tests | `Ch. 8` |
| Search reflects published products | Eventual | Not implemented yet | Outbox plus projection consumer and rebuild process | `Ch. 12`, `Ch. 13` |
| Cache reflects product and cohort source data | Eventual | Not implemented yet | Cache invalidation and drift checks | `Ch. 6`, `Ch. 13` |
| Analytics counters match source orders and payments | Eventual | Not implemented yet | Rebuildable batch reports and stream projections | `Ch. 11`, `Ch. 12` |
| Notification history matches outbox events | Eventual | Not implemented yet | Idempotent consumer plus replay and DLQ support | `Ch. 12`, `Ch. 13` |

## Most important immediate gap

The biggest correctness gap today is this:

- `cohort seats are never oversold`

It is intentionally not solved in the schema yet, because solving it is one of the main `DDIA Chapter 8` learning goals for the project.

## How to use this matrix

Before implementing any feature, decide:

1. Is the invariant `hard` or `eventual`?
2. If hard, is schema enforcement enough or do we need transaction logic?
3. If eventual, what is the source of truth and how will we detect drift?
4. How will we reproduce the failure intentionally?
5. What metric, audit, or test tells us it broke?
