package cadence

import (
	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence/activities"
	"go.uber.org/fx"
)

// Module exports the Cadence dependency injection module for Uber FX.
var Module = fx.Module(
	"nutrition-cadence",
	fx.Provide(
		NewCadenceConfig,
		NewCadenceServiceClient,
		NewCadenceClient,
		activities.NewNutritionActivities,
	),
	fx.Invoke(StartCadenceWorker),
)
