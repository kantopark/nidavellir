CREATE TABLE schedule
(
    id        SERIAL PRIMARY KEY,
    source_id INTEGER REFERENCES source (id),
    interval  INTEGER     NOT NULL,
    state     VARCHAR(20) NOT NULL,
    next_time TIMESTAMP   NOT NULL
);

CREATE INDEX schedule_state_next_time ON schedule
    (state, next_time);
