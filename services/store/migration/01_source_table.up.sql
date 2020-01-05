CREATE TABLE IF NOT EXISTS source
(
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) UNIQUE,
    repo_url   VARCHAR,
    commit_tag VARCHAR(40)
);
