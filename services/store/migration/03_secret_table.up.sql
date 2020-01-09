CREATE TABLE secret
(
    id        SERIAL PRIMARY KEY,
    source_id INTEGER REFERENCES source (id),
    key       VARCHAR(255) CHECK ( length(key) >= 1 ) NOT NULL,
    value     TEXT CHECK ( length(value) >= 1 )       NOT NULL,
    UNIQUE (source_id, key)
);
