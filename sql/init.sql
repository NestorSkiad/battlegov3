set timezone = 'Europe/Paris';

CREATE TABLE IF NOT EXISTS users (
    username varchar(16) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS tokens (
    username REFERENCES users(username) ON DELETE CASCADE,
    token uuid NOT NULL,
    lastaccess timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY(username, token),
    UNIQUE(username),
    UNIQUE(token)
);

CREATE TABLE IF NOT EXISTS user_status (
    user_token REFERENCES users(token) PRIMARY KEY ON DELETE CASCADE,
    user_status REFERENCES user_status_types(status_type),
    game_id REFERENCES games(game_id) DEFAULT NULL ON DELETE SET NULL,
    PRIMARY KEY(username)
);

CREATE TABLE IF NOT EXISTS user_status_types (
    status_type varchar(16)
);

INSERT INTO user_status_types VALUES ('idle', 'hosting', 'playing');

CREATE TABLE IF NOT EXISTS games (
    game_id uuid,
    player_one REFERENCES users(token) NOT NULL ON DELETE CASCADE,
    player_two REFERENCES users(token) NOT NULL ON DELETE CASCADE,
    host inet
    PRIMARY KEY(game_id)
);

-- todo: game location table

