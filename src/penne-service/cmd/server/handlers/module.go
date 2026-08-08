package handlers

import "go.uber.org/fx"

var Module = fx.Module(
	"handlers",
	fx.Provide(NewTransactionServiceHandler),
	fx.Provide(NewUserServiceHandler),
	fx.Provide(NewBudgetingServiceHandler),
)
