package main

import (
	"net/http"

	"github.com/barathsurya2004/go-code/pkg"
	"github.com/barathsurya2004/go-code/pulse-service/internal/api"
	"github.com/barathsurya2004/go-code/pulse-service/internal/config"
	"github.com/barathsurya2004/go-code/pulse-service/internal/db"
	"go.uber.org/fx"
)

func buildApp(opts ...fx.Option) *fx.App {
	baseOpts := []fx.Option{
		pkg.Module,
		config.Module,
		db.Module,
		api.Module,
		fx.Provide(
			NewHTTPServer,
		),
		fx.Invoke(
			func(s *http.Server) {},
		),
	}
	baseOpts = append(baseOpts, opts...)
	return fx.New(baseOpts...)
}

var appFactory = buildApp

func main() {
	app := appFactory()
	app.Run()
}
