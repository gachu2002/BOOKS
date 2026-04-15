ALTER TABLE analytics_daily_revenue
    ADD COLUMN IF NOT EXISTS rebuilt_by TEXT NOT NULL DEFAULT 'batch-reports',
    ADD COLUMN IF NOT EXISTS rebuild_fencing_token BIGINT NOT NULL DEFAULT 0;

ALTER TABLE analytics_cohort_fill
    ADD COLUMN IF NOT EXISTS rebuilt_by TEXT NOT NULL DEFAULT 'batch-reports',
    ADD COLUMN IF NOT EXISTS rebuild_fencing_token BIGINT NOT NULL DEFAULT 0;
