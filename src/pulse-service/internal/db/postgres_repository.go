package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type PostgresPulseRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewPostgresPulseRepository(db *sql.DB, log *zap.Logger) *PostgresPulseRepository {
	return &PostgresPulseRepository{
		db:  db,
		log: log,
	}
}

func NewPulseRepository(log *zap.Logger) PulseRepository {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Info("No DATABASE_URL configured, using in-memory pulse repository")
		return NewMemoryPulseRepository(log)
	}

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Warn("Failed to open PostgreSQL connection, using in-memory fallback", zap.Error(err))
		return NewMemoryPulseRepository(log)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := dbConn.PingContext(ctx); err != nil {
		log.Warn("PostgreSQL ping failed, using in-memory fallback", zap.Error(err))
		_ = dbConn.Close()
		return NewMemoryPulseRepository(log)
	}

	repo := NewPostgresPulseRepository(dbConn, log)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		log.Warn("Failed to verify/migrate schema on pulse_app, using in-memory fallback", zap.Error(err))
		_ = dbConn.Close()
		return NewMemoryPulseRepository(log)
	}

	log.Info("Connected dedicated PostgreSQL database successfully for Pulse service")
	return repo
}

func (r *PostgresPulseRepository) EnsureSchema(ctx context.Context) error {
	schemaQuery := `
CREATE TABLE IF NOT EXISTS pulse_users (
    id VARCHAR(64) PRIMARY KEY,
    user_uuid VARCHAR(64),
    email VARCHAR(255),
    name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE pulse_users ADD COLUMN IF NOT EXISTS user_uuid VARCHAR(64);
ALTER TABLE pulse_users ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE pulse_users ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE pulse_users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();
CREATE UNIQUE INDEX IF NOT EXISTS idx_pulse_users_uuid ON pulse_users(user_uuid);

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
    id VARCHAR(64) PRIMARY KEY,
    user_uuid VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    focus VARCHAR(255) NOT NULL DEFAULT '',
    color VARCHAR(32) NOT NULL DEFAULT 'mint',
    est_minutes INT NOT NULL DEFAULT 45,
    last_performed VARCHAR(64),
    warmup_note TEXT DEFAULT '',
    nutrition_note TEXT DEFAULT '',
    blocks JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE pulse_routines ADD COLUMN IF NOT EXISTS blocks JSONB NOT NULL DEFAULT '[]';
ALTER TABLE pulse_routines ADD COLUMN IF NOT EXISTS last_performed VARCHAR(64);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pulse_routines_user_id ON pulse_routines(user_uuid, id);

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
    id VARCHAR(64) PRIMARY KEY,
    user_uuid VARCHAR(64) NOT NULL,
    client_mutation_id VARCHAR(64),
    entity_type VARCHAR(64),
    operation VARCHAR(32),
    client_time BIGINT,
    routines_count INT NOT NULL DEFAULT 0,
    sessions_count INT NOT NULL DEFAULT 0,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE pulse_sync_logs ADD COLUMN IF NOT EXISTS client_time BIGINT;
ALTER TABLE pulse_sync_logs ADD COLUMN IF NOT EXISTS routines_count INT DEFAULT 0;
ALTER TABLE pulse_sync_logs ADD COLUMN IF NOT EXISTS sessions_count INT DEFAULT 0;
ALTER TABLE pulse_sync_logs ALTER COLUMN client_mutation_id DROP NOT NULL;
ALTER TABLE pulse_sync_logs ALTER COLUMN entity_type DROP NOT NULL;
ALTER TABLE pulse_sync_logs ALTER COLUMN operation DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pulse_workout_sessions_user_started ON pulse_workout_sessions(user_uuid, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_pulse_routines_user ON pulse_routines(user_uuid);
`
	_, err := r.db.ExecContext(ctx, schemaQuery)
	return err
}

