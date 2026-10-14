package db

import (
	"context"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"go.uber.org/zap"
)

func TestMemoryNutritionRepository(t *testing.T) {
	logger := zap.NewNop()
	repo := NewMemoryNutritionRepository(logger)
	ctx := context.Background()
	userUUID := "user-123"

	// 1. Get default goals
	goals, err := repo.GetGoals(ctx, userUUID)
	if err != nil {
		t.Fatalf("unexpected error getting default goals: %v", err)
	}
	if goals.CaloriesTarget != 2200 {
		t.Errorf("expected 2200 cal, got %v", goals.CaloriesTarget)
	}

	// 2. Set custom goals
	newGoals := &core.NutritionGoals{
		UserUUID:       userUUID,
		CaloriesTarget: 2500,
		ProteinGrams:   180,
		CarbsGrams:     250,
		FatGrams:       70,
		FiberGrams:     35,
		SodiumMg:       2000,
	}
	savedGoals, err := repo.SetGoals(ctx, newGoals)
	if err != nil {
		t.Fatalf("unexpected error setting goals: %v", err)
	}
	if savedGoals.CaloriesTarget != 2500 {
		t.Errorf("expected 2500, got %v", savedGoals.CaloriesTarget)
	}

	// Get goals when exists
	retrievedGoals, err := repo.GetGoals(ctx, userUUID)
	if err != nil || retrievedGoals.CaloriesTarget != 2500 {
		t.Errorf("expected 2500, got %v", retrievedGoals)
	}

	// Test invalid goals
	if _, err := repo.SetGoals(ctx, nil); err == nil {
		t.Error("expected error for nil goals")
	}

	// 3. Create meal log with 2 items
	meal := &core.MealLog{
		UserUUID: userUUID,
		MealType: "lunch",
		LoggedAt: time.Now(),
		Notes:    "Post-workout lunch",
		Items: []core.MealItem{
			{
				Name:                 "Chicken & Rice",
				EstimatedWeightGrams: 300,
				Calories:             500,
				ProteinGrams:         50,
				CarbsGrams:           60,
				FatGrams:             8,
				FiberGrams:           4,
				SodiumMg:             300,
			},
			{
				Name:                 "Greek Salad",
				EstimatedWeightGrams: 150,
				Calories:             150,
				ProteinGrams:         5,
				CarbsGrams:           8,
				FatGrams:             10,
				FiberGrams:           3,
				SodiumMg:             250,
			},
		},
	}
	savedMeal, err := repo.CreateMealLog(ctx, meal)
	if err != nil {
		t.Fatalf("unexpected error creating meal: %v", err)
	}
	if savedMeal.TotalCalories != 650 {
		t.Errorf("expected 650 cal, got %v", savedMeal.TotalCalories)
	}

	// Add an older meal on the same day to test sorting
	olderMeal := &core.MealLog{
		UserUUID: userUUID,
		MealType: "breakfast",
		LoggedAt: time.Now().Add(-1 * time.Minute),
		Items: []core.MealItem{
			{Name: "Oatmeal", Calories: 200},
		},
	}
	_, _ = repo.CreateMealLog(ctx, olderMeal)

	// Add a meal for another user or different date
	otherMeal := &core.MealLog{
		UserUUID: "other-user",
		MealType: "breakfast",
		LoggedAt: time.Now(),
	}
	_, _ = repo.CreateMealLog(ctx, otherMeal)

	diffDateMeal := &core.MealLog{
		UserUUID: userUUID,
		MealType: "breakfast",
		LoggedAt: time.Now().Add(-48 * time.Hour),
	}
	_, _ = repo.CreateMealLog(ctx, diffDateMeal)

	// Test nil meal log
	if _, err := repo.CreateMealLog(ctx, nil); err == nil {
		t.Error("expected error for nil meal log")
	}

	// 4. Get daily log
	today := time.Now().Format("2006-01-02")
	summary, err := repo.GetDailyLog(ctx, userUUID, today)
	if err != nil {
		t.Fatalf("unexpected error getting daily log: %v", err)
	}
	if summary.TotalCalories != 850 {
		t.Errorf("expected 850 total calories, got %v", summary.TotalCalories)
	}
	if len(summary.Meals) != 2 {
		t.Errorf("expected 2 meals, got %d", len(summary.Meals))
	}

	// 5. Update meal item
	item := savedMeal.Items[0]
	item.Calories = 550
	item.ProteinGrams = 55
	updatedItem, err := repo.UpdateMealItem(ctx, userUUID, &item)
	if err != nil {
		t.Fatalf("unexpected error updating meal item: %v", err)
	}
	if updatedItem.Calories != 550 {
		t.Errorf("expected 550 cal, got %v", updatedItem.Calories)
	}

	// Test invalid update item
	if _, err := repo.UpdateMealItem(ctx, userUUID, nil); err == nil {
		t.Error("expected error for nil item update")
	}
	if _, err := repo.UpdateMealItem(ctx, userUUID, &core.MealItem{ID: "non-existent"}); err == nil {
		t.Error("expected error for non-existent item update")
	}

	// 6. Delete 1 item from multi-item meal (keeps remaining item)
	saladItem := savedMeal.Items[1]
	if err := repo.DeleteMealItem(ctx, userUUID, saladItem.ID); err != nil {
		t.Fatalf("unexpected error deleting salad item: %v", err)
	}

	// Now delete the last remaining item (removes entire meal)
	if err := repo.DeleteMealItem(ctx, userUUID, item.ID); err != nil {
		t.Fatalf("unexpected error deleting last item: %v", err)
	}

	// Verify non-existent delete
	if err := repo.DeleteMealItem(ctx, userUUID, "non-existent"); err == nil {
		t.Error("expected error deleting non-existent item")
	}

	// 7. Save & get job
	job := &core.JobStatus{
		JobID:    "job-1",
		UserUUID: userUUID,
		Status:   "PENDING",
	}
	if err := repo.SaveJob(ctx, job); err != nil {
		t.Fatalf("unexpected error saving job: %v", err)
	}
	retrievedJob, err := repo.GetJob(ctx, userUUID, "job-1")
	if err != nil {
		t.Fatalf("unexpected error getting job: %v", err)
	}
	if retrievedJob.Status != "PENDING" {
		t.Errorf("expected PENDING, got %s", retrievedJob.Status)
	}

	// Test invalid job operations
	if err := repo.SaveJob(ctx, nil); err == nil {
		t.Error("expected error for nil job")
	}
	if _, err := repo.GetJob(ctx, userUUID, "non-existent"); err == nil {
		t.Error("expected error for non-existent job")
	}

	// 8. Delete meal log
	dummyMeal, _ := repo.CreateMealLog(ctx, &core.MealLog{
		UserUUID: userUUID,
		MealType: "dinner",
	})
	if err := repo.DeleteMealLog(ctx, userUUID, dummyMeal.ID); err != nil {
		t.Fatalf("unexpected error deleting meal log: %v", err)
	}
	if err := repo.DeleteMealLog(ctx, userUUID, "non-existent"); err == nil {
		t.Error("expected error deleting non-existent meal log")
	}
}
