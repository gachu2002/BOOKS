package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// CheckoutCohort is the main Chapter 8 workflow: lock, validate, charge, grant, and emit.
func (s *Store) CheckoutCohort(ctx context.Context, params CheckoutParams) (CheckoutResult, error) {
	existing, found, err := s.findCheckoutByIdempotencyKey(ctx, s.db, params.IdempotencyKey)
	if err != nil {
		return CheckoutResult{}, err
	}
	if found {
		existing.IdempotentReplay = true
		return existing, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("begin checkout transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	cohort, err := lockCohortForCheckout(ctx, tx, params.CohortID)
	if err != nil {
		return CheckoutResult{}, err
	}
	if cohort.Status != "open" {
		return CheckoutResult{}, ErrCohortNotOpen
	}

	product, err := getProductForCheckout(ctx, tx, cohort.ProductID)
	if err != nil {
		return CheckoutResult{}, err
	}
	if product.ProductType != "live_cohort" {
		return CheckoutResult{}, fmt.Errorf("product %s is not a live cohort", product.ID)
	}

	sold, err := countSoldSeats(ctx, tx, cohort.ID)
	if err != nil {
		return CheckoutResult{}, err
	}
	if sold >= cohort.Capacity {
		return CheckoutResult{}, ErrSoldOut
	}

	discountCents := 0
	var appliedPromo *PromoCode
	if params.PromoCode != nil && strings.TrimSpace(*params.PromoCode) != "" {
		promo, discount, err := validateAndLockPromoCode(ctx, tx, strings.TrimSpace(*params.PromoCode), product.PriceCents)
		if err != nil {
			return CheckoutResult{}, err
		}
		discountCents = discount
		appliedPromo = &promo
	}

	subtotal := product.PriceCents
	total := subtotal - discountCents
	if total < 0 {
		total = 0
	}

	now := time.Now().UTC()

	order, err := insertOrder(ctx, tx, params.UserID, subtotal, discountCents, total, product.Currency, params.IdempotencyKey, now)
	if err != nil {
		if replay, handled, replayErr := tryLoadIdempotentReplay(ctx, s.db, params.IdempotencyKey, err); handled {
			if replayErr != nil {
				return CheckoutResult{}, replayErr
			}
			replay.IdempotentReplay = true
			return replay, nil
		}
		return CheckoutResult{}, err
	}

	if err := insertOrderItem(ctx, tx, order.ID, product.ID, cohort.ID, subtotal); err != nil {
		return CheckoutResult{}, err
	}

	payment, err := insertPayment(ctx, tx, order.ID, params.PaymentProvider, params.ProviderPaymentID, total, product.Currency, params.IdempotencyKey)
	if err != nil {
		if replay, handled, replayErr := tryLoadIdempotentReplay(ctx, s.db, params.IdempotencyKey, err); handled {
			if replayErr != nil {
				return CheckoutResult{}, replayErr
			}
			replay.IdempotentReplay = true
			return replay, nil
		}
		return CheckoutResult{}, err
	}

	if appliedPromo != nil {
		if err := insertPromoRedemption(ctx, tx, appliedPromo.ID, order.ID, params.UserID); err != nil {
			return CheckoutResult{}, err
		}
	}

	entitlement, err := insertEntitlement(ctx, tx, params.UserID, product.ID, order.ID, cohort.ID, now)
	if err != nil {
		return CheckoutResult{}, err
	}

	if err := insertAuditLog(ctx, tx, params.UserID, "order", order.ID, "checkout.completed", map[string]any{
		"cohort_id":       cohort.ID,
		"payment_id":      payment.ID,
		"idempotency_key": params.IdempotencyKey,
		"discount_cents":  discountCents,
		"total_cents":     total,
		"seats_remaining": cohort.Capacity - sold - 1,
	}); err != nil {
		return CheckoutResult{}, err
	}

	if err := insertOutboxEvent(ctx, tx, "order", order.ID, "order.paid", map[string]any{
		"order_id":        order.ID,
		"user_id":         params.UserID,
		"product_id":      product.ID,
		"cohort_id":       cohort.ID,
		"payment_id":      payment.ID,
		"entitlement_id":  entitlement.ID,
		"idempotency_key": params.IdempotencyKey,
		"total_cents":     total,
		"currency":        product.Currency,
	}); err != nil {
		return CheckoutResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return CheckoutResult{}, fmt.Errorf("commit checkout transaction: %w", err)
	}

	return CheckoutResult{
		Order:            order,
		Payment:          payment,
		Entitlement:      entitlement,
		Product:          product,
		Cohort:           cohort,
		AppliedPromoCode: appliedPromo,
		SeatsRemaining:   cohort.Capacity - sold - 1,
	}, nil
}

func (s *Store) findCheckoutByIdempotencyKey(ctx context.Context, db queryer, idempotencyKey string) (CheckoutResult, bool, error) {
	return findCheckoutByIdempotencyKey(ctx, db, idempotencyKey)
}

// findCheckoutByIdempotencyKey turns retries into stable reads of prior successful work.
func findCheckoutByIdempotencyKey(ctx context.Context, db queryer, idempotencyKey string) (CheckoutResult, bool, error) {
	const query = `
SELECT
    o.id,
    o.user_id,
    o.status,
    o.currency,
    o.subtotal_cents,
    o.discount_cents,
    o.total_cents,
    o.idempotency_key,
    o.placed_at,
    o.created_at,
    o.updated_at,
    p.id,
    p.order_id,
    p.provider,
    p.provider_payment_id,
    p.status,
    p.amount_cents,
    p.currency,
    p.idempotency_key,
    p.created_at,
    p.updated_at,
    e.id,
    e.user_id,
    e.product_id,
    e.order_id,
    e.cohort_id,
    e.status,
    e.granted_at,
    e.revoked_at,
    e.created_at,
    e.updated_at,
    pr.id,
    pr.slug,
    pr.title,
    pr.description,
    pr.product_type,
    pr.price_cents,
    pr.currency,
    pr.published,
    pr.metadata,
    pr.created_at,
    pr.updated_at,
    c.id,
    c.product_id,
    c.slug,
    c.title,
    c.capacity,
    c.status,
    c.starts_at,
    c.ends_at,
    c.enrollment_opens_at,
    c.enrollment_closes_at,
    c.created_at,
    c.updated_at
FROM payments p
JOIN orders o ON o.id = p.order_id
JOIN entitlements e ON e.order_id = o.id
JOIN products pr ON pr.id = e.product_id
LEFT JOIN cohorts c ON c.id = e.cohort_id
WHERE p.idempotency_key = $1
LIMIT 1`

	var result CheckoutResult
	err := db.QueryRowContext(ctx, query, idempotencyKey).Scan(
		&result.Order.ID,
		&result.Order.UserID,
		&result.Order.Status,
		&result.Order.Currency,
		&result.Order.SubtotalCents,
		&result.Order.DiscountCents,
		&result.Order.TotalCents,
		&result.Order.IdempotencyKey,
		&result.Order.PlacedAt,
		&result.Order.CreatedAt,
		&result.Order.UpdatedAt,
		&result.Payment.ID,
		&result.Payment.OrderID,
		&result.Payment.Provider,
		&result.Payment.ProviderPaymentID,
		&result.Payment.Status,
		&result.Payment.AmountCents,
		&result.Payment.Currency,
		&result.Payment.IdempotencyKey,
		&result.Payment.CreatedAt,
		&result.Payment.UpdatedAt,
		&result.Entitlement.ID,
		&result.Entitlement.UserID,
		&result.Entitlement.ProductID,
		&result.Entitlement.OrderID,
		&result.Entitlement.CohortID,
		&result.Entitlement.Status,
		&result.Entitlement.GrantedAt,
		&result.Entitlement.RevokedAt,
		&result.Entitlement.CreatedAt,
		&result.Entitlement.UpdatedAt,
		&result.Product.ID,
		&result.Product.Slug,
		&result.Product.Title,
		&result.Product.Description,
		&result.Product.ProductType,
		&result.Product.PriceCents,
		&result.Product.Currency,
		&result.Product.Published,
		&result.Product.Metadata,
		&result.Product.CreatedAt,
		&result.Product.UpdatedAt,
		&result.Cohort.ID,
		&result.Cohort.ProductID,
		&result.Cohort.Slug,
		&result.Cohort.Title,
		&result.Cohort.Capacity,
		&result.Cohort.Status,
		&result.Cohort.StartsAt,
		&result.Cohort.EndsAt,
		&result.Cohort.EnrollmentOpensAt,
		&result.Cohort.EnrollmentClosesAt,
		&result.Cohort.CreatedAt,
		&result.Cohort.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CheckoutResult{}, false, nil
		}
		return CheckoutResult{}, false, fmt.Errorf("find checkout by idempotency key: %w", err)
	}

	sold, err := countSoldSeatsFromDB(ctx, db, result.Cohort.ID)
	if err == nil {
		result.SeatsRemaining = result.Cohort.Capacity - sold
		if result.SeatsRemaining < 0 {
			result.SeatsRemaining = 0
		}
	}

	return result, true, nil
}

func tryLoadIdempotentReplay(ctx context.Context, db *sql.DB, idempotencyKey string, originalErr error) (CheckoutResult, bool, error) {
	var pgErr *pgconn.PgError
	if !errors.As(originalErr, &pgErr) {
		return CheckoutResult{}, false, nil
	}
	if pgErr.Code != "23505" {
		return CheckoutResult{}, false, nil
	}

	result, found, err := findCheckoutByIdempotencyKey(ctx, db, idempotencyKey)
	if err != nil {
		return CheckoutResult{}, true, err
	}
	if !found {
		return CheckoutResult{}, true, originalErr
	}

	return result, true, nil
}

func lockCohortForCheckout(ctx context.Context, tx *sql.Tx, cohortID string) (Cohort, error) {
	const query = `
SELECT id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at
FROM cohorts
WHERE id = $1
FOR UPDATE`

	var cohort Cohort
	err := tx.QueryRowContext(ctx, query, cohortID).Scan(
		&cohort.ID,
		&cohort.ProductID,
		&cohort.Slug,
		&cohort.Title,
		&cohort.Capacity,
		&cohort.Status,
		&cohort.StartsAt,
		&cohort.EndsAt,
		&cohort.EnrollmentOpensAt,
		&cohort.EnrollmentClosesAt,
		&cohort.CreatedAt,
		&cohort.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Cohort{}, ErrNotFound
		}
		return Cohort{}, fmt.Errorf("lock cohort: %w", err)
	}

	return cohort, nil
}

