package activities

import (
	"context"
	"database/sql"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/barathsurya2004/go-code/penne-service/internal/wishlist"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WishlistActivities struct {
	repos  core.RepoContainer
	engine *wishlist.WishlistEngine
	logger *zap.Logger
}

func NewWishlistActivities(repos core.RepoContainer, db *sql.DB, logger *zap.Logger) *WishlistActivities {
	engine := wishlist.NewWishlistEngine(repos, db, logger)
	return &WishlistActivities{
		repos:  repos,
		engine: engine,
		logger: logger,
	}
}

func (a *WishlistActivities) CalculateWishlistForecastActivity(ctx context.Context, userUUID uuid.UUID) (*core.WishlistForecastSummary, error) {
	asOf := utils.NowUTC()
	user, err := a.repos.User.GetUserByUUID(userUUID)
	if err != nil {
		a.logger.Error("Failed to get user for forecast activity", zap.Error(err), zap.String("user_uuid", userUUID.String()))
		return nil, err
	}

	budget, expenses, _, _, _, err := a.engine.CalculateCycleSurplus(ctx, userUUID, asOf)
	if err != nil {
		a.logger.Error("Failed to calculate cycle surplus in activity", zap.Error(err), zap.String("user_uuid", userUUID.String()))
		return nil, err
	}

	items, err := a.repos.Wishlist.GetWishlistItemsByUserUUID(userUUID)
	if err != nil {
		a.logger.Error("Failed to fetch wishlist items in activity", zap.Error(err), zap.String("user_uuid", userUUID.String()))
		return nil, err
	}

	forecast := a.engine.CalculateForecasts(items, budget, expenses, asOf, user.SalaryDay)
	return forecast, nil
}

func (a *WishlistActivities) ApplyWishlistSurplusActivity(ctx context.Context, userUUID uuid.UUID) ([]*core.ItemAllocationSimulation, error) {
	asOf := utils.NowUTC()
	results, err := a.engine.ApplySurplusDistribution(ctx, userUUID, asOf)
	if err != nil {
		a.logger.Error("Failed to apply surplus distribution in activity", zap.Error(err), zap.String("user_uuid", userUUID.String()))
		return nil, err
	}
	return results, nil
}
