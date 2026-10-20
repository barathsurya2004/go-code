package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/pulse-service/internal/config"
	"github.com/barathsurya2004/go-code/pulse-service/internal/db"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestServerLifecycle(t *testing.T) {
	log := zap.NewNop()
	cfg := &config.Config{
		Port: "18082",
	}
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := fxtest.New(
		t,
		fx.Provide(
			func() *config.Config { return cfg },
			func() *zap.Logger { return log },
			func() *mux.Router { return r },
			NewHTTPServer,
		),
		fx.Invoke(func(s *http.Server) {}),
	)

	app.RequireStart()

	// Verify server responds
	resp, err := http.Get("http://localhost:18082/health")
	assert.NoError(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		_ = resp.Body.Close()
	}

	app.RequireStop()
}

func TestBuildApp(t *testing.T) {
	log := zap.NewNop()
	repo := db.NewMemoryPulseRepository(log)

	// Build app with test overrides to avoid port collisions and real postgres
	app := buildApp(
		fx.Replace(repo),
		fx.Decorate(func(cfg *config.Config) *config.Config {
			return &config.Config{Port: "18083"}
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	assert.NoError(t, app.Start(ctx))
	assert.NoError(t, app.Stop(ctx))
}

func TestServerListenError(t *testing.T) {
	log := zap.NewNop()
	cfg := &config.Config{
		Port: "-1",
	}
	r := mux.NewRouter()

	app := fxtest.New(
		t,
		fx.Provide(
			func() *config.Config { return cfg },
			func() *zap.Logger { return log },
			func() *mux.Router { return r },
			NewHTTPServer,
		),
		fx.Invoke(func(s *http.Server) {}),
	)

	err := app.Start(context.Background())
	assert.Error(t, err)
}

func TestMainFunction(t *testing.T) {
	oldFactory := appFactory
	defer func() { appFactory = oldFactory }()

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

