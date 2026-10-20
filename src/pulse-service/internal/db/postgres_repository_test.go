package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestPostgresPulseRepository(t *testing.T) {
	log := zap.NewNop()
	ctx := context.Background()

	t.Run("EnsureSchema executes schema SQL", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectExec("CREATE TABLE IF NOT EXISTS pulse_users").
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := NewPostgresPulseRepository(db, log)
		err = repo.EnsureSchema(ctx)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload nil payload", func(t *testing.T) {
		repo := NewPostgresPulseRepository(nil, log)
		resp, err := repo.SaveSyncPayload(ctx, nil)
		assert.ErrorIs(t, err, ErrEmptyPayload)
		assert.Nil(t, resp)
	})

	t.Run("SaveSyncPayload empty user_uuid", func(t *testing.T) {
		repo := NewPostgresPulseRepository(nil, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{UserUUID: ""})
		assert.ErrorIs(t, err, ErrInvalidUserUUID)
		assert.Nil(t, resp)
	})

	t.Run("SaveSyncPayload full upsert happy path", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		// 1. User upsert
		mock.ExpectExec("INSERT INTO pulse_users").
			WithArgs("u-1", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 2. Settings upsert
		mock.ExpectExec("INSERT INTO pulse_settings").
			WithArgs("u-1", "kg", float64(20), 90, true, true, true, 4, float64(75), "Barath", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 3. Routine upsert
		mock.ExpectExec("INSERT INTO pulse_routines").
			WithArgs("r-1", "u-1", "Push Day", "Chest", "mint", 45, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 4. Plan upsert
		mock.ExpectExec("INSERT INTO pulse_plans").
			WithArgs("u-1", 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 5. Session upsert
		mock.ExpectExec("INSERT INTO pulse_workout_sessions").
			WithArgs("s-1", "u-1", sqlmock.AnyArg(), "Push Day", "mint", int64(1000), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 6. Skipped days upsert
		mock.ExpectExec("INSERT INTO pulse_skipped_days").
			WithArgs("u-1", "2026-10-20", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 7. Audit log insert
		mock.ExpectExec("INSERT INTO pulse_sync_logs").
			WithArgs(sqlmock.AnyArg(), "u-1", sqlmock.AnyArg(), "sync_batch", "push", int64(1000), 1, 1, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		lastPerf := "2026-10-20T10:00:00Z"
		routineID := "r-1"
		payload := &core.SyncPayload{
			UserUUID:   "u-1",
			ClientTime: 1000,
			Settings: &core.Settings{
				Unit:          "kg",
				BarWeight:     20,
				DefaultRest:   90,
				Haptics:       true,
				Sound:         true,
				AutoStartRest: true,
				WeeklyGoal:    4,
				Bodyweight:    75,
				Name:          "Barath",
			},
			Routines: []core.Routine{
				{
					ID:            "r-1",
					Name:          "Push Day",
					Focus:         "Chest",
					Color:         "mint",
					EstMinutes:    45,
					LastPerformed: &lastPerf,
				},
			},
			Plan: []core.PlanDay{
				{Weekday: 0, RoutineID: &routineID},
			},
			Sessions: []core.WorkoutSession{
				{
					ID:        "s-1",
					Name:      "Push Day",
					Color:     "mint",
					StartedAt: 1000,
				},
			},
			SkippedDates: []string{"2026-10-20"},
		}

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, payload)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, 1, resp.SessionsSynced)
		assert.Equal(t, 1, resp.RoutinesSynced)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload begin tx error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin().WillReturnError(errors.New("connection closed"))

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{UserUUID: "u-1"})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload settings error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO pulse_users").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO pulse_settings").WillReturnError(errors.New("settings fail"))
		mock.ExpectRollback()

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{
			UserUUID: "u-1",
			Settings: &core.Settings{Unit: "kg"},
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload routines error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO pulse_users").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO pulse_routines").WillReturnError(errors.New("routine fail"))
		mock.ExpectRollback()

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{
			UserUUID: "u-1",
			Routines: []core.Routine{{ID: "r-1", Name: "Legs"}},
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload plans error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO pulse_users").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO pulse_plans").WillReturnError(errors.New("plan fail"))
		mock.ExpectRollback()

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{
			UserUUID: "u-1",
			Plan:     []core.PlanDay{{Weekday: 1}},
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload sessions error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO pulse_users").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO pulse_workout_sessions").WillReturnError(errors.New("session fail"))
		mock.ExpectRollback()

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{
			UserUUID: "u-1",
			Sessions: []core.WorkoutSession{{ID: "s-1", Name: "Chest"}},
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload skipped days error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO pulse_users").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO pulse_skipped_days").WillReturnError(errors.New("skipped fail"))
		mock.ExpectRollback()

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{
			UserUUID:     "u-1",
			SkippedDates: []string{"2026-10-20"},
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload sync log error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO pulse_users").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO pulse_sync_logs").WillReturnError(errors.New("sync log fail"))
		mock.ExpectRollback()

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{UserUUID: "u-1"})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveSyncPayload commit error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO pulse_users").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO pulse_sync_logs").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit fail"))

		repo := NewPostgresPulseRepository(db, log)
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{UserUUID: "u-1"})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserStats queries postgres", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT updated_at FROM pulse_users WHERE user_uuid = \\$1 OR id = \\$1").
			WithArgs("u-1").
			WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pulse_workout_sessions").
			WithArgs("u-1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pulse_routines").
			WithArgs("u-1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		repo := NewPostgresPulseRepository(db, log)
		stats, err := repo.GetUserStats(ctx, "u-1")
		assert.NoError(t, err)
		assert.Equal(t, "u-1", stats.UserUUID)
		assert.Equal(t, 5, stats.TotalSessions)
		assert.Equal(t, 2, stats.TotalRoutines)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserStats empty user_uuid", func(t *testing.T) {
		repo := NewPostgresPulseRepository(nil, log)
		stats, err := repo.GetUserStats(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidUserUUID)
		assert.Nil(t, stats)
	})

	t.Run("Close with db", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		mock.ExpectClose()
		repo := NewPostgresPulseRepository(db, log)
		assert.NoError(t, repo.Close())
		assert.NoError(t, mock.ExpectationsWereMet())

		nilRepo := NewPostgresPulseRepository(nil, log)
		assert.NoError(t, nilRepo.Close())
	})

	t.Run("NewPulseRepository fallback on empty DATABASE_URL", func(t *testing.T) {
		orig := os.Getenv("DATABASE_URL")
		os.Setenv("DATABASE_URL", "")
		defer os.Setenv("DATABASE_URL", orig)

		repo := NewPulseRepository(log)
		assert.NotNil(t, repo)
		_, ok := repo.(*MemoryPulseRepository)
		assert.True(t, ok)
	})

	t.Run("NewPulseRepository fallback on unreachable DATABASE_URL", func(t *testing.T) {
		orig := os.Getenv("DATABASE_URL")
		os.Setenv("DATABASE_URL", "postgres://invalid:invalid@127.0.0.1:54399/invalid?sslmode=disable&connect_timeout=1")
		defer os.Setenv("DATABASE_URL", orig)

		repo := NewPulseRepository(log)
		assert.NotNil(t, repo)
		_, ok := repo.(*MemoryPulseRepository)
		assert.True(t, ok)
	})

	t.Run("NewPulseRepository fallback on malformed DATABASE_URL", func(t *testing.T) {
		orig := os.Getenv("DATABASE_URL")
		os.Setenv("DATABASE_URL", "postgres://%gh&invalid")
		defer os.Setenv("DATABASE_URL", orig)

		repo := NewPulseRepository(log)
		assert.NotNil(t, repo)
		_, ok := repo.(*MemoryPulseRepository)
		assert.True(t, ok)
	})
}
