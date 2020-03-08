CREATE TABLE account
(
    id       SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE CHECK ( length(username) >= 1 ) NOT NULL,
    password VARCHAR(255),
    is_admin BOOLEAN DEFAULT FALSE
);
