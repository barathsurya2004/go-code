package main

import (
	"net/http"

	"github.com/barathsurya2004/go-code/penne-service/cmd/server/handlers"
	"github.com/barathsurya2004/go-code/penne-service/internal/db"
	mcpserver "github.com/barathsurya2004/go-code/penne-service/internal/server"
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
)

func buildApp(opts ...fx.Option) *fx.App {
	baseOpts := []fx.Option{
		pkg.Module,
		db.Module,
		handlers.Module,
		mcpserver.Module,
		fx.Provide(
			NewApplication,
			NewMux,
			NewHTTPServer,
		),
		fx.Invoke(
			RegisterRoutes,
			func(s *http.Server) {},
		),
	}
	baseOpts = append(baseOpts, opts...)
	return fx.New(baseOpts...)
}

func main() {
	app := buildApp()
	app.Run()
}
