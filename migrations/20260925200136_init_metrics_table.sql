-- +goose Up
CREATE TABLE IF NOT EXISTS metrics (
    id    VARCHAR(250) NOT NULL,
    type  VARCHAR(50)  NOT NULL CHECK (type IN ('counter', 'gauge')),
    delta BIGINT,
    value DOUBLE PRECISION,
    hash  VARCHAR(500) DEFAULT '',
    PRIMARY KEY (id, type)
);

-- +goose Down
DROP TABLE IF EXISTS metrics;
