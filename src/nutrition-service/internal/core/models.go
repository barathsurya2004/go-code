package core

import (
	"context"
	"time"
)

type NutritionGoals struct {
	UserUUID       string    `json:"user_uuid"`
	CaloriesTarget float64   `json:"calories_target"`
	ProteinGrams   float64   `json:"protein_grams"`
	CarbsGrams     float64   `json:"carbs_grams"`
	FatGrams       float64   `json:"fat_grams"`
	FiberGrams     float64   `json:"fiber_grams"`
	SodiumMg       float64   `json:"sodium_mg"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MealItem struct {
	ID                   string   `json:"id"`
	MealLogID            string   `json:"meal_log_id"`
	Name                 string   `json:"name"`
	EstimatedWeightGrams float64  `json:"estimated_weight_grams"`
	Calories             float64  `json:"calories"`
	ProteinGrams         float64  `json:"protein_grams"`
	CarbsGrams           float64  `json:"carbs_grams"`
	FatGrams             float64  `json:"fat_grams"`
	FiberGrams           float64  `json:"fiber_grams"`
	SodiumMg             float64  `json:"sodium_mg"`
	ConfidenceScore      float64  `json:"confidence_score"`
	Ingredients          []string `json:"ingredients"`
}

type MealLog struct {
	ID            string     `json:"id"`
	UserUUID      string     `json:"user_uuid"`
	MealType      string     `json:"meal_type"` // breakfast, lunch, dinner, snack
	LoggedAt      time.Time  `json:"logged_at"`
	Notes         string     `json:"notes"`
	Items         []MealItem `json:"items"`
	TotalCalories float64    `json:"total_calories"`
	TotalProtein  float64    `json:"total_protein"`
	TotalCarbs    float64    `json:"total_carbs"`
	TotalFat      float64    `json:"total_fat"`
}

type JobStatus struct {
	JobID         string     `json:"job_id"`
	UserUUID      string     `json:"user_uuid"`
	Status        string     `json:"status"` // PENDING, PROCESSING, COMPLETED, FAILED
	ErrorMessage  string     `json:"error_message"`
	MealLogID     string     `json:"meal_log_id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DetectedItems []MealItem `json:"detected_items"`
}

type DailySummary struct {
	Date          string          `json:"date"` // YYYY-MM-DD
	TotalCalories float64         `json:"total_calories"`
	TotalProtein  float64         `json:"total_protein"`
	TotalCarbs    float64         `json:"total_carbs"`
	TotalFat      float64         `json:"total_fat"`
	TotalFiber    float64         `json:"total_fiber"`
	TotalSodium   float64         `json:"total_sodium"`
	Goals         *NutritionGoals `json:"goals"`
	Meals         []MealLog       `json:"meals"`
}

type NutritionRepository interface {
	GetGoals(ctx context.Context, userUUID string) (*NutritionGoals, error)
	SetGoals(ctx context.Context, goals *NutritionGoals) (*NutritionGoals, error)
	GetDailyLog(ctx context.Context, userUUID, date string) (*DailySummary, error)
	CreateMealLog(ctx context.Context, log *MealLog) (*MealLog, error)
	DeleteMealLog(ctx context.Context, userUUID, mealLogID string) error
	UpdateMealItem(ctx context.Context, userUUID string, item *MealItem) (*MealItem, error)
	DeleteMealItem(ctx context.Context, userUUID, itemID string) error
	SaveJob(ctx context.Context, job *JobStatus) error
	GetJob(ctx context.Context, userUUID, jobID string) (*JobStatus, error)
}

type AIParser interface {
	AnalyzeMealImage(ctx context.Context, imageData []byte, mimeType string, notes ...string) ([]MealItem, error)
}

type ImageQueueWorker interface {
	Enqueue(ctx context.Context, jobID, userUUID string, imageData []byte, mimeType, mealType, notes string) error
	Start(ctx context.Context)
	Stop()
}
