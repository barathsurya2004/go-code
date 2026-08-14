package db

import "go.uber.org/fx"

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
)
