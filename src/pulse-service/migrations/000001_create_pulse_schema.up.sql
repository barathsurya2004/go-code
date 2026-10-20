CREATE TABLE IF NOT EXISTS pulse_users (
    user_uuid VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255),
    name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pulse_settings (
    user_uuid VARCHAR(64) PRIMARY KEY,
    unit VARCHAR(8) NOT NULL DEFAULT 'kg',
    bar_weight DOUBLE PRECISION NOT NULL DEFAULT 20,
    default_rest INT NOT NULL DEFAULT 90,
    haptics BOOLEAN NOT NULL DEFAULT true,
    sound BOOLEAN NOT NULL DEFAULT true,
    auto_start_rest BOOLEAN NOT NULL DEFAULT true,
    weekly_goal INT NOT NULL DEFAULT 4,
    bodyweight DOUBLE PRECISION NOT NULL DEFAULT 75,
    name VARCHAR(255) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pulse_routines (
    id VARCHAR(64) NOT NULL,
    user_uuid VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    focus VARCHAR(255) NOT NULL DEFAULT '',
    color VARCHAR(32) NOT NULL DEFAULT 'mint',
    est_minutes INT NOT NULL DEFAULT 45,
    last_performed TIMESTAMPTZ,
    warmup_note TEXT DEFAULT '',
    nutrition_note TEXT DEFAULT '',
    blocks JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_uuid, id)
);

CREATE TABLE IF NOT EXISTS pulse_plans (
    user_uuid VARCHAR(64) NOT NULL,
    weekday INT NOT NULL,
    routine_id VARCHAR(64),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_uuid, weekday)
);

CREATE TABLE IF NOT EXISTS pulse_workout_sessions (
    id VARCHAR(64) NOT NULL,
    user_uuid VARCHAR(64) NOT NULL,
    routine_id VARCHAR(64),
    name VARCHAR(255) NOT NULL,
    color VARCHAR(32) NOT NULL DEFAULT 'mint',
    started_at BIGINT NOT NULL,
    ended_at BIGINT,
    rpe DOUBLE PRECISION,
    mood INT,
    notes TEXT DEFAULT '',
    blocks JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_uuid, id)
);

CREATE TABLE IF NOT EXISTS pulse_skipped_days (
    user_uuid VARCHAR(64) NOT NULL,
    date_key VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_uuid, date_key)
);

CREATE TABLE IF NOT EXISTS pulse_sync_logs (
    id BIGSERIAL PRIMARY KEY,
    user_uuid VARCHAR(64) NOT NULL,
    client_time BIGINT NOT NULL,
    routines_count INT NOT NULL DEFAULT 0,
    sessions_count INT NOT NULL DEFAULT 0,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pulse_workout_sessions_user_started ON pulse_workout_sessions(user_uuid, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_pulse_routines_user ON pulse_routines(user_uuid);
