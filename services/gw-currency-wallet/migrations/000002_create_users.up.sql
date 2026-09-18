CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL
);
CREATE UNIQUE INDEX users_email_lower_idx
    ON users (lower(email));