func getProductForCheckout(ctx context.Context, tx *sql.Tx, productID string) (Product, error) {
	const query = `
SELECT id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at
FROM products
WHERE id = $1`

	var product Product
	err := tx.QueryRowContext(ctx, query, productID).Scan(
		&product.ID,
		&product.Slug,
		&product.Title,
		&product.Description,
		&product.ProductType,
		&product.PriceCents,
		&product.Currency,
		&product.Published,
		&product.Metadata,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		return Product{}, fmt.Errorf("get product for checkout: %w", err)
	}

	return product, nil
}

func countSoldSeats(ctx context.Context, tx *sql.Tx, cohortID string) (int, error) {
	const query = `
SELECT COALESCE(SUM(oi.quantity), 0)
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
WHERE oi.cohort_id = $1
  AND o.status = 'paid'`

	var sold int
	if err := tx.QueryRowContext(ctx, query, cohortID).Scan(&sold); err != nil {
		return 0, fmt.Errorf("count sold seats: %w", err)
	}

	return sold, nil
}

func countSoldSeatsFromDB(ctx context.Context, db queryer, cohortID string) (int, error) {
	const query = `
SELECT COALESCE(SUM(oi.quantity), 0)
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
WHERE oi.cohort_id = $1
  AND o.status = 'paid'`

	var sold int
	if err := db.QueryRowContext(ctx, query, cohortID).Scan(&sold); err != nil {
		return 0, fmt.Errorf("count sold seats from db: %w", err)
	}

	return sold, nil
}

