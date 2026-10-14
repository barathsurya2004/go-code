package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPostgresNutritionRepository_Goals(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	repo := NewPostgresNutritionRepository(db, logger)
	ctx := context.Background()
	userUUID := "user-test-123"

	// 1. GetGoals - not found -> default goals
	mock.ExpectQuery(`SELECT user_uuid, calories_target.*FROM nutrition_goals WHERE user_uuid = \$1`).
		WithArgs(userUUID).
		WillReturnError(sql.ErrNoRows)

	goals, err := repo.GetGoals(ctx, userUUID)
	require.NoError(t, err)
	assert.Equal(t, 2200.0, goals.CaloriesTarget)

	// 2. SetGoals
	targetGoals := &core.NutritionGoals{
		UserUUID:       userUUID,
		CaloriesTarget: 2500,
		ProteinGrams:   180,
		CarbsGrams:     260,
		FatGrams:       75,
		FiberGrams:     35,
		SodiumMg:       2200,
	}

	mock.ExpectQuery(`INSERT INTO nutrition_goals.*VALUES.*ON CONFLICT.*RETURNING`).
		WithArgs(userUUID, 2500.0, 180.0, 260.0, 75.0, 35.0, 2200.0).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_uuid", "calories_target", "protein_grams", "carbs_grams", "fat_grams", "fiber_grams", "sodium_mg", "updated_at",
		}).AddRow(userUUID, 2500.0, 180.0, 260.0, 75.0, 35.0, 2200.0, time.Now()))

	saved, err := repo.SetGoals(ctx, targetGoals)
	require.NoError(t, err)
	assert.Equal(t, 2500.0, saved.CaloriesTarget)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresNutritionRepository_Meals(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	repo := NewPostgresNutritionRepository(db, logger)
	ctx := context.Background()
	userUUID := "user-test-123"

	// 1. CreateMealLog
	meal := &core.MealLog{
		UserUUID: userUUID,
		MealType: "lunch",
		LoggedAt: time.Now(),
		Notes:    "Healthy bowl",
		Items: []core.MealItem{
			{
				Name:                 "Chicken & Rice",
				EstimatedWeightGrams: 350,
				Calories:             500,
				ProteinGrams:         45,
				CarbsGrams:           50,
				FatGrams:             10,
				FiberGrams:           5,
				SodiumMg:             400,
				ConfidenceScore:      0.98,
				Ingredients:          []string{"Chicken", "Rice"},
			},
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO meal_logs`).
		WithArgs(sqlmock.AnyArg(), userUUID, "lunch", meal.LoggedAt, "Healthy bowl", 500.0, 45.0, 50.0, 10.0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO meal_items`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Chicken & Rice", 350.0, 500.0, 45.0, 50.0, 10.0, 5.0, 400.0, 0.98, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	created, err := repo.CreateMealLog(ctx, meal)
	require.NoError(t, err)
	assert.Equal(t, 500.0, created.TotalCalories)

	// 2. DeleteMealLog
	mock.ExpectExec(`DELETE FROM meal_logs WHERE id = \$1 AND user_uuid = \$2`).
		WithArgs(created.ID, userUUID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.DeleteMealLog(ctx, userUUID, created.ID)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresNutritionRepository_Jobs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zap.NewNop()
	repo := NewPostgresNutritionRepository(db, logger)
	ctx := context.Background()
	userUUID := "user-test-123"
	jobID := "job-xyz-456"

	// 1. SaveJob
	mock.ExpectExec(`INSERT INTO nutrition_jobs`).
		WithArgs(jobID, userUUID, "PROCESSING", "", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveJob(ctx, &core.JobStatus{
		JobID:     jobID,
		UserUUID:  userUUID,
		Status:    "PROCESSING",
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)

	// 2. GetJob
	mock.ExpectQuery(`SELECT job_id, user_uuid, status, error_message, meal_log_id, detected_items_json, created_at, updated_at FROM nutrition_jobs WHERE job_id = \$1 AND user_uuid = \$2`).
		WithArgs(jobID, userUUID).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "user_uuid", "status", "error_message", "meal_log_id", "detected_items_json", "created_at", "updated_at",
		}).AddRow(jobID, userUUID, "PROCESSING", "", "", "[]", time.Now(), time.Now()))

	retrieved, err := repo.GetJob(ctx, userUUID, jobID)
	require.NoError(t, err)
	assert.Equal(t, "PROCESSING", retrieved.Status)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewNutritionRepository_Fallback(t *testing.T) {
	logger := zap.NewNop()
	// When DATABASE_URL is unset, returns memory repository
	repo := NewNutritionRepository(logger)
	assert.NotNil(t, repo)

	goals, err := repo.GetGoals(context.Background(), "any-user")
	require.NoError(t, err)
	assert.Equal(t, 2200.0, goals.CaloriesTarget)
}
