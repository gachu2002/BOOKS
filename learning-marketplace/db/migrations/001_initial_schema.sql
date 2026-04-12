CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'student' CHECK (role IN ('student', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    product_type TEXT NOT NULL CHECK (product_type IN ('digital_download', 'live_cohort')),
    price_cents INTEGER NOT NULL CHECK (price_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'USD',
    published BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cohorts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    status TEXT NOT NULL CHECK (status IN ('draft', 'open', 'full', 'closed', 'cancelled')),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    enrollment_opens_at TIMESTAMPTZ,
    enrollment_closes_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE promo_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    discount_type TEXT NOT NULL CHECK (discount_type IN ('percent', 'fixed')),
    discount_value INTEGER NOT NULL CHECK (discount_value > 0),
    max_redemptions INTEGER,
    starts_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (max_redemptions IS NULL OR max_redemptions > 0),
    CHECK (expires_at IS NULL OR starts_at IS NULL OR expires_at > starts_at)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL CHECK (status IN ('pending', 'awaiting_payment', 'paid', 'cancelled', 'expired', 'refunded')),
    currency TEXT NOT NULL DEFAULT 'USD',
    subtotal_cents INTEGER NOT NULL DEFAULT 0 CHECK (subtotal_cents >= 0),
    discount_cents INTEGER NOT NULL DEFAULT 0 CHECK (discount_cents >= 0),
    total_cents INTEGER NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    idempotency_key TEXT UNIQUE,
    placed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    cohort_id UUID REFERENCES cohorts(id),
    item_type TEXT NOT NULL CHECK (item_type IN ('digital_download', 'live_cohort')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents >= 0),
    line_total_cents INTEGER NOT NULL CHECK (line_total_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    provider TEXT NOT NULL,
    provider_payment_id TEXT UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'authorized', 'captured', 'failed', 'refunded')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'USD',
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE promo_code_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promo_code_id UUID NOT NULL REFERENCES promo_codes(id),
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (promo_code_id, user_id)
);

CREATE TABLE entitlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    product_id UUID NOT NULL REFERENCES products(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    cohort_id UUID REFERENCES cohorts(id),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'revoked')),
    granted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (user_id, product_id, cohort_id)
);

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id UUID REFERENCES users(id),
    entity_type TEXT NOT NULL,
    entity_id UUID,
    action TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_products_published ON products (published, created_at DESC);
CREATE INDEX idx_cohorts_product_id ON cohorts (product_id);
CREATE INDEX idx_cohorts_status_starts_at ON cohorts (status, starts_at);
CREATE INDEX idx_orders_user_id_created_at ON orders (user_id, created_at DESC);
CREATE INDEX idx_orders_status_created_at ON orders (status, created_at DESC);
CREATE INDEX idx_order_items_order_id ON order_items (order_id);
CREATE INDEX idx_order_items_product_id ON order_items (product_id);
CREATE INDEX idx_order_items_cohort_id ON order_items (cohort_id) WHERE cohort_id IS NOT NULL;
CREATE INDEX idx_payments_order_id ON payments (order_id);
CREATE INDEX idx_promo_code_redemptions_code_user ON promo_code_redemptions (promo_code_id, user_id);
CREATE INDEX idx_entitlements_user_id_status ON entitlements (user_id, status);
CREATE INDEX idx_outbox_events_unpublished ON outbox_events (created_at) WHERE published_at IS NULL;
