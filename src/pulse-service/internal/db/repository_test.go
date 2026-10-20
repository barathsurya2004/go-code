package db

import (
	"context"
	"testing"

	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMemoryPulseRepository(t *testing.T) {
	log := zap.NewNop()
	repo := NewMemoryPulseRepository(log)
	ctx := context.Background()

	t.Run("nil payload returns error", func(t *testing.T) {
		resp, err := repo.SaveSyncPayload(ctx, nil)
		assert.ErrorIs(t, err, ErrEmptyPayload)
		assert.Nil(t, resp)
	})

	t.Run("empty user_uuid returns error", func(t *testing.T) {
		resp, err := repo.SaveSyncPayload(ctx, &core.SyncPayload{
			UserUUID: "",
		})
		assert.ErrorIs(t, err, ErrInvalidUserUUID)
		assert.Nil(t, resp)
	})

	t.Run("valid payload saves routines, sessions, plans, and settings", func(t *testing.T) {
		lastPerf := "2026-10-20T12:00:00Z"
		rpe := 8.5
		mood := 4
		notes := "Great leg day"
		routineID := "r-1"

		payload := &core.SyncPayload{
			UserUUID:   "u-123",
			ClientTime: 1729430000000,
			Settings: &core.Settings{
				Unit:        "kg",
				BarWeight:   20,
				DefaultRest: 90,
				Name:        "Barath",
			},
			Routines: []core.Routine{
				{
					ID:            "r-1",
					Name:          "Upper Body A",
					Focus:         "Chest & Back",
					Color:         "mint",
					EstMinutes:    50,
					LastPerformed: &lastPerf,
					Blocks: []core.RoutineBlock{
						{
							ID:         "b-1",
							ExerciseID: "bench_press",
							Rest:       120,
							Sets: []core.PlannedSet{
								{ID: "ps-1", Type: "working", Reps: 8, Weight: 80},
							},
						},
					},
				},
			},
			Plan: []core.PlanDay{
				{Weekday: 0, RoutineID: &routineID},
			},
			Sessions: []core.WorkoutSession{
				{
					ID:        "s-1",
					RoutineID: &routineID,
					Name:      "Upper Body A",
					Color:     "mint",
					StartedAt: 1729430000000,
					RPE:       &rpe,
					Mood:      &mood,
					Notes:     &notes,
					Blocks: []core.SessionBlock{
						{
							ID:         "sb-1",
							ExerciseID: "bench_press",
							Rest:       120,
							Sets: []core.LoggedSet{
								{ID: "ls-1", Type: "working", Reps: 8, Weight: 80, Done: true},
							},
						},
					},
				},
			},
			SkippedDates: []string{"2026-10-19"},
		}

		resp, err := repo.SaveSyncPayload(ctx, payload)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, 1, resp.SessionsSynced)
		assert.Equal(t, 1, resp.RoutinesSynced)

		stats, err := repo.GetUserStats(ctx, "u-123")
		assert.NoError(t, err)
		assert.Equal(t, "u-123", stats.UserUUID)
		assert.Equal(t, 1, stats.TotalSessions)
		assert.Equal(t, 1, stats.TotalRoutines)
		assert.False(t, stats.LastSyncedAt.IsZero())
	})

	t.Run("GetUserStats empty user_uuid", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidUserUUID)
		assert.Nil(t, stats)
	})

	t.Run("GetUserStats non-existent user returns zeroes", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, "non-existent")
		assert.NoError(t, err)
		assert.Equal(t, 0, stats.TotalSessions)
		assert.Equal(t, 0, stats.TotalRoutines)
	})

	t.Run("Close returns nil", func(t *testing.T) {
		assert.NoError(t, repo.Close())
	})
}