func (r *PostgresPulseRepository) SaveSyncPayload(ctx context.Context, payload *core.SyncPayload) (*core.SyncResponse, error) {
	if payload == nil {
		return nil, ErrEmptyPayload
	}
	if payload.UserUUID == "" {
		return nil, ErrInvalidUserUUID
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	now := time.Now()

	// 1. Upsert User
	_, err = tx.ExecContext(ctx, `
		INSERT INTO pulse_users (id, user_uuid, updated_at)
		VALUES ($1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET updated_at = $2
	`, payload.UserUUID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert pulse_user: %w", err)
	}

	// 2. Upsert Settings if provided
	if payload.Settings != nil {
		s := payload.Settings
		_, err = tx.ExecContext(ctx, `
			INSERT INTO pulse_settings (
				user_uuid, unit, bar_weight, default_rest, haptics, sound, auto_start_rest, weekly_goal, bodyweight, name, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (user_uuid) DO UPDATE SET
				unit = EXCLUDED.unit,
				bar_weight = EXCLUDED.bar_weight,
				default_rest = EXCLUDED.default_rest,
				haptics = EXCLUDED.haptics,
				sound = EXCLUDED.sound,
				auto_start_rest = EXCLUDED.auto_start_rest,
				weekly_goal = EXCLUDED.weekly_goal,
				bodyweight = EXCLUDED.bodyweight,
				name = EXCLUDED.name,
				updated_at = EXCLUDED.updated_at
		`, payload.UserUUID, s.Unit, s.BarWeight, s.DefaultRest, s.Haptics, s.Sound, s.AutoStartRest, s.WeeklyGoal, s.Bodyweight, s.Name, now)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert pulse_settings: %w", err)
		}
	}

	// 3. Upsert Routines
	for _, rt := range payload.Routines {
		blocksJSON, err := json.Marshal(rt.Blocks)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal routine blocks: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO pulse_routines (
				id, user_uuid, name, focus, color, est_minutes, last_performed, warmup_note, nutrition_note, blocks, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				focus = EXCLUDED.focus,
				color = EXCLUDED.color,
				est_minutes = EXCLUDED.est_minutes,
				last_performed = EXCLUDED.last_performed,
				warmup_note = EXCLUDED.warmup_note,
				nutrition_note = EXCLUDED.nutrition_note,
				blocks = EXCLUDED.blocks,
				updated_at = EXCLUDED.updated_at
		`, rt.ID, payload.UserUUID, rt.Name, rt.Focus, rt.Color, rt.EstMinutes, rt.LastPerformed, rt.WarmupNote, rt.NutritionNote, string(blocksJSON), now)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert routine %s: %w", rt.ID, err)
		}
	}

	// 4. Upsert Plans
	for _, pl := range payload.Plan {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO pulse_plans (user_uuid, weekday, routine_id, updated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_uuid, weekday) DO UPDATE SET
				routine_id = EXCLUDED.routine_id,
				updated_at = EXCLUDED.updated_at
		`, payload.UserUUID, pl.Weekday, pl.RoutineID, now)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert plan for weekday %d: %w", pl.Weekday, err)
		}
	}

	// 5. Upsert Sessions
	for _, ss := range payload.Sessions {
		blocksJSON, err := json.Marshal(ss.Blocks)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session blocks: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO pulse_workout_sessions (
				id, user_uuid, routine_id, name, color, started_at, ended_at, rpe, mood, notes, blocks, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (user_uuid, id) DO UPDATE SET
				routine_id = EXCLUDED.routine_id,
				name = EXCLUDED.name,
				color = EXCLUDED.color,
				started_at = EXCLUDED.started_at,
				ended_at = EXCLUDED.ended_at,
				rpe = EXCLUDED.rpe,
				mood = EXCLUDED.mood,
				notes = EXCLUDED.notes,
				blocks = EXCLUDED.blocks,
				updated_at = EXCLUDED.updated_at
		`, ss.ID, payload.UserUUID, ss.RoutineID, ss.Name, ss.Color, ss.StartedAt, ss.EndedAt, ss.RPE, ss.Mood, ss.Notes, string(blocksJSON), now)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert session %s: %w", ss.ID, err)
		}
	}

	// 6. Upsert Skipped Days
	for _, dateKey := range payload.SkippedDates {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO pulse_skipped_days (user_uuid, date_key, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_uuid, date_key) DO NOTHING
		`, payload.UserUUID, dateKey, now)
		if err != nil {
			return nil, fmt.Errorf("failed to record skipped day %s: %w", dateKey, err)
		}
	}

	// 7. Insert Audit Sync Log
	syncLogID := uuid.New().String()
	clientMutationID := fmt.Sprintf("%d", payload.ClientTime)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO pulse_sync_logs (
			id, user_uuid, client_mutation_id, entity_type, operation, client_time, routines_count, sessions_count, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, syncLogID, payload.UserUUID, clientMutationID, "sync_batch", "push", payload.ClientTime, len(payload.Routines), len(payload.Sessions), now)
	if err != nil {
		return nil, fmt.Errorf("failed to record sync log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &core.SyncResponse{
		Status:         "ok",
		Message:        "Push sync completed successfully",
		SyncedAt:       now.UnixMilli(),
		SessionsSynced: len(payload.Sessions),
		RoutinesSynced: len(payload.Routines),
	}, nil
}

