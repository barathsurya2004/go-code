package main

import (
	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/grpcserver"
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

func buildApp(opts ...fx.Option) *fx.App {
	baseOpts := []fx.Option{
		pkg.Module,
		cadence.Module,
		grpcserver.Module,
		fx.Invoke(func(s *grpc.Server) {}),
	}
	baseOpts = append(baseOpts, opts...)
	return fx.New(baseOpts...)
}

func main() {
	app := buildApp()
	app.Run()
}
