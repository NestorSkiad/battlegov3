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
    host_addr REFERENCES hosts(host_addr) DEFAULT NULL ON DELETE SET NULL,
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
    host_addr REFERENCES hosts(host_addr),
    PRIMARY KEY(game_id)
);

CREATE TABLE IF NOT EXISTS game_history (
    player REFERENCES tokens(token) ON DELETE CASCADE,
    game_id uuid,
    won boolean,
    PRIMARY KEY(player)
);

-- if I do busy logic, add status field here
-- busy logic =: thread checks server status every few minutes, sets status based on metrics
-- when user tries to host, if server is busy, redirect to random non-busy server
CREATE TABLE IF NOT EXISTS hosts (
    host_addr varchar(45) PRIMARY KEY
);

