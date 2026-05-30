-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS cpe (
    cpe_name TEXT PRIMARY KEY,
    product TEXT,
    version TEXT
);
CREATE INDEX IF NOT EXISTS idx_cpe_products_product ON cpe(product,version);
-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS cpe;