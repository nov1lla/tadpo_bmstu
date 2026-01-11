BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 0),
    win_streak INTEGER NOT NULL CHECK (win_streak >= 0),
    last_game_id TEXT
);

CREATE TABLE IF NOT EXISTS games (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    status TEXT NOT NULL,
    player_color TEXT NOT NULL,
    is_player_first BOOLEAN NOT NULL,
    board_size INTEGER NOT NULL DEFAULT 8 CHECK (board_size > 0)
);

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_last_game_fk;

ALTER TABLE users
    ADD CONSTRAINT users_last_game_fk
    FOREIGN KEY (last_game_id) REFERENCES games(id)
    ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS game_board_states (
    game_id TEXT PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    board_size INTEGER NOT NULL CHECK (board_size >= 0),
    pieces_json JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS moves (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    piece_id TEXT NOT NULL,
    number INTEGER NOT NULL CHECK (number > 0),
    start_row INTEGER NOT NULL,
    start_col INTEGER NOT NULL,
    end_row INTEGER NOT NULL,
    end_col INTEGER NOT NULL,
    path_json JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL,
    UNIQUE (game_id, number)
);

ALTER TABLE moves
    ADD COLUMN IF NOT EXISTS path_json JSONB;

ALTER TABLE moves
    ALTER COLUMN path_json SET DEFAULT '[]'::jsonb;

UPDATE moves SET path_json = '[]'::jsonb WHERE path_json IS NULL;

ALTER TABLE moves
    ALTER COLUMN path_json SET NOT NULL;

COMMIT;