func (r *PostgresPulseRepository) GetUserStats(ctx context.Context, userUUID string) (*core.PulseUserStats, error) {
	if userUUID == "" {
		return nil, ErrInvalidUserUUID
	}

	var stats core.PulseUserStats
	stats.UserUUID = userUUID

	row := r.db.QueryRowContext(ctx, `
		SELECT updated_at FROM pulse_users WHERE user_uuid = $1 OR id = $1
	`, userUUID)
	_ = row.Scan(&stats.LastSyncedAt)

	row = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pulse_workout_sessions WHERE user_uuid = $1
	`, userUUID)
	_ = row.Scan(&stats.TotalSessions)

	row = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pulse_routines WHERE user_uuid = $1
	`, userUUID)
	_ = row.Scan(&stats.TotalRoutines)

	return &stats, nil
}

func (r *PostgresPulseRepository) DeleteUserData(ctx context.Context, userUUID string) error {
	if userUUID == "" {
		return ErrInvalidUserUUID
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	tables := []string{
		"pulse_workout_sessions",
		"pulse_routines",
		"pulse_plans",
		"pulse_settings",
		"pulse_skipped_days",
		"pulse_sync_logs",
	}
	for _, tbl := range tables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE user_uuid = $1", tbl), userUUID); err != nil {
			return fmt.Errorf("failed to delete from %s: %w", tbl, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM pulse_users WHERE user_uuid = $1 OR id = $1", userUUID); err != nil {
		return fmt.Errorf("failed to delete from pulse_users: %w", err)
	}

	return tx.Commit()
}

func (r *PostgresPulseRepository) CreateUser(ctx context.Context, req *core.CreateUserRequest) (*core.PulseUser, error) {
	if req == nil {
		return nil, errors.New("empty create user request")
	}
	if req.Email == "" {
		return nil, errors.New("email is required")
	}

	uuidStr := req.UserUUID
	if uuidStr == "" {
		uuidStr = uuid.New().String()
	}

	var passHash string
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		passHash = string(hashed)
	}

	now := time.Now()
	// Check if user with this email already exists
	existing, err := r.GetUserByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrUserAlreadyExists
	}

	query := `
		INSERT INTO pulse_users (id, user_uuid, email, name, password_hash, created_at, updated_at)
		VALUES ($1, $1, $2, $3, $4, $5, $5)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			password_hash = CASE WHEN EXCLUDED.password_hash != '' THEN EXCLUDED.password_hash ELSE pulse_users.password_hash END,
			updated_at = EXCLUDED.updated_at
		RETURNING id, user_uuid, email, name, created_at, updated_at
	`
	user := core.PulseUser{
		ID:           uuidStr,
		UserUUID:     uuidStr,
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: passHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = r.db.QueryRowContext(ctx, query, uuidStr, req.Email, req.Name, passHash, now).
		Scan(&user.ID, &user.UserUUID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create pulse user: %w", err)
	}

	return &user, nil
}

func (r *PostgresPulseRepository) GetUserByEmail(ctx context.Context, email string) (*core.PulseUser, error) {
	if email == "" {
		return nil, ErrUserNotFound
	}
	query := `
		SELECT id, COALESCE(user_uuid, id), COALESCE(email, ''), COALESCE(name, ''), COALESCE(password_hash, ''), created_at, updated_at
		FROM pulse_users
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`
	var u core.PulseUser
	err := r.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.UserUUID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}
	return &u, nil
}

func (r *PostgresPulseRepository) GetUserByUUID(ctx context.Context, userUUID string) (*core.PulseUser, error) {
	if userUUID == "" {
		return nil, ErrInvalidUserUUID
	}
	query := `
		SELECT id, COALESCE(user_uuid, id), COALESCE(email, ''), COALESCE(name, ''), COALESCE(password_hash, ''), created_at, updated_at
		FROM pulse_users
		WHERE user_uuid = $1 OR id = $1
		LIMIT 1
	`
	var u core.PulseUser
	err := r.db.QueryRowContext(ctx, query, userUUID).Scan(&u.ID, &u.UserUUID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by uuid: %w", err)
	}
	return &u, nil
}

func (r *PostgresPulseRepository) AuthenticateUser(ctx context.Context, email, password string) (*core.PulseUser, error) {
	user, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			return nil, ErrInvalidCredentials
		}
	}
	return user, nil
}

func (r *PostgresPulseRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
