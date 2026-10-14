package main

import (
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		pkg.Module,
		fx.Provide(
			NewConfig,
			NewPenneClient,
			NewNutritionClient,
			NewGatewayHandlerFX,
			NewRouter,
		),
		fx.Invoke(StartServer),
	).Run()
}
