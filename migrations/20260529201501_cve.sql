-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS cve (
    id TEXT PRIMARY KEY,
    description TEXT,
    severity TEXT,
    references TEXT
);
-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS cve;
