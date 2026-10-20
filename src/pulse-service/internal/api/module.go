package api

import "go.uber.org/fx"

var Module = fx.Provide(
	NewPulseHandler,
	NewRouter,
)
