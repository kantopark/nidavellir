CREATE TABLE IF NOT EXISTS source
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) CHECK ( length(name) >= 4 ) UNIQUE,
    unique_name VARCHAR(100) UNIQUE,
    repo_url    VARCHAR(2000),
    state       VARCHAR(20)  NOT NULL,
    next_time   TIMESTAMP    NOT NULL,
    cron_expr   VARCHAR(100) NOT NULL
);

CREATE TABLE secret
(
    id        SERIAL PRIMARY KEY,
    source_id INTEGER REFERENCES source (id) ON DELETE CASCADE,
    key       VARCHAR(255) CHECK ( length(key) >= 1 ) NOT NULL,
    value     TEXT CHECK ( length(value) >= 1 )       NOT NULL,
    UNIQUE (source_id, key)
);


CREATE INDEX source_state_next_time ON source
    (state, next_time);
