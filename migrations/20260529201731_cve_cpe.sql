-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS cpe_cve (
    cpe_name TEXT REFERENCES cpe(cpe_name) ON DELETE CASCADE,
    cve_id TEXT REFERENCES cve(id) ON DELETE CASCADE,
    PRIMARY KEY (cpe_name, cve_id)
);
CREATE INDEX IF NOT EXISTS idx_cpe_products_product ON cpe_cve(cpe_name);
CREATE INDEX IF NOT EXISTS idx_cpe_products_product ON cpe_cve(cpe_id);
-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS cpe_cve;