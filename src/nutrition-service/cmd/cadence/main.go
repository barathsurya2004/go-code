package main

import (
	"github.com/barathsurya2004/go-code/nutrition-service/internal/ai"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/db"
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
)

func buildApp(opts ...fx.Option) *fx.App {
	baseOpts := []fx.Option{
		pkg.Module,
		fx.Provide(
			fx.Annotate(db.NewNutritionRepository, fx.As(new(core.NutritionRepository))),
			fx.Annotate(ai.NewVisionAIParser, fx.As(new(core.AIParser))),
		),
		cadence.WorkerModule,
	}
	baseOpts = append(baseOpts, opts...)
	return fx.New(baseOpts...)
}

var appFactory = buildApp

func main() {
	appFactory().Run()
}
