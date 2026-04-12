CREATE TABLE analytics_daily_revenue (
    report_date DATE NOT NULL,
    currency TEXT NOT NULL,
    orders_count INTEGER NOT NULL CHECK (orders_count >= 0),
    gross_revenue_cents BIGINT NOT NULL CHECK (gross_revenue_cents >= 0),
    discount_cents BIGINT NOT NULL CHECK (discount_cents >= 0),
    net_revenue_cents BIGINT NOT NULL CHECK (net_revenue_cents >= 0),
    rebuilt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (report_date, currency)
);

CREATE TABLE analytics_cohort_fill (
    cohort_id UUID PRIMARY KEY REFERENCES cohorts(id),
    product_id UUID NOT NULL REFERENCES products(id),
    cohort_slug TEXT NOT NULL,
    cohort_title TEXT NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    sold_seats INTEGER NOT NULL CHECK (sold_seats >= 0),
    fill_rate_percent NUMERIC(5,2) NOT NULL CHECK (fill_rate_percent >= 0),
    revenue_cents BIGINT NOT NULL CHECK (revenue_cents >= 0),
    starts_at TIMESTAMPTZ NOT NULL,
    rebuilt_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_analytics_daily_revenue_date ON analytics_daily_revenue (report_date DESC);
CREATE INDEX idx_analytics_cohort_fill_starts_at ON analytics_cohort_fill (starts_at DESC);
