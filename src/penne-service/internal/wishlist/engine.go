package wishlist

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WishlistEngine struct {
	repos  core.RepoContainer
	db     *sql.DB
	logger *zap.Logger
}

func NewWishlistEngine(repos core.RepoContainer, db *sql.DB, logger *zap.Logger) *WishlistEngine {
	return &WishlistEngine{
		repos:  repos,
		db:     db,
		logger: logger,
	}
}

// GetCycleDateRange calculates the start and end of the billing/budget cycle.
func (e *WishlistEngine) GetCycleDateRange(salaryDay int, targetDate time.Time) (time.Time, time.Time) {
	loc := targetDate.Location()
	year, month, day := targetDate.Date()

	if salaryDay <= 1 {
		start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
		end := time.Date(year, month+1, 1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
		return start, end
	}

	if salaryDay > 28 {
		// Cap at 28 to prevent invalid dates in shorter months (e.g. Feb)
		salaryDay = 28
	}

	if day >= salaryDay {
		start := time.Date(year, month, salaryDay, 0, 0, 0, 0, loc)
		end := time.Date(year, month+1, salaryDay, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
		return start, end
	}

	start := time.Date(year, month-1, salaryDay, 0, 0, 0, 0, loc)
	end := time.Date(year, month, salaryDay, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
	return start, end
}

// CalculateCycleSurplus computes the unspent surplus for the current user's cycle.
func (e *WishlistEngine) CalculateCycleSurplus(ctx context.Context, userUUID uuid.UUID, asOf time.Time) (int64, int64, int64, time.Time, time.Time, error) {
	if userUUID == uuid.Nil {
		return 0, 0, 0, time.Time{}, time.Time{}, errors.New("user UUID is required")
	}

	user, err := e.repos.User.GetUserByUUID(userUUID)
	if err != nil {
		e.logger.Error("Failed to fetch user for cycle surplus calculation", zap.Error(err), zap.String("user_uuid", userUUID.String()))
		return 0, 0, 0, time.Time{}, time.Time{}, err
	}

	cycleStart, cycleEnd := e.GetCycleDateRange(user.SalaryDay, asOf)

	query := `
		SELECT COALESCE(SUM(amount_e5), 0)
		FROM transactionrows
		WHERE user_id = $1 AND txn_type = 'debit' AND created_at BETWEEN $2 AND $3
	`

	var cycleExpenses int64
	err = e.db.QueryRowContext(ctx, query, userUUID, cycleStart, cycleEnd).Scan(&cycleExpenses)
	if err != nil {
		e.logger.Error("Failed to query cycle expenses", zap.Error(err), zap.String("user_uuid", userUUID.String()))
		return 0, 0, 0, time.Time{}, time.Time{}, err
	}

	surplus := user.MonthlyBudgetE5 - cycleExpenses
	if surplus < 0 {
		surplus = 0
	}

	return user.MonthlyBudgetE5, cycleExpenses, surplus, cycleStart, cycleEnd, nil
}

// SimulateDistribution distributes the given surplus across active items based on Priority * Urgency weighting with overflow cascading.
func (e *WishlistEngine) SimulateDistribution(items []*core.WishlistItem, surplusE5 int64) []*core.ItemAllocationSimulation {
	allocations := make(map[uuid.UUID]int64)
	remainingSurplus := surplusE5

	if remainingSurplus > 0 && len(items) > 0 {
		for remainingSurplus > 0 {
			var eligibleItems []*core.WishlistItem
			totalWeight := 0

			for _, item := range items {
				if item.Status != "active" {
					continue
				}
				needed := (item.TargetAmountE5 - item.SavedAmountE5) - allocations[item.ID]
				if needed > 0 {
					eligibleItems = append(eligibleItems, item)
					w := item.Priority * item.Urgency
					if w <= 0 {
						w = 1
					}
					totalWeight += w
				}
			}

			if len(eligibleItems) == 0 || totalWeight == 0 {
				break
			}

			allocatedInRound := int64(0)
			roundSurplus := remainingSurplus

			for _, item := range eligibleItems {
				w := item.Priority * item.Urgency
				if w <= 0 {
					w = 1
				}

				share := (roundSurplus * int64(w)) / int64(totalWeight)
				needed := (item.TargetAmountE5 - item.SavedAmountE5) - allocations[item.ID]
				if share > needed {
					share = needed
				}
				if share > remainingSurplus {
					share = remainingSurplus
				}

				allocations[item.ID] += share
				allocatedInRound += share
				remainingSurplus -= share
			}

			if allocatedInRound == 0 {
				if remainingSurplus > 0 && len(eligibleItems) > 0 {
					needed := (eligibleItems[0].TargetAmountE5 - eligibleItems[0].SavedAmountE5) - allocations[eligibleItems[0].ID]
					grant := remainingSurplus
					if grant > needed {
						grant = needed
					}
					allocations[eligibleItems[0].ID] += grant
					remainingSurplus -= grant
				}
				break
			}
		}
	}

	var results []*core.ItemAllocationSimulation
	for _, item := range items {
		alloc := allocations[item.ID]
		newSaved := item.SavedAmountE5 + alloc
		w := item.Priority * item.Urgency
		if w <= 0 {
			w = 1
		}

		results = append(results, &core.ItemAllocationSimulation{
			ItemID:          item.ID,
			ItemTitle:       item.Title,
			AllocatedE5:     alloc,
			PreviousSavedE5: item.SavedAmountE5,
			NewSavedE5:      newSaved,
			TargetAmountE5:  item.TargetAmountE5,
			IsFulfilled:     newSaved >= item.TargetAmountE5,
			Weight:          w,
		})
	}

	return results
}

// CalculateForecasts computes progress %, expected monthly contribution, estimated months to fulfill, and ETA for each item.
func (e *WishlistEngine) CalculateForecasts(items []*core.WishlistItem, monthlyBudgetE5 int64, cycleExpensesE5 int64, asOf time.Time, salaryDay int) *core.WishlistForecastSummary {
	cycleStart, cycleEnd := e.GetCycleDateRange(salaryDay, asOf)

	projectedSurplus := monthlyBudgetE5 - cycleExpensesE5
	if projectedSurplus < 0 {
		projectedSurplus = 0
	}

	savingsRate := 0.0
	if monthlyBudgetE5 > 0 {
		savingsRate = (float64(projectedSurplus) / float64(monthlyBudgetE5)) * 100.0
	}

	simulations := e.SimulateDistribution(items, projectedSurplus)
	simMap := make(map[uuid.UUID]*core.ItemAllocationSimulation)
	for _, s := range simulations {
		simMap[s.ItemID] = s
	}

	var itemForecasts []*core.ItemForecast
	for _, item := range items {
		remaining := item.TargetAmountE5 - item.SavedAmountE5
		if remaining < 0 {
			remaining = 0
		}

		progress := 0.0
		if item.TargetAmountE5 > 0 {
			progress = (float64(item.SavedAmountE5) / float64(item.TargetAmountE5)) * 100.0
			if progress > 100.0 {
				progress = 100.0
			}
		}

		w := item.Priority * item.Urgency
		if w <= 0 {
			w = 1
		}

		monthlyContrib := int64(0)
		if sim, ok := simMap[item.ID]; ok {
			monthlyContrib = sim.AllocatedE5
		}

		estimatedMonths := -1.0
		var estimatedDate *time.Time

		if remaining == 0 || item.Status == "fulfilled" {
			estimatedMonths = 0
			estimatedDate = &asOf
		} else if monthlyContrib > 0 {
			estimatedMonths = float64(remaining) / float64(monthlyContrib)
			ceilMonths := int(math.Ceil(estimatedMonths))
			targetDate := asOf.AddDate(0, ceilMonths, 0)
			estimatedDate = &targetDate
		}

		itemForecasts = append(itemForecasts, &core.ItemForecast{
			ItemID:                item.ID,
			ItemTitle:             item.Title,
			TargetAmountE5:        item.TargetAmountE5,
			SavedAmountE5:         item.SavedAmountE5,
			RemainingAmountE5:     remaining,
			Priority:              item.Priority,
			Urgency:               item.Urgency,
			Weight:                w,
			MonthlyContributionE5: monthlyContrib,
			EstimatedMonths:       estimatedMonths,
			EstimatedDate:         estimatedDate,
			ProgressPercentage:    progress,
		})
	}

	return &core.WishlistForecastSummary{
		CycleStartDate:     cycleStart,
		CycleEndDate:       cycleEnd,
		MonthlyBudgetE5:    monthlyBudgetE5,
		CycleExpensesE5:    cycleExpensesE5,
		ProjectedSurplusE5: projectedSurplus,
		SavingsRatePercent: savingsRate,
		Items:              itemForecasts,
	}
}

// ApplySurplusDistribution executes the surplus allocation and commits the allocations & updated saved amounts.
func (e *WishlistEngine) ApplySurplusDistribution(ctx context.Context, userUUID uuid.UUID, asOf time.Time) ([]*core.ItemAllocationSimulation, error) {
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	_, _, surplus, _, _, err := e.CalculateCycleSurplus(ctx, userUUID, asOf)
	if err != nil {
		return nil, err
	}

	if surplus <= 0 {
		return nil, errors.New("no surplus available to distribute")
	}

	items, err := e.repos.Wishlist.GetActiveWishlistItemsByUserUUID(userUUID)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, errors.New("no active wishlist items found")
	}

	simulations := e.SimulateDistribution(items, surplus)

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		e.logger.Error("Failed to begin transaction for surplus allocation", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	cycleDate := asOf
	for _, sim := range simulations {
		if sim.AllocatedE5 <= 0 {
			continue
		}

		alloc := &core.WishlistAllocation{
			WishlistItemID: sim.ItemID,
			UserUUID:       userUUID,
			AmountE5:       sim.AllocatedE5,
			SourceType:     "cycle_surplus",
			CycleDate:      cycleDate,
		}

		if _, err := e.repos.Wishlist.CreateWishlistAllocation(alloc, tx); err != nil {
			e.logger.Error("Failed to create wishlist allocation", zap.Error(err), zap.String("item_id", sim.ItemID.String()))
			return nil, err
		}

		item, err := e.repos.Wishlist.GetWishlistItemByID(sim.ItemID)
		if err != nil {
			e.logger.Error("Failed to retrieve item for update", zap.Error(err), zap.String("item_id", sim.ItemID.String()))
			return nil, err
		}

		item.SavedAmountE5 = sim.NewSavedE5
		if sim.IsFulfilled {
			item.Status = "fulfilled"
		}

		if err := e.repos.Wishlist.UpdateWishlistItem(item, tx); err != nil {
			e.logger.Error("Failed to update item saved amount", zap.Error(err), zap.String("item_id", sim.ItemID.String()))
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		e.logger.Error("Failed to commit surplus allocation transaction", zap.Error(err))
		return nil, err
	}

	e.logger.Info("Successfully applied wishlist surplus distribution",
		zap.String("user_uuid", userUUID.String()),
		zap.Int64("surplus_e5", surplus),
		zap.Int("items_funded", len(simulations)),
	)

	return simulations, nil
}

// ApplyManualAllocation allocates funds to a specific wishlist item, records the allocation, and updates saved amount/status.
func (e *WishlistEngine) ApplyManualAllocation(ctx context.Context, userUUID uuid.UUID, itemID uuid.UUID, amountE5 int64, asOf time.Time) (*core.ItemAllocationSimulation, error) {
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}
	if itemID == uuid.Nil {
		return nil, errors.New("item ID is required")
	}

	item, err := e.repos.Wishlist.GetWishlistItemByID(itemID)
	if err != nil {
		e.logger.Error("Failed to fetch wishlist item for allocation", zap.Error(err), zap.String("item_id", itemID.String()))
		return nil, err
	}

	if item.UserUUID != userUUID {
		return nil, errors.New("item does not belong to user")
	}

	needed := item.TargetAmountE5 - item.SavedAmountE5
	if needed <= 0 {
		return nil, errors.New("wishlist item is already fulfilled")
	}

	if amountE5 <= 0 {
		amountE5 = needed
	}

	prevSaved := item.SavedAmountE5
	newSaved := prevSaved + amountE5
	isFulfilled := newSaved >= item.TargetAmountE5

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		e.logger.Error("Failed to begin transaction for manual allocation", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	alloc := &core.WishlistAllocation{
		WishlistItemID: item.ID,
		UserUUID:       userUUID,
		AmountE5:       amountE5,
		SourceType:     "manual",
		CycleDate:      asOf,
	}

	if _, err := e.repos.Wishlist.CreateWishlistAllocation(alloc, tx); err != nil {
		e.logger.Error("Failed to create wishlist allocation", zap.Error(err), zap.String("item_id", item.ID.String()))
		return nil, err
	}

	item.SavedAmountE5 = newSaved
	if isFulfilled {
		item.Status = "fulfilled"
	}

	if err := e.repos.Wishlist.UpdateWishlistItem(item, tx); err != nil {
		e.logger.Error("Failed to update item saved amount", zap.Error(err), zap.String("item_id", item.ID.String()))
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		e.logger.Error("Failed to commit manual allocation transaction", zap.Error(err))
		return nil, err
	}

	e.logger.Info("Successfully applied manual wishlist allocation",
		zap.String("user_uuid", userUUID.String()),
		zap.String("item_id", item.ID.String()),
		zap.Int64("allocated_e5", amountE5),
	)

	w := item.Priority * item.Urgency
	if w <= 0 {
		w = 1
	}

	return &core.ItemAllocationSimulation{
		ItemID:          item.ID,
		ItemTitle:       item.Title,
		AllocatedE5:     amountE5,
		PreviousSavedE5: prevSaved,
		NewSavedE5:      newSaved,
		TargetAmountE5:  item.TargetAmountE5,
		IsFulfilled:     isFulfilled,
		Weight:          w,
	}, nil
}
