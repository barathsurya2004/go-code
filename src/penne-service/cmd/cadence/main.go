package main

import (
	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"github.com/barathsurya2004/go-code/penne-service/internal/db"
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
)

func main() {

	app := fx.New(
		pkg.Module,
		db.Module,
		cadence.Module,
		fx.Invoke(cadence.StartWorker),
	)

	app.Run()

}
