-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS cpe (
    cpe_name TEXT PRIMARY KEY,
    vendor TEXT NOT NULL,
    product TEXT,
    ver TEXT
);
CREATE INDEX IF NOT EXISTS idx_cpe_product_version ON cpe(product, ver);
CREATE INDEX IF NOT EXISTS idx_cpe_vendor_product_version ON cpe(vendor, product, ver);
-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS cpe;