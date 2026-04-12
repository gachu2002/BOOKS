CREATE TABLE user_library_projection (
    order_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    product_id UUID NOT NULL REFERENCES products(id),
    cohort_id UUID REFERENCES cohorts(id),
    product_slug TEXT NOT NULL,
    product_title TEXT NOT NULL,
    cohort_slug TEXT,
    cohort_title TEXT,
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    currency TEXT NOT NULL,
    source_event_id UUID NOT NULL UNIQUE,
    source_event_type TEXT NOT NULL,
    projected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_library_projection_user_id ON user_library_projection (user_id, projected_at DESC);
