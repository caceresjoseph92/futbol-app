-- Migración 003: Tablas de partidos y jugadores por partido
--
-- matches: cabecera del partido (fecha, estado, resultado, arqueros)
-- match_players: los 12 jugadores convocados con snapshot de rating
--   - team 1 o 2: equipo asignado
--   - rating_snapshot: calificación del jugador EN ESE MOMENTO
--     (permite historial fiel aunque el rating cambie después)

CREATE TABLE IF NOT EXISTS matches (
    id              UUID        PRIMARY KEY,
    played_at       DATE        NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'published', 'finished')),
    team1_score     SMALLINT,
    team2_score     SMALLINT,
    goalkeeper_info TEXT        NOT NULL DEFAULT '',
    created_by      UUID        NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS match_players (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id         UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    player_id        UUID        NOT NULL REFERENCES players(id),
    player_name      VARCHAR(100) NOT NULL,   -- snapshot del nombre
    primary_position VARCHAR(20) NOT NULL,    -- snapshot de la posición
    notes            TEXT        NOT NULL DEFAULT '',
    team             SMALLINT    NOT NULL CHECK (team IN (0, 1, 2)),
                                              -- 0 = sin asignar, 1 = equipo 1, 2 = equipo 2
    rating_snapshot  SMALLINT    NOT NULL CHECK (rating_snapshot BETWEEN 1 AND 10),

    UNIQUE (match_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_played_at ON matches(played_at DESC);
CREATE INDEX IF NOT EXISTS idx_matches_status    ON matches(status);
CREATE INDEX IF NOT EXISTS idx_match_players_match ON match_players(match_id);
