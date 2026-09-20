package activities

import (
	"context"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserActivities struct {
	Repos  core.RepoContainer
	logger *zap.Logger
}

func NewUserActivities(repos core.RepoContainer, logger *zap.Logger) *UserActivities {
	return &UserActivities{
		Repos:  repos,
		logger: logger,
	}
}

func (a *UserActivities) CreateUserActivity(ctx context.Context, user core.User) (*uuid.UUID, error) {
	userUUID, err := a.Repos.User.CreateUser(&user, nil)
	if err != nil {
		a.logger.Error("Failed to create user in activity", zap.Error(err))
		return nil, err
	}
	a.logger.Info("User created successfully in activity", zap.String("user_uuid", userUUID.String()))
	return &userUUID, nil
}

func (a *UserActivities) CreateSystemEnvelopeGroupActivity(ctx context.Context, userUUID uuid.UUID) (*uuid.UUID, error) {
	now := utils.NowUTC()
	envGroup := core.EnvelopeGroup{
		ID:        uuid.New(),
		UserUUID:  userUUID,
		Name:      "Unallocated Budget",
		IsSystem:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	envGroupUUID, err := a.Repos.EnvelopeGroup.CreateEnvelopeGroup(&envGroup, nil)
	if err != nil {
		a.logger.Error("Failed to create system envelope group in activity", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return nil, err
	}
	a.logger.Info("System envelope group created", zap.String("group_uuid", envGroupUUID.String()))
	return &envGroupUUID, nil
}

func (a *UserActivities) CreateSystemEnvelopeActivity(ctx context.Context, input core.CreateSystemEnvelopeActivityInput) (*uuid.UUID, error) {
	now := utils.NowUTC()
	env := core.Envelope{
		UserUUID:        input.UserUUID,
		EnvelopeGroupID: input.EnvelopeGroupID,
		Name:            "Unallocated",
		TargetAmountE5:  0,
		Cadence:         "monthly",
		CountryISO:      "IN",
		IsSystem:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	envUUID, err := a.Repos.Envelope.CreateEnvelope(&env, nil)
	if err != nil {
		a.logger.Error("Failed to create system envelope in activity", zap.String("user_uuid", input.UserUUID.String()), zap.Error(err))
		return nil, err
	}
	a.logger.Info("System envelope created", zap.String("env_uuid", envUUID.String()))
	return &envUUID, nil
}

func (a *UserActivities) CreateDefaultAllocationActivity(ctx context.Context, envelopeID uuid.UUID) (*uuid.UUID, error) {
	now := utils.NowUTC()
	endDate := now.AddDate(100, 0, 0)
	alloc := core.Allocation{
		EnvelopeID:        envelopeID,
		AllocatedAmountE5: 0,
		StartDate:         &now,
		EndDate:           &endDate,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	allocUUID, err := a.Repos.Allocation.CreateAllocation(&alloc, nil)
	if err != nil {
		a.logger.Error("Failed to create default allocation in activity", zap.String("envelope_id", envelopeID.String()), zap.Error(err))
		return nil, err
	}
	a.logger.Info("Default allocation created", zap.String("allocation_uuid", allocUUID.String()))
	return &allocUUID, nil
}

func (a *UserActivities) CreateUserTokenActivity(ctx context.Context, userUUID uuid.UUID) (*uuid.UUID, error) {
	now := utils.NowUTC()
	token := core.Token{
		UserUUID:  userUUID,
		Prefix:    core.AuthToken,
		Name:      core.DefaultName,
		Scope:     []string{"all"},
		ExpiresAt: nil,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tokenUUID, err := a.Repos.Token.CreateToken(&token, nil)
	if err != nil {
		a.logger.Error("Failed to create user token in activity", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return nil, err
	}
	a.logger.Info("User token created", zap.String("token_uuid", tokenUUID.String()))
	return &tokenUUID, nil
}
