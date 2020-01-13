CREATE TABLE IF NOT EXISTS source
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) CHECK ( length(name) >= 4 ) UNIQUE,
    unique_name VARCHAR(100) UNIQUE,
    repo_url    VARCHAR(2000),
    interval    INTEGER     NOT NULL,
    state       VARCHAR(20) NOT NULL,
    next_time   TIMESTAMP   NOT NULL
);

CREATE INDEX source_state_next_time ON source
    (state, next_time);
