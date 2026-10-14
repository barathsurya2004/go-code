package main

import (
	"context"
	"testing"

	"go.uber.org/fx"
)

func TestBuildApp(t *testing.T) {
	app := buildApp(
		fx.NopLogger,
	)
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}

func TestMainFunction(t *testing.T) {
	orig := appFactory
	defer func() { appFactory = orig }()

	appFactory = func(opts ...fx.Option) *fx.App {
		return fx.New(
			fx.NopLogger,
			fx.Invoke(func(lc fx.Lifecycle, s fx.Shutdowner) {
				lc.Append(fx.Hook{
					OnStart: func(context.Context) error {
						go func() {
							_ = s.Shutdown()
						}()
						return nil
					},
				})
			}),
		)
	}

	main()
}
