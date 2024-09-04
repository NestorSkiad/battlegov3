set timezone = 'Europe/Paris';

CREATE TABLE IF NOT EXISTS users (
    username varchar(16) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS token (
    username REFERENCES users(username) PRIMARY KEY,
    token uuid NOT NULL,
    lastaccess timestamptz DEFAULT current_timestamp
);

