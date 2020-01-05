CREATE TABLE job
(
    id         SERIAL PRIMARY KEY,
    source_id  INTEGER REFERENCES source (id),
    init_time  TIMESTAMP   NOT NULL DEFAULT now(),
    start_time TIMESTAMP,
    end_time   TIMESTAMP,
    state      VARCHAR(20) NOT NULL,
    trigger    VARCHAR(20) NOT NULL
);
