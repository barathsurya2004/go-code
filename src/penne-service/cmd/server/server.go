package main

import (
	"context"
	"net/http"

	"github.com/barathsurya2004/go-code/penne-service/cmd/server/handlers"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	mcpserver "github.com/barathsurya2004/go-code/penne-service/internal/server"
	"github.com/gorilla/mux"
	mcp "github.com/mark3labs/mcp-go/server"
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
	mcpServer          *mcp.SSEServer
	tokenRepo          core.TokenRepository
}

func NewApplication(
	transactionHandler *handlers.TransactionServiceHandler,
	userHandler *handlers.UserServiceHandler,
	mcpServer *mcp.SSEServer,
	tokenRepo core.TokenRepository,
) *Application {
	return &Application{
		transactionHandler: transactionHandler,
		userHandler:        userHandler,
		mcpServer:          mcpServer,
		tokenRepo:          tokenRepo,
	}
}

func RegisterRoutes(mux *mux.Router, log *zap.Logger, app *Application) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Health check endpoint hit")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// user endpoints
	mux.HandleFunc("/user", app.userHandler.CreateUser).Methods("POST")
	mux.HandleFunc("/user", app.userHandler.GetUserByUUID).Methods("GET")

	// transaction endpoints
	mux.HandleFunc("/transaction", app.transactionHandler.CreateTransaction).Methods("POST")
	mux.HandleFunc("/transaction", app.transactionHandler.GetTransactionByUUID).Methods("GET")
	mux.HandleFunc("/transactions", app.transactionHandler.GetTransactionsByUserUUID).Methods("GET")
	mux.HandleFunc("/transaction", app.transactionHandler.UpdateTransaction).Methods("PUT")
	mux.HandleFunc("/transaction", app.transactionHandler.DeleteTransaction).Methods("DELETE")

	// mcp endpoints protected by auth middleware
	if app.mcpServer != nil {
		mcpAuth := mcpserver.MCPAuthMiddleWare(app.tokenRepo)
		mux.Handle("/sse", mcpAuth(app.mcpServer))
		mux.Handle("/message", mcpAuth(app.mcpServer))
	}
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
