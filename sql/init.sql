set timezone = 'Europe/Paris';

CREATE TABLE IF NOT EXISTS tokens (
    token uuid,
    username varchar(16) NOT NULL,
    lastaccess timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY(token),
    UNIQUE(username),
    UNIQUE(token)
);

CREATE TABLE IF NOT EXISTS user_status (
    user_token REFERENCES tokens(token) ON DELETE CASCADE,
    user_status REFERENCES user_status_types(status_type),
    game_id REFERENCES games(game_id) DEFAULT NULL ON DELETE SET NULL,
    PRIMARY KEY(user_token)
);

CREATE TABLE IF NOT EXISTS user_status_types (
    status_type varchar(16)
);

INSERT INTO user_status_types VALUES ('idle', 'hosting', 'playing');

CREATE TABLE IF NOT EXISTS games (
    game_id uuid,
    player_one REFERENCES tokens(token) NOT NULL ON DELETE CASCADE,
    player_two REFERENCES tokens(token) NOT NULL ON DELETE CASCADE,
    host inet
    PRIMARY KEY(game_id)
);

-- todo: game location table

