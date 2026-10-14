package grpcserver

import (
	"testing"

	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx/fxtest"
)

func TestModule(t *testing.T) {
	app := fxtest.New(
		t,
		pkg.Module,
		Module,
	)
	app.RequireStart()
	app.RequireStop()
}