// validateAndLockPromoCode keeps promo usage inside the same correctness boundary as checkout.
func validateAndLockPromoCode(ctx context.Context, tx *sql.Tx, code string, subtotal int) (PromoCode, int, error) {
	const query = `
SELECT id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at
FROM promo_codes
WHERE code = $1
FOR UPDATE`

	var promo PromoCode
	err := tx.QueryRowContext(ctx, query, code).Scan(
		&promo.ID,
		&promo.Code,
		&promo.DiscountType,
		&promo.DiscountValue,
		&promo.MaxRedemptions,
		&promo.StartsAt,
		&promo.ExpiresAt,
		&promo.CreatedAt,
		&promo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PromoCode{}, 0, ErrPromoCodeInvalid
		}
		return PromoCode{}, 0, fmt.Errorf("lock promo code: %w", err)
	}

	now := time.Now().UTC()
	if promo.StartsAt != nil && now.Before(*promo.StartsAt) {
		return PromoCode{}, 0, ErrPromoCodeInvalid
	}
	if promo.ExpiresAt != nil && !now.Before(*promo.ExpiresAt) {
		return PromoCode{}, 0, ErrPromoCodeInvalid
	}

	if promo.MaxRedemptions != nil {
		const countQuery = `SELECT COUNT(*) FROM promo_code_redemptions WHERE promo_code_id = $1`
		var redemptions int
		if err := tx.QueryRowContext(ctx, countQuery, promo.ID).Scan(&redemptions); err != nil {
			return PromoCode{}, 0, fmt.Errorf("count promo redemptions: %w", err)
		}
		if redemptions >= *promo.MaxRedemptions {
			return PromoCode{}, 0, ErrPromoCodeExhausted
		}
	}

	discount := 0
	switch promo.DiscountType {
	case "percent":
		discount = subtotal * promo.DiscountValue / 100
	case "fixed":
		discount = promo.DiscountValue
	}
	if discount > subtotal {
		discount = subtotal
	}

	return promo, discount, nil
}

