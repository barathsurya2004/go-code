package grpcserver

import (
	"github.com/barathsurya2004/go-code/nutrition-service/internal/ai"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/db"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/queue"
	"go.uber.org/fx"
)

var Module = fx.Module("nutrition-grpcserver",
	fx.Provide(
		fx.Annotate(
			db.NewNutritionRepository,
			fx.As(new(core.NutritionRepository)),
		),
		fx.Annotate(
			ai.NewVisionAIParser,
			fx.As(new(core.AIParser)),
		),
		fx.Annotate(
			queue.NewFXAsyncImageQueueWorker,
			fx.As(new(core.ImageQueueWorker)),
		),
		NewNutritionGRPCServer,
		StartGRPCServer,
	),
	fx.Invoke(func(s *NutritionGRPCServer) {}),
)
