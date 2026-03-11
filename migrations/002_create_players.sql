-- Migración 002: Tabla de jugadores
-- Rating: 1-10, solo visible para admins
-- can_play_positions: array de posiciones secundarias
-- active: soft delete — no se borra, se desactiva

CREATE TABLE IF NOT EXISTS players (
    id                  UUID         PRIMARY KEY,
    name                VARCHAR(100) NOT NULL,
    primary_position    VARCHAR(20)  NOT NULL CHECK (primary_position IN ('delantero', 'defensa', 'creador')),
    can_play_positions  TEXT[]       NOT NULL DEFAULT '{}',
    notes               TEXT         NOT NULL DEFAULT '',
    rating              SMALLINT     NOT NULL CHECK (rating BETWEEN 1 AND 10),
    active              BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_players_active ON players(active);
CREATE INDEX IF NOT EXISTS idx_players_name   ON players(name);
