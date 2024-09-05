set timezone = 'Europe/Paris';

CREATE TABLE IF NOT EXISTS users (
    username varchar(16) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS tokens (
    username REFERENCES users(username) ON DELETE CASCADE,
    token uuid NOT NULL,
    lastaccess timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY(username, token),
    UNIQUE(username)
);

CREATE TABLE IF NOT EXISTS user_status (
    username REFERENCES users(username) PRIMARY KEY ON DELETE CASCADE,
    user_status 
    PRIMARY KEY(username)
);

CREATE TABLE IF NOT EXISTS user_status_types (
    user_status varchar(16)
);

INSERT INTO user_status_types VALUES ('idle', 'hosting', 'playing');

