package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type PostgresNutritionRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewPostgresNutritionRepository(db *sql.DB, log *zap.Logger) *PostgresNutritionRepository {
	return &PostgresNutritionRepository{
		db:  db,
		log: log,
	}
}

// NewNutritionRepository returns Postgres repository if DATABASE_URL is available and healthy,
// falling back gracefully to MemoryNutritionRepository.
func NewNutritionRepository(log *zap.Logger) core.NutritionRepository {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Info("No DATABASE_URL configured, using in-memory nutrition repository")
		return NewMemoryNutritionRepository(log)
	}

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Warn("Failed to open PostgreSQL connection, using in-memory fallback", zap.Error(err))
		return NewMemoryNutritionRepository(log)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := dbConn.PingContext(ctx); err != nil {
		log.Warn("PostgreSQL ping failed, using in-memory fallback", zap.Error(err))
		_ = dbConn.Close()
		return NewMemoryNutritionRepository(log)
	}

	repo := NewPostgresNutritionRepository(dbConn, log)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		log.Warn("Failed to verify/migrate schema on nouri_db, using in-memory fallback", zap.Error(err))
		_ = dbConn.Close()
		return NewMemoryNutritionRepository(log)
	}

	log.Info("Connected dedicated PostgreSQL database successfully for Nouri service")
	return repo
}

