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
		pkg.Module,
		db.Module,
		handlers.Module,
		fx.Provide(
			NewApplication,
			NewMux,
			NewHTTPServer,
		),
		fx.Invoke(
			RegisterRoutes,
			func(s *http.Server) {},
		),
	)

	app.Run()

}
