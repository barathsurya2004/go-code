package main

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/barathsurya2004/go-code/pulse-service/internal/config"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHTTPServer(lc fx.Lifecycle, r *mux.Router, cfg *config.Config, log *zap.Logger) *http.Server {
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				log.Error("Failed to listen for pulse-service HTTP server", zap.Error(err), zap.String("addr", srv.Addr))
				return err
			}

			log.Info("Starting Pulse service HTTP server", zap.String("addr", srv.Addr))
			go func() {
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					log.Error("Pulse HTTP server stopped with error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Stopping Pulse service HTTP server gracefully")
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
