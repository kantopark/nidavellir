CREATE TABLE app_user
(
    id       SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE CHECK ( length(password) >= 1 ) NOT NULL,
    password VARCHAR(255) CHECK ( length(password) >= 1 )        NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE
);
