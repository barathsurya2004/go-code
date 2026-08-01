package pkg

import (
	"go.uber.org/fx"

	"github.com/barathsurya2004/go-code/pkg/logger"
)

var Module = fx.Module(
	"pkg",
	fx.Provide(logger.NewLogger),
)
