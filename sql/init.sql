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
    user_status REFERENCES user_status_types(user_status),
    game_id REFERENCES games(game_id) DEFAULT NULL ON DELETE SET NULL,
    PRIMARY KEY(username)
);

CREATE TABLE IF NOT EXISTS user_status_types (
    user_status varchar(16)
);

INSERT INTO user_status_types VALUES ('idle', 'hosting', 'playing');

CREATE TABLE IF NOT EXISTS games (
    game_id uuid,
    player_one REFERENCES users(username) ON DELETE CASCADE,
    player_two REFERENCES users(username) ON DELETE CASCADE,
    PRIMARY KEY(game_id)
);

-- todo: game location table

