package db

import (
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"db",
	fx.Provide(CreateConfig),
	fx.Provide(NewDb),
	fx.Provide(NewPgTransactionRowsRepo),
	fx.Provide(NewPgUserRepo),
	fx.Provide(NewTokenRepo),
	fx.Provide(NewEnvelopeGroupRepository),
	fx.Provide(NewPgEnvelopeRepo),
	fx.Provide(NewPgAllocationRepo),
	fx.Provide(NewPgShortcutIntentRepo),
	fx.Provide(NewRepoContainer),
)

func NewRepoContainer(
	Transaction core.TransactionRepository,
	User core.UserRepository,
	Token core.TokenRepository,
	EnvelopeGroup core.EnvelopeGroupRepository,
	Envelope core.EnvelopeRepository,
	Allocation core.AllocationRepository,
	ShortcutIntent core.ShortcutIntentRepository,
) core.RepoContainer {
	return core.RepoContainer{
		Transaction:    Transaction,
		User:           User,
		Token:          Token,
		EnvelopeGroup:  EnvelopeGroup,
		Envelope:       Envelope,
		Allocation:     Allocation,
		ShortcutIntent: ShortcutIntent,
	}
}
