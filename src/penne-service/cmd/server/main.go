package main

import (
	"context"
	"net/http"

	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		fx.Provide(
			pkg.NewLogger,
			NewMux,
			NewHTTPServer,
		),
		fx.Invoke(
			func(httpServer *http.Server) {},
		),
	)

	app.Run()

}

func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	return mux
}

func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger) *http.Server {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Starting HTTP server", zap.String("addr", srv.Addr))
			go srv.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Stopping HTTP server")
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
