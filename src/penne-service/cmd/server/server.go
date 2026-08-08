package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/cmd/server/handlers"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
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
	budgetingHandler   *handlers.BudgetingServiceHandler
	tokenRepo          core.TokenRepository
}

func NewApplication(
	transactionHandler *handlers.TransactionServiceHandler,
	userHandler *handlers.UserServiceHandler,
	budgetingHandler *handlers.BudgetingServiceHandler,
	tokenRepo core.TokenRepository,
) *Application {
	return &Application{
		transactionHandler: transactionHandler,
		userHandler:        userHandler,
		budgetingHandler:   budgetingHandler,
		tokenRepo:          tokenRepo,
	}
}

func RegisterRoutes(mux *mux.Router, log *zap.Logger, app *Application) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Health check endpoint hit(deployment checkup)")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK(deployment check)"))
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

	// envelope group endpoints
	mux.HandleFunc("/envelope-group", app.budgetingHandler.CreateEnvelopeGroup).Methods("POST")
	mux.HandleFunc("/envelope-group", app.budgetingHandler.GetEnvelopeGroupByID).Methods("GET")
	mux.HandleFunc("/envelope-groups", app.budgetingHandler.GetEnvelopeGroupsByUserUUID).Methods("GET")
	mux.HandleFunc("/envelope-group", app.budgetingHandler.UpdateEnvelopeGroup).Methods("PUT")
	mux.HandleFunc("/envelope-group", app.budgetingHandler.DeleteEnvelopeGroup).Methods("DELETE")

	// envelope endpoints
	mux.HandleFunc("/envelope", app.budgetingHandler.CreateEnvelope).Methods("POST")
	mux.HandleFunc("/envelope", app.budgetingHandler.GetEnvelopeByID).Methods("GET")
	mux.HandleFunc("/envelopes", app.budgetingHandler.GetEnvelopesByUserUUID).Methods("GET")
	mux.HandleFunc("/envelope", app.budgetingHandler.UpdateEnvelope).Methods("PUT")
	mux.HandleFunc("/envelope", app.budgetingHandler.DeleteEnvelope).Methods("DELETE")

	// allocation endpoints
	mux.HandleFunc("/allocation", app.budgetingHandler.CreateAllocation).Methods("POST")
	mux.HandleFunc("/allocation", app.budgetingHandler.GetAllocationByID).Methods("GET")
	mux.HandleFunc("/allocations", app.budgetingHandler.GetAllocationsByEnvelopeID).Methods("GET")
	mux.HandleFunc("/allocations/active", app.budgetingHandler.GetActiveAllocationsByUserUUID).Methods("GET")
	mux.HandleFunc("/allocation", app.budgetingHandler.UpdateAllocation).Methods("PUT")
	mux.HandleFunc("/allocation", app.budgetingHandler.DeleteAllocation).Methods("DELETE")

	mux.Use(AuthMiddleware(app.tokenRepo))
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

func AuthMiddleware(Tokenrepo core.TokenRepository) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || (r.URL.Path == "/user" && r.Method == http.MethodPost) {
				next.ServeHTTP(w, r)
				return
			}

			authToken := r.Header.Get("Authorization")
			if authToken == "" {
				http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authToken, "Bearer ")

			userToken, err := Tokenrepo.GetToken(token)

			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if userToken.ExpiresAt != nil && userToken.ExpiresAt.Before(time.Now()) {
				http.Error(w, "Token expired", http.StatusUnauthorized)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), "user_uuid", userToken.UserUUID))

			next.ServeHTTP(w, r)

		})
	}
}
