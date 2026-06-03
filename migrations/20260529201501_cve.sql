-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS cve (
    id TEXT PRIMARY KEY,
    descr TEXT,
    severity TEXT,
    refs TEXT
);
CREATE INDEX IF NOT EXISTS idx_cve_severity ON cve(severity);
-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS cve;