func (r *PostgresNutritionRepository) EnsureSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS nutrition_goals (
		user_uuid VARCHAR(64) PRIMARY KEY,
		calories_target DOUBLE PRECISION NOT NULL DEFAULT 2200,
		protein_grams DOUBLE PRECISION NOT NULL DEFAULT 160,
		carbs_grams DOUBLE PRECISION NOT NULL DEFAULT 220,
		fat_grams DOUBLE PRECISION NOT NULL DEFAULT 65,
		fiber_grams DOUBLE PRECISION NOT NULL DEFAULT 30,
		sodium_mg DOUBLE PRECISION NOT NULL DEFAULT 2300,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS meal_logs (
		id VARCHAR(64) PRIMARY KEY,
		user_uuid VARCHAR(64) NOT NULL,
		meal_type VARCHAR(32) NOT NULL,
		logged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		notes TEXT DEFAULT '',
		total_calories DOUBLE PRECISION NOT NULL DEFAULT 0,
		total_protein DOUBLE PRECISION NOT NULL DEFAULT 0,
		total_carbs DOUBLE PRECISION NOT NULL DEFAULT 0,
		total_fat DOUBLE PRECISION NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_meal_logs_user_date ON meal_logs(user_uuid, logged_at);

	CREATE TABLE IF NOT EXISTS meal_items (
		id VARCHAR(64) PRIMARY KEY,
		meal_log_id VARCHAR(64) NOT NULL REFERENCES meal_logs(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		estimated_weight_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
		calories DOUBLE PRECISION NOT NULL DEFAULT 0,
		protein_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
		carbs_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
		fat_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
		fiber_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
		sodium_mg DOUBLE PRECISION NOT NULL DEFAULT 0,
		confidence_score DOUBLE PRECISION NOT NULL DEFAULT 0.95,
		ingredients TEXT[] DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_meal_items_log_id ON meal_items(meal_log_id);

	CREATE TABLE IF NOT EXISTS nutrition_jobs (
		job_id VARCHAR(64) PRIMARY KEY,
		user_uuid VARCHAR(64) NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
		error_message TEXT DEFAULT '',
		meal_log_id VARCHAR(64) DEFAULT '',
		detected_items_json JSONB DEFAULT '[]'::jsonb,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_nutrition_jobs_user ON nutrition_jobs(user_uuid, job_id);
	`
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

func (r *PostgresNutritionRepository) GetGoals(ctx context.Context, userUUID string) (*core.NutritionGoals, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_uuid, calories_target, protein_grams, carbs_grams, fat_grams, fiber_grams, sodium_mg, updated_at
		FROM nutrition_goals WHERE user_uuid = $1
	`, userUUID)

	g := &core.NutritionGoals{}
	err := row.Scan(&g.UserUUID, &g.CaloriesTarget, &g.ProteinGrams, &g.CarbsGrams, &g.FatGrams, &g.FiberGrams, &g.SodiumMg, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return defaultGoals(userUUID), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query nutrition goals: %w", err)
	}
	return g, nil
}

func (r *PostgresNutritionRepository) SetGoals(ctx context.Context, goals *core.NutritionGoals) (*core.NutritionGoals, error) {
	if goals == nil || goals.UserUUID == "" {
		return nil, fmt.Errorf("invalid goals or user uuid")
	}

	query := `
		INSERT INTO nutrition_goals (user_uuid, calories_target, protein_grams, carbs_grams, fat_grams, fiber_grams, sodium_mg, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_uuid) DO UPDATE SET
			calories_target = EXCLUDED.calories_target,
			protein_grams = EXCLUDED.protein_grams,
			carbs_grams = EXCLUDED.carbs_grams,
			fat_grams = EXCLUDED.fat_grams,
			fiber_grams = EXCLUDED.fiber_grams,
			sodium_mg = EXCLUDED.sodium_mg,
			updated_at = NOW()
		RETURNING user_uuid, calories_target, protein_grams, carbs_grams, fat_grams, fiber_grams, sodium_mg, updated_at;
	`
	row := r.db.QueryRowContext(ctx, query,
		goals.UserUUID, goals.CaloriesTarget, goals.ProteinGrams, goals.CarbsGrams, goals.FatGrams, goals.FiberGrams, goals.SodiumMg,
	)

	saved := &core.NutritionGoals{}
	if err := row.Scan(&saved.UserUUID, &saved.CaloriesTarget, &saved.ProteinGrams, &saved.CarbsGrams, &saved.FatGrams, &saved.FiberGrams, &saved.SodiumMg, &saved.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to save nutrition goals: %w", err)
	}
	return saved, nil
}

func (r *PostgresNutritionRepository) CreateMealLog(ctx context.Context, log *core.MealLog) (*core.MealLog, error) {
	if log == nil || log.UserUUID == "" {
		return nil, fmt.Errorf("invalid meal log")
	}

	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.LoggedAt.IsZero() {
		log.LoggedAt = time.Now()
	}

	var cal, prot, carb, fat float64
	for _, it := range log.Items {
		cal += it.Calories
		prot += it.ProteinGrams
		carb += it.CarbsGrams
		fat += it.FatGrams
	}
	log.TotalCalories = cal
	log.TotalProtein = prot
	log.TotalCarbs = carb
	log.TotalFat = fat

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO meal_logs (id, user_uuid, meal_type, logged_at, notes, total_calories, total_protein, total_carbs, total_fat)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, log.ID, log.UserUUID, log.MealType, log.LoggedAt, log.Notes, log.TotalCalories, log.TotalProtein, log.TotalCarbs, log.TotalFat)
	if err != nil {
		return nil, fmt.Errorf("failed to insert meal log: %w", err)
	}

	for i := range log.Items {
		item := &log.Items[i]
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		item.MealLogID = log.ID

		_, err = tx.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_log_id, name, estimated_weight_grams, calories, protein_grams, carbs_grams, fat_grams, fiber_grams, sodium_mg, confidence_score, ingredients)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, item.ID, item.MealLogID, item.Name, item.EstimatedWeightGrams, item.Calories, item.ProteinGrams, item.CarbsGrams, item.FatGrams, item.FiberGrams, item.SodiumMg, item.ConfidenceScore, pq.Array(item.Ingredients))
		if err != nil {
			return nil, fmt.Errorf("failed to insert meal item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit meal log: %w", err)
	}

	return log, nil
}

func (r *PostgresNutritionRepository) GetDailyLog(ctx context.Context, userUUID, date string) (*core.DailySummary, error) {
	goals, err := r.GetGoals(ctx, userUUID)
	if err != nil {
		goals = defaultGoals(userUUID)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_uuid, meal_type, logged_at, notes, total_calories, total_protein, total_carbs, total_fat
		FROM meal_logs
		WHERE user_uuid = $1 AND DATE(logged_at AT TIME ZONE 'UTC') = $2::DATE
		ORDER BY logged_at ASC
	`, userUUID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily meal logs: %w", err)
	}
	defer rows.Close()

	meals := make([]core.MealLog, 0)
	mealIDs := make([]string, 0)

	for rows.Next() {
		var m core.MealLog
		if err := rows.Scan(&m.ID, &m.UserUUID, &m.MealType, &m.LoggedAt, &m.Notes, &m.TotalCalories, &m.TotalProtein, &m.TotalCarbs, &m.TotalFat); err != nil {
			return nil, fmt.Errorf("failed to scan meal log: %w", err)
		}
		m.Items = make([]core.MealItem, 0)
		meals = append(meals, m)
		mealIDs = append(mealIDs, m.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during meal logs row scan: %w", err)
	}

	// Fetch items for all retrieved meals
	if len(mealIDs) > 0 {
		itemRows, err := r.db.QueryContext(ctx, `
			SELECT id, meal_log_id, name, estimated_weight_grams, calories, protein_grams, carbs_grams, fat_grams, fiber_grams, sodium_mg, confidence_score, ingredients
			FROM meal_items
			WHERE meal_log_id = ANY($1)
			ORDER BY created_at ASC
		`, pq.Array(mealIDs))
		if err == nil {
			defer itemRows.Close()
			itemMap := make(map[string][]core.MealItem)
			for itemRows.Next() {
				var it core.MealItem
				var ings []string
				if err := itemRows.Scan(&it.ID, &it.MealLogID, &it.Name, &it.EstimatedWeightGrams, &it.Calories, &it.ProteinGrams, &it.CarbsGrams, &it.FatGrams, &it.FiberGrams, &it.SodiumMg, &it.ConfidenceScore, pq.Array(&ings)); err == nil {
					it.Ingredients = ings
					itemMap[it.MealLogID] = append(itemMap[it.MealLogID], it)
				}
			}
			if err := itemRows.Err(); err != nil {
				r.log.Warn("error during meal items row scan", zap.Error(err))
			}
			for i := range meals {
				if items, ok := itemMap[meals[i].ID]; ok {
					meals[i].Items = items
				}
			}
		}
	}

	summary := &core.DailySummary{
		Date:   date,
		Goals:  goals,
		Meals:  meals,
	}

	for _, m := range meals {
		summary.TotalCalories += m.TotalCalories
		summary.TotalProtein += m.TotalProtein
		summary.TotalCarbs += m.TotalCarbs
		summary.TotalFat += m.TotalFat
		for _, it := range m.Items {
			summary.TotalFiber += it.FiberGrams
			summary.TotalSodium += it.SodiumMg
		}
	}

	return summary, nil
}

func (r *PostgresNutritionRepository) DeleteMealLog(ctx context.Context, userUUID, mealLogID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM meal_logs WHERE id = $1 AND user_uuid = $2
	`, mealLogID, userUUID)
	if err != nil {
		return fmt.Errorf("failed to delete meal log: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("meal log not found")
	}
	return nil
}

func (r *PostgresNutritionRepository) UpdateMealItem(ctx context.Context, userUUID string, item *core.MealItem) (*core.MealItem, error) {
	if item == nil || item.ID == "" {
		return nil, fmt.Errorf("invalid item")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Verify ownership through parent meal
	var mealLogID string
	err = tx.QueryRowContext(ctx, `
		SELECT m.id FROM meal_logs m
		INNER JOIN meal_items i ON i.meal_log_id = m.id
		WHERE i.id = $1 AND m.user_uuid = $2
	`, item.ID, userUUID).Scan(&mealLogID)
	if err != nil {
		return nil, fmt.Errorf("meal item not found or unauthorized: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE meal_items SET
			name = $1,
			estimated_weight_grams = $2,
			calories = $3,
			protein_grams = $4,
			carbs_grams = $5,
			fat_grams = $6,
			fiber_grams = $7
		WHERE id = $8
	`, item.Name, item.EstimatedWeightGrams, item.Calories, item.ProteinGrams, item.CarbsGrams, item.FatGrams, item.FiberGrams, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update meal item: %w", err)
	}

	// Recalculate parent meal totals
	_, err = tx.ExecContext(ctx, `
		UPDATE meal_logs SET
			total_calories = COALESCE((SELECT SUM(calories) FROM meal_items WHERE meal_log_id = $1), 0),
			total_protein = COALESCE((SELECT SUM(protein_grams) FROM meal_items WHERE meal_log_id = $1), 0),
			total_carbs = COALESCE((SELECT SUM(carbs_grams) FROM meal_items WHERE meal_log_id = $1), 0),
			total_fat = COALESCE((SELECT SUM(fat_grams) FROM meal_items WHERE meal_log_id = $1), 0)
		WHERE id = $1
	`, mealLogID)
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate meal totals: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return item, nil
}

func (r *PostgresNutritionRepository) DeleteMealItem(ctx context.Context, userUUID, itemID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var mealLogID string
	err = tx.QueryRowContext(ctx, `
		SELECT m.id FROM meal_logs m
		INNER JOIN meal_items i ON i.meal_log_id = m.id
		WHERE i.id = $1 AND m.user_uuid = $2
	`, itemID, userUUID).Scan(&mealLogID)
	if err != nil {
		return fmt.Errorf("meal item not found or unauthorized: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM meal_items WHERE id = $1`, itemID); err != nil {
		return fmt.Errorf("failed to delete meal item: %w", err)
	}

	// Check if parent meal still has any items
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM meal_items WHERE meal_log_id = $1`, mealLogID).Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM meal_logs WHERE id = $1`, mealLogID); err != nil {
			return err
		}
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE meal_logs SET
				total_calories = COALESCE((SELECT SUM(calories) FROM meal_items WHERE meal_log_id = $1), 0),
				total_protein = COALESCE((SELECT SUM(protein_grams) FROM meal_items WHERE meal_log_id = $1), 0),
				total_carbs = COALESCE((SELECT SUM(carbs_grams) FROM meal_items WHERE meal_log_id = $1), 0),
				total_fat = COALESCE((SELECT SUM(fat_grams) FROM meal_items WHERE meal_log_id = $1), 0)
			WHERE id = $1
		`, mealLogID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresNutritionRepository) SaveJob(ctx context.Context, job *core.JobStatus) error {
	if job == nil || job.JobID == "" {
		return fmt.Errorf("invalid job")
	}

	itemsJSON, err := json.Marshal(job.DetectedItems)
	if err != nil {
		itemsJSON = []byte("[]")
	}

	query := `
		INSERT INTO nutrition_jobs (job_id, user_uuid, status, error_message, meal_log_id, detected_items_json, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (job_id) DO UPDATE SET
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			meal_log_id = EXCLUDED.meal_log_id,
			detected_items_json = EXCLUDED.detected_items_json,
			updated_at = NOW()
	`
	_, err = r.db.ExecContext(ctx, query,
		job.JobID, job.UserUUID, job.Status, job.ErrorMessage, job.MealLogID, string(itemsJSON), job.CreatedAt,
	)
	return err
}

func (r *PostgresNutritionRepository) GetJob(ctx context.Context, userUUID, jobID string) (*core.JobStatus, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT job_id, user_uuid, status, error_message, meal_log_id, detected_items_json, created_at, updated_at
		FROM nutrition_jobs
		WHERE job_id = $1 AND user_uuid = $2
	`, jobID, userUUID)

	job := &core.JobStatus{}
	var itemsJSON string
	err := row.Scan(&job.JobID, &job.UserUUID, &job.Status, &job.ErrorMessage, &job.MealLogID, &itemsJSON, &job.CreatedAt, &job.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job: %w", err)
	}

	if itemsJSON != "" {
		_ = json.Unmarshal([]byte(itemsJSON), &job.DetectedItems)
	}
	return job, nil
}
