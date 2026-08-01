package main

import (
	"net/http"

	"github.com/barathsurya2004/go-code/penne-service/cmd/server/handlers"
	"github.com/barathsurya2004/go-code/penne-service/internal/db"
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			db.CreateConfig,
			db.NewDb,
			db.NewPgTransactionRowsRepo,
			handlers.NewTransactionServiceHandler,
			NewApplication,
			NewMux,
			NewHTTPServer,
			pkg.NewLogger,
		),
		fx.Invoke(
			func(httpServer *http.Server) {},
			RegisterRoutes,
		),
	)

	app.Run()

}
