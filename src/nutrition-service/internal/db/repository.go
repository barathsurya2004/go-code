package db

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MemoryNutritionRepository struct {
	mu       sync.RWMutex
	goals    map[string]*core.NutritionGoals // userUUID -> goals
	meals    map[string]*core.MealLog        // mealID -> MealLog
	jobs     map[string]*core.JobStatus      // jobID -> JobStatus
	userJobs map[string][]string             // userUUID -> []jobID
	log      *zap.Logger
}

func NewMemoryNutritionRepository(log *zap.Logger) *MemoryNutritionRepository {
	return &MemoryNutritionRepository{
		goals:    make(map[string]*core.NutritionGoals),
		meals:    make(map[string]*core.MealLog),
		jobs:     make(map[string]*core.JobStatus),
		userJobs: make(map[string][]string),
		log:      log,
	}
}

func defaultGoals(userUUID string) *core.NutritionGoals {
	return &core.NutritionGoals{
		UserUUID:       userUUID,
		CaloriesTarget: 2200,
		ProteinGrams:   160,
		CarbsGrams:     220,
		FatGrams:       65,
		FiberGrams:     30,
		SodiumMg:       2300,
		UpdatedAt:      time.Now(),
	}
}

func (r *MemoryNutritionRepository) GetGoals(ctx context.Context, userUUID string) (*core.NutritionGoals, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, exists := r.goals[userUUID]
	if !exists {
		return defaultGoals(userUUID), nil
	}
	copy := *g
	return &copy, nil
}

func (r *MemoryNutritionRepository) SetGoals(ctx context.Context, goals *core.NutritionGoals) (*core.NutritionGoals, error) {
	if goals == nil || goals.UserUUID == "" {
		return nil, fmt.Errorf("invalid goals data")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	goalsCopy := *goals
	goalsCopy.UpdatedAt = time.Now()
	r.goals[goals.UserUUID] = &goalsCopy
	return &goalsCopy, nil
}

func (r *MemoryNutritionRepository) GetDailyLog(ctx context.Context, userUUID, date string) (*core.DailySummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	goals, exists := r.goals[userUUID]
	if !exists {
		goals = defaultGoals(userUUID)
	}

	summary := &core.DailySummary{
		Date:  date,
		Goals: goals,
		Meals: make([]core.MealLog, 0),
	}

	for _, m := range r.meals {
		if m.UserUUID == userUUID {
			mealDate := m.LoggedAt.Format("2006-01-02")
			if mealDate == date {
				summary.Meals = append(summary.Meals, *m)
				summary.TotalCalories += m.TotalCalories
				summary.TotalProtein += m.TotalProtein
				summary.TotalCarbs += m.TotalCarbs
				summary.TotalFat += m.TotalFat

				for _, item := range m.Items {
					summary.TotalFiber += item.FiberGrams
					summary.TotalSodium += item.SodiumMg
				}
			}
		}
	}

	sort.Slice(summary.Meals, func(i, j int) bool {
		return summary.Meals[i].LoggedAt.Before(summary.Meals[j].LoggedAt)
	})

	return summary, nil
}

func (r *MemoryNutritionRepository) CreateMealLog(ctx context.Context, log *core.MealLog) (*core.MealLog, error) {
	if log == nil {
		return nil, fmt.Errorf("meal log cannot be nil")
	}
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.LoggedAt.IsZero() {
		log.LoggedAt = time.Now()
	}

	var cal, prot, carb, fat float64
	for i := range log.Items {
		if log.Items[i].ID == "" {
			log.Items[i].ID = uuid.New().String()
		}
		log.Items[i].MealLogID = log.ID
		cal += log.Items[i].Calories
		prot += log.Items[i].ProteinGrams
		carb += log.Items[i].CarbsGrams
		fat += log.Items[i].FatGrams
	}
	log.TotalCalories = cal
	log.TotalProtein = prot
	log.TotalCarbs = carb
	log.TotalFat = fat

	r.mu.Lock()
	defer r.mu.Unlock()

	r.meals[log.ID] = log
	copy := *log
	return &copy, nil
}

func (r *MemoryNutritionRepository) DeleteMealLog(ctx context.Context, userUUID, mealLogID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	m, exists := r.meals[mealLogID]
	if !exists || m.UserUUID != userUUID {
		return fmt.Errorf("meal log not found")
	}

	delete(r.meals, mealLogID)
	return nil
}

func (r *MemoryNutritionRepository) UpdateMealItem(ctx context.Context, userUUID string, item *core.MealItem) (*core.MealItem, error) {
	if item == nil || item.ID == "" {
		return nil, fmt.Errorf("invalid meal item")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, meal := range r.meals {
		if meal.UserUUID == userUUID {
			for i, it := range meal.Items {
				if it.ID == item.ID {
					meal.Items[i].Name = item.Name
					meal.Items[i].EstimatedWeightGrams = item.EstimatedWeightGrams
					meal.Items[i].Calories = item.Calories
					meal.Items[i].ProteinGrams = item.ProteinGrams
					meal.Items[i].CarbsGrams = item.CarbsGrams
					meal.Items[i].FatGrams = item.FatGrams
					meal.Items[i].FiberGrams = item.FiberGrams

					// Recalculate meal totals
					var cal, prot, carb, fat float64
					for _, itm := range meal.Items {
						cal += itm.Calories
						prot += itm.ProteinGrams
						carb += itm.CarbsGrams
						fat += itm.FatGrams
					}
					meal.TotalCalories = cal
					meal.TotalProtein = prot
					meal.TotalCarbs = carb
					meal.TotalFat = fat

					updated := meal.Items[i]
					return &updated, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("meal item not found")
}

func (r *MemoryNutritionRepository) DeleteMealItem(ctx context.Context, userUUID, itemID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for mealID, meal := range r.meals {
		if meal.UserUUID == userUUID {
			for i, it := range meal.Items {
				if it.ID == itemID {
					// Remove item
					meal.Items = append(meal.Items[:i], meal.Items[i+1:]...)
					if len(meal.Items) == 0 {
						delete(r.meals, mealID)
					} else {
						// Recalculate totals
						var cal, prot, carb, fat float64
						for _, itm := range meal.Items {
							cal += itm.Calories
							prot += itm.ProteinGrams
							carb += itm.CarbsGrams
							fat += itm.FatGrams
						}
						meal.TotalCalories = cal
						meal.TotalProtein = prot
						meal.TotalCarbs = carb
						meal.TotalFat = fat
					}
					return nil
				}
			}
		}
	}

	return fmt.Errorf("meal item not found")
}

func (r *MemoryNutritionRepository) SaveJob(ctx context.Context, job *core.JobStatus) error {
	if job == nil || job.JobID == "" {
		return fmt.Errorf("invalid job")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.jobs[job.JobID] = job
	return nil
}

func (r *MemoryNutritionRepository) GetJob(ctx context.Context, userUUID, jobID string) (*core.JobStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	j, exists := r.jobs[jobID]
	if !exists || j.UserUUID != userUUID {
		return nil, fmt.Errorf("job not found")
	}

	copy := *j
	return &copy, nil
}
