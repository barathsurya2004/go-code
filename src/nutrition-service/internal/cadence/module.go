package cadence

import (
	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence/activities"
	"go.uber.org/fx"
)

// ClientModule exports the Cadence client and config without starting a worker (used by gRPC server).
var ClientModule = fx.Module(
	"nutrition-cadence-client",
	fx.Provide(
		NewCadenceConfig,
		NewCadenceServiceClient,
		NewCadenceClient,
	),
)

// WorkerModule exports the Cadence worker, registering workflows and activities and starting pollers.
var WorkerModule = fx.Module(
	"nutrition-cadence-worker",
	fx.Provide(
		NewCadenceConfig,
		NewCadenceServiceClient,
		NewCadenceClient,
		activities.NewNutritionActivities,
	),
	fx.Invoke(StartCadenceWorker),
)

// Module aliases WorkerModule for backward compatibility.
var Module = WorkerModule
