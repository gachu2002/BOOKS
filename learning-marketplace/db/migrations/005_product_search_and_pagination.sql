CREATE INDEX idx_products_published_created_id ON products (published, created_at DESC, id DESC);

CREATE INDEX idx_products_search ON products
USING GIN (to_tsvector('simple', COALESCE(title, '') || ' ' || COALESCE(description, '')));
