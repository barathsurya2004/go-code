package main

import (
	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"github.com/barathsurya2004/go-code/penne-service/internal/db"
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
)

func buildApp(opts ...fx.Option) *fx.App {
	baseOpts := []fx.Option{
		pkg.Module,
		db.Module,
		cadence.Module,
		fx.Invoke(cadence.StartWorker, cadence.StartEmailWorker),
	}
	baseOpts = append(baseOpts, opts...)
	return fx.New(baseOpts...)
}

func main() {
	app := buildApp()
	app.Run()
}