func insertOrder(ctx context.Context, tx *sql.Tx, userID string, subtotal, discount, total int, currency string, idempotencyKey string, placedAt time.Time) (Order, error) {
	const query = `
INSERT INTO orders (user_id, status, currency, subtotal_cents, discount_cents, total_cents, idempotency_key, placed_at)
VALUES ($1, 'paid', $2, $3, $4, $5, $6, $7)
RETURNING id, user_id, status, currency, subtotal_cents, discount_cents, total_cents, idempotency_key, placed_at, created_at, updated_at`

	var order Order
	err := tx.QueryRowContext(ctx, query, userID, currency, subtotal, discount, total, idempotencyKey, placedAt).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.Currency,
		&order.SubtotalCents,
		&order.DiscountCents,
		&order.TotalCents,
		&order.IdempotencyKey,
		&order.PlacedAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	return order, nil
}

func insertOrderItem(ctx context.Context, tx *sql.Tx, orderID, productID, cohortID string, price int) error {
	const query = `
INSERT INTO order_items (order_id, product_id, cohort_id, item_type, quantity, unit_price_cents, line_total_cents)
VALUES ($1, $2, $3, 'live_cohort', 1, $4, $4)`

	if _, err := tx.ExecContext(ctx, query, orderID, productID, cohortID, price); err != nil {
		return fmt.Errorf("insert order item: %w", err)
	}

	return nil
}

func insertPayment(ctx context.Context, tx *sql.Tx, orderID, provider string, providerPaymentID *string, amount int, currency, idempotencyKey string) (Payment, error) {
	const query = `
INSERT INTO payments (order_id, provider, provider_payment_id, status, amount_cents, currency, idempotency_key)
VALUES ($1, $2, $3, 'captured', $4, $5, $6)
RETURNING id, order_id, provider, provider_payment_id, status, amount_cents, currency, idempotency_key, created_at, updated_at`

	var payment Payment
	err := tx.QueryRowContext(ctx, query, orderID, provider, providerPaymentID, amount, currency, idempotencyKey).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Provider,
		&payment.ProviderPaymentID,
		&payment.Status,
		&payment.AmountCents,
		&payment.Currency,
		&payment.IdempotencyKey,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return Payment{}, fmt.Errorf("insert payment: %w", err)
	}

	return payment, nil
}

func insertPromoRedemption(ctx context.Context, tx *sql.Tx, promoCodeID, orderID, userID string) error {
	const query = `
INSERT INTO promo_code_redemptions (promo_code_id, order_id, user_id)
VALUES ($1, $2, $3)`

	if _, err := tx.ExecContext(ctx, query, promoCodeID, orderID, userID); err != nil {
		return fmt.Errorf("insert promo redemption: %w", err)
	}

	return nil
}

func insertEntitlement(ctx context.Context, tx *sql.Tx, userID, productID, orderID, cohortID string, grantedAt time.Time) (Entitlement, error) {
	const query = `
INSERT INTO entitlements (user_id, product_id, order_id, cohort_id, status, granted_at)
VALUES ($1, $2, $3, $4, 'active', $5)
RETURNING id, user_id, product_id, order_id, cohort_id, status, granted_at, revoked_at, created_at, updated_at`

	var entitlement Entitlement
	err := tx.QueryRowContext(ctx, query, userID, productID, orderID, cohortID, grantedAt).Scan(
		&entitlement.ID,
		&entitlement.UserID,
		&entitlement.ProductID,
		&entitlement.OrderID,
		&entitlement.CohortID,
		&entitlement.Status,
		&entitlement.GrantedAt,
		&entitlement.RevokedAt,
		&entitlement.CreatedAt,
		&entitlement.UpdatedAt,
	)
	if err != nil {
		return Entitlement{}, fmt.Errorf("insert entitlement: %w", err)
	}

	return entitlement, nil
}

func insertAuditLog(ctx context.Context, tx *sql.Tx, actorUserID, entityType, entityID, action string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal audit log payload: %w", err)
	}

	const query = `
INSERT INTO audit_log (actor_user_id, entity_type, entity_id, action, payload)
VALUES ($1, $2, $3, $4, $5)`

	if _, err := tx.ExecContext(ctx, query, actorUserID, entityType, entityID, action, body); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

func insertOutboxEvent(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	const query = `
INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
VALUES ($1, $2, $3, $4)`

	if _, err := tx.ExecContext(ctx, query, aggregateType, aggregateID, eventType, body); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}
