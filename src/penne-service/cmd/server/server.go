package main

import (
	"context"
	"net/http"

	"github.com/barathsurya2004/go-code/penne-service/cmd/server/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewMux() *mux.Router {
	mux := mux.NewRouter()
	return mux
}

type Application struct {
	transactionHandler *handlers.TransactionServiceHandler
	userHandler        *handlers.UserServiceHandler
}

func NewApplication(
	transactionHandler *handlers.TransactionServiceHandler,
	userHandler *handlers.UserServiceHandler,
) *Application {
	return &Application{
		transactionHandler: transactionHandler,
		userHandler:        userHandler,
	}
}

func RegisterRoutes(mux *mux.Router, log *zap.Logger, app *Application) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Health check endpoint hit")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	//user endpoints
	mux.HandleFunc("/user", app.userHandler.CreateUser).Methods("POST")
	mux.HandleFunc("/user", app.userHandler.GetUserByUUID).Methods("GET")

	//transaction endpoints
	mux.HandleFunc("/transaction", app.transactionHandler.CreateTransaction).Methods("POST")
	mux.HandleFunc("/transaction", app.transactionHandler.GetTransactionByUUID).Methods("GET")
	mux.HandleFunc("/transactions", app.transactionHandler.GetTransactionsByUserUUID).Methods("GET")
	mux.HandleFunc("/transaction", app.transactionHandler.UpdateTransaction).Methods("PUT")
	mux.HandleFunc("/transaction", app.transactionHandler.DeleteTransaction).Methods("DELETE")

}

func NewHTTPServer(lc fx.Lifecycle, mux *mux.Router, log *zap.Logger) *http.Server {
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
