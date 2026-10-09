package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/barathsurya2004/go-code/pkg/grpcclient"
	pennev1 "github.com/barathsurya2004/go-code/pkg/proto/pennev1"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GatewayHandler struct {
	penneClient pennev1.PenneServiceClient
	log         *zap.Logger
}

func NewGatewayHandler(penneClient pennev1.PenneServiceClient, log *zap.Logger) *GatewayHandler {
	return &GatewayHandler{
		penneClient: penneClient,
		log:         log,
	}
}

type ProtocolMultiplexer struct {
	router     *mux.Router
	grpcServer *grpc.Server
}

func (m *ProtocolMultiplexer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") && m.grpcServer != nil {
		m.grpcServer.ServeHTTP(w, r)
		return
	}
	m.router.ServeHTTP(w, r)
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-user-uuid, x-request-id")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AuthAndCorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("x-request-id")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		r.Header.Set("x-request-id", reqID)
		w.Header().Set("x-request-id", reqID)

		userUUID := r.Header.Get("x-user-uuid")
		if userUUID == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				userUUID = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Attach to outgoing gRPC context
		ctx := grpcclient.WithUserMetadata(r.Context(), userUUID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *GatewayHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp, err := h.penneClient.CheckHealth(r.Context(), &pennev1.HealthCheckRequest{})
	statusStr := "UP"
	downstream := "SERVING"
	if err != nil {
		downstream = "UNAVAILABLE"
	} else if resp != nil {
		downstream = resp.GetStatus()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"gateway_status":    statusStr,
		"downstream_status": downstream,
	})
}

func (h *GatewayHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("id")
	resp, err := h.penneClient.GetUser(r.Context(), &pennev1.GetUserRequest{
		UserUuid: userUUID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserUUID   string  `json:"user_uuid"`
		Amount     float64 `json:"amount"`
		Category   string  `json:"category"`
		Remarks    string  `json:"remarks"`
		EnvelopeID string  `json:"envelope_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.penneClient.CreateTransaction(r.Context(), &pennev1.CreateTransactionRequest{
		UserUuid:   body.UserUUID,
		Amount:     body.Amount,
		Category:   body.Category,
		Remarks:    body.Remarks,
		EnvelopeId: body.EnvelopeID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	txUUID := r.URL.Query().Get("id")
	resp, err := h.penneClient.GetTransaction(r.Context(), &pennev1.GetTransactionRequest{
		TransactionUuid: txUUID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) GetTransactionsByUser(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	resp, err := h.penneClient.GetTransactionsByUser(r.Context(), &pennev1.GetTransactionsByUserRequest{
		UserUuid: userUUID,
		Limit:    int32(limit),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID         string  `json:"id"`
		UserUUID   string  `json:"user_uuid"`
		Amount     float64 `json:"amount"`
		Category   string  `json:"category"`
		Remarks    string  `json:"remarks"`
		EnvelopeID string  `json:"envelope_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.penneClient.UpdateTransaction(r.Context(), &pennev1.UpdateTransactionRequest{
		Id:         body.ID,
		UserUuid:   body.UserUUID,
		Amount:     body.Amount,
		Category:   body.Category,
		Remarks:    body.Remarks,
		EnvelopeId: body.EnvelopeID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	txUUID := r.URL.Query().Get("id")
	resp, err := h.penneClient.DeleteTransaction(r.Context(), &pennev1.DeleteTransactionRequest{
		TransactionUuid: txUUID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) GetEnvelope(w http.ResponseWriter, r *http.Request) {
	envID := r.URL.Query().Get("id")
	resp, err := h.penneClient.GetEnvelope(r.Context(), &pennev1.GetEnvelopeRequest{
		EnvelopeId: envID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) GetEnvelopesByUser(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_id")
	resp, err := h.penneClient.GetEnvelopesByUser(r.Context(), &pennev1.GetEnvelopesByUserRequest{
		UserUuid: userUUID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var httpCode int
	switch st.Code() {
	case codes.InvalidArgument:
		httpCode = http.StatusBadRequest
	case codes.NotFound:
		httpCode = http.StatusNotFound
	case codes.Unauthenticated:
		httpCode = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpCode = http.StatusForbidden
	case codes.AlreadyExists:
		httpCode = http.StatusConflict
	default:
		httpCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": st.Message(),
	})
}

func NewRouter(handler *GatewayHandler, cfg *Config) *mux.Router {
	r := mux.NewRouter()
	r.Use(AuthAndCorrelationMiddleware)

	// Health check
	r.HandleFunc("/health", handler.HealthCheck).Methods(http.MethodGet)

	// Direct gRPC routes under /api/v1 for service-to-service / external REST-to-gRPC calls
	prefix := "/api/v1"
	r.HandleFunc(prefix+"/user", handler.GetUser).Methods(http.MethodGet)
	r.HandleFunc(prefix+"/transaction", handler.CreateTransaction).Methods(http.MethodPost)
	r.HandleFunc(prefix+"/transaction", handler.GetTransaction).Methods(http.MethodGet)
	r.HandleFunc(prefix+"/transaction", handler.UpdateTransaction).Methods(http.MethodPut)
	r.HandleFunc(prefix+"/transaction", handler.DeleteTransaction).Methods(http.MethodDelete)
	r.HandleFunc(prefix+"/transactions", handler.GetTransactionsByUser).Methods(http.MethodGet)
	r.HandleFunc(prefix+"/envelope", handler.GetEnvelope).Methods(http.MethodGet)
	r.HandleFunc(prefix+"/envelopes", handler.GetEnvelopesByUser).Methods(http.MethodGet)

	// Fallback reverse proxy for legacy/unmigrated endpoints (/auth/*, /envelope-groups, /allocations/*, etc.)
	if cfg != nil && cfg.PenneHTTPTarget != "" {
		targetURL, err := url.Parse(cfg.PenneHTTPTarget)
		if err == nil && targetURL.Host != "" {
			proxy := httputil.NewSingleHostReverseProxy(targetURL)
			r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				proxy.ServeHTTP(w, req)
			})
		}
	}

	return r
}

func NewPenneClient(cfg *Config) (pennev1.PenneServiceClient, *grpcclient.ClientPool, error) {
	pool, err := grpcclient.NewClientPool(grpcclient.PoolConfig{
		Target: cfg.PenneServiceTarget,
		Size:   4,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to penne-service pool: %w", err)
	}
	return pennev1.NewPenneServiceClient(pool.Get()), pool, nil
}

func StartServer(cfg *Config, router *mux.Router, lc fx.Lifecycle, log *zap.Logger) (*http.Server, error) {
	multiplexer := &ProtocolMultiplexer{
		router:     router,
		grpcServer: grpc.NewServer(),
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      CORSMiddleware(multiplexer),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	if lc != nil {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				lis, err := net.Listen("tcp", server.Addr)
				if err != nil {
					return err
				}
				log.Info("Gateway started on single ingress port", zap.String("port", cfg.Port))
				go func() {
					if err := server.Serve(lis); err != nil && err != http.ErrServerClosed {
						log.Error("Gateway server terminated unexpectedly", zap.Error(err))
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Info("Gracefully stopping Gateway server")
				return server.Shutdown(ctx)
			},
		})
	}

	return server, nil
}
