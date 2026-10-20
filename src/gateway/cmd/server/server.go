package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/barathsurya2004/go-code/pkg/grpcclient"
	nutritionv1 "github.com/barathsurya2004/go-code/pkg/proto/nutritionv1"
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
	penneClient     pennev1.PenneServiceClient
	nutritionClient nutritionv1.NutritionServiceClient
	log             *zap.Logger
}

type GatewayHandlerParams struct {
	fx.In
	PenneClient     pennev1.PenneServiceClient
	Log             *zap.Logger
	NutritionClient nutritionv1.NutritionServiceClient `optional:"true"`
}

func NewGatewayHandlerFX(p GatewayHandlerParams) *GatewayHandler {
	return NewGatewayHandlerWithNutrition(p.PenneClient, p.NutritionClient, p.Log)
}

func NewGatewayHandler(penneClient pennev1.PenneServiceClient, log *zap.Logger) *GatewayHandler {
	return NewGatewayHandlerWithNutrition(penneClient, nil, log)
}

func NewGatewayHandlerWithNutrition(penneClient pennev1.PenneServiceClient, nutritionClient nutritionv1.NutritionServiceClient, log *zap.Logger) *GatewayHandler {
	return &GatewayHandler{
		penneClient:     penneClient,
		nutritionClient: nutritionClient,
		log:             log,
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
		origin := r.Header.Get("Origin")
		allowOrigin := "*"
		if origin != "" {
			allowOrigin = origin
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-user-uuid, x-request-id, ngrok-skip-browser-warning, X-Requested-With, Accept, Origin, Cache-Control, Pragma")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			reqHeaders := r.Header.Get("Access-Control-Request-Headers")
			if reqHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			}
			w.Header().Set("Access-Control-Max-Age", "86400")
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

	// Nutrition routes
	r.HandleFunc(prefix+"/nutrition/upload", handler.UploadMealImage).Methods(http.MethodPost)
	r.HandleFunc(prefix+"/nutrition/jobs/{id}", handler.GetNutritionJob).Methods(http.MethodGet)
	r.HandleFunc(prefix+"/nutrition/daily", handler.GetDailyNutrition).Methods(http.MethodGet)
	r.HandleFunc(prefix+"/nutrition/goals", handler.GetNutritionGoals).Methods(http.MethodGet)
	r.HandleFunc(prefix+"/nutrition/goals", handler.SetNutritionGoals).Methods(http.MethodPost)
	r.HandleFunc(prefix+"/nutrition/meals", handler.AddManualNutritionMeal).Methods(http.MethodPost)
	r.HandleFunc(prefix+"/nutrition/meals/{id}", handler.DeleteNutritionMeal).Methods(http.MethodDelete)
	r.HandleFunc(prefix+"/nutrition/items/{id}", handler.UpdateNutritionItem).Methods(http.MethodPut)
	r.HandleFunc(prefix+"/nutrition/items/{id}", handler.DeleteNutritionItem).Methods(http.MethodDelete)

	// Pulse service routes forwarding
	if cfg != nil && cfg.PulseHTTPTarget != "" {
		targetURL, err := url.Parse(cfg.PulseHTTPTarget)
		if err == nil && targetURL.Host != "" {
			pulseProxy := httputil.NewSingleHostReverseProxy(targetURL)
			pulseProxy.ModifyResponse = func(resp *http.Response) error {
				resp.Header.Del("Access-Control-Allow-Origin")
				resp.Header.Del("Access-Control-Allow-Methods")
				resp.Header.Del("Access-Control-Allow-Headers")
				resp.Header.Del("Access-Control-Allow-Credentials")
				return nil
			}
			r.PathPrefix(prefix + "/pulse").Handler(pulseProxy)
		}
	}

	// Fallback reverse proxy for legacy/unmigrated endpoints (/auth/*, /envelope-groups, /allocations/*, etc.)
	if cfg != nil && cfg.PenneHTTPTarget != "" {
		targetURL, err := url.Parse(cfg.PenneHTTPTarget)
		if err == nil && targetURL.Host != "" {
			proxy := httputil.NewSingleHostReverseProxy(targetURL)
			proxy.ModifyResponse = func(resp *http.Response) error {
				resp.Header.Del("Access-Control-Allow-Origin")
				resp.Header.Del("Access-Control-Allow-Methods")
				resp.Header.Del("Access-Control-Allow-Headers")
				resp.Header.Del("Access-Control-Allow-Credentials")
				return nil
			}
			r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				proxy.ServeHTTP(w, req)
			})
		}
	}

	return r
}

func (h *GatewayHandler) UploadMealImage(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}

	var imageData []byte
	var mimeType string
	var mealType string
	var notes string

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}
		mealType = r.FormValue("meal_type")
		notes = r.FormValue("notes")
		file, header, err := r.FormFile("image")
		if err != nil {
			file, header, err = r.FormFile("file")
		}
		if err != nil {
			http.Error(w, "image file missing in form field 'image' or 'file'", http.StatusBadRequest)
			return
		}
		defer file.Close()
		mimeType = header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "failed to read image file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		imageData = data
	} else if strings.HasPrefix(contentType, "image/") {
		mimeType = contentType
		data, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "failed to read image body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		imageData = data
	} else {
		http.Error(w, "invalid content-type, expected multipart/form-data or image/*", http.StatusBadRequest)
		return
	}

	if len(imageData) == 0 {
		http.Error(w, "image data cannot be empty", http.StatusBadRequest)
		return
	}

	resp, err := h.nutritionClient.ProcessImage(r.Context(), &nutritionv1.ProcessImageRequest{
		ImageData: imageData,
		MimeType:  mimeType,
		MealType:  mealType,
		Notes:     notes,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (h *GatewayHandler) GetNutritionJob(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	jobID := mux.Vars(r)["id"]
	resp, err := h.nutritionClient.GetJobStatus(r.Context(), &nutritionv1.GetJobStatusRequest{
		JobId: jobID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) GetDailyNutrition(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	date := r.URL.Query().Get("date")
	resp, err := h.nutritionClient.GetDailyLog(r.Context(), &nutritionv1.GetDailyLogRequest{
		Date: date,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) GetNutritionGoals(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	resp, err := h.nutritionClient.GetGoals(r.Context(), &nutritionv1.GetGoalsRequest{})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) SetNutritionGoals(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		CaloriesTarget float64 `json:"calories_target"`
		ProteinGrams   float64 `json:"protein_grams"`
		CarbsGrams     float64 `json:"carbs_grams"`
		FatGrams       float64 `json:"fat_grams"`
		FiberGrams     float64 `json:"fiber_grams"`
		SodiumMg       float64 `json:"sodium_mg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.nutritionClient.SetGoals(r.Context(), &nutritionv1.SetGoalsRequest{
		Goals: &nutritionv1.NutritionGoals{
			CaloriesTarget: body.CaloriesTarget,
			ProteinGrams:   body.ProteinGrams,
			CarbsGrams:     body.CarbsGrams,
			FatGrams:       body.FatGrams,
			FiberGrams:     body.FiberGrams,
			SodiumMg:       body.SodiumMg,
		},
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) AddManualNutritionMeal(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		MealType string `json:"meal_type"`
		LoggedAt string `json:"logged_at"`
		Notes    string `json:"notes"`
		Items    []struct {
			Name                 string   `json:"name"`
			EstimatedWeightGrams float64  `json:"estimated_weight_grams"`
			Calories             float64  `json:"calories"`
			ProteinGrams         float64  `json:"protein_grams"`
			CarbsGrams           float64  `json:"carbs_grams"`
			FatGrams             float64  `json:"fat_grams"`
			FiberGrams           float64  `json:"fiber_grams"`
			SodiumMg             float64  `json:"sodium_mg"`
			Ingredients          []string `json:"ingredients"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var protoItems []*nutritionv1.MealItem
	for _, it := range body.Items {
		protoItems = append(protoItems, &nutritionv1.MealItem{
			Name:                 it.Name,
			EstimatedWeightGrams: it.EstimatedWeightGrams,
			Calories:             it.Calories,
			ProteinGrams:         it.ProteinGrams,
			CarbsGrams:           it.CarbsGrams,
			FatGrams:             it.FatGrams,
			FiberGrams:           it.FiberGrams,
			SodiumMg:             it.SodiumMg,
			Ingredients:          it.Ingredients,
		})
	}

	resp, err := h.nutritionClient.AddManualMeal(r.Context(), &nutritionv1.AddManualMealRequest{
		MealType: body.MealType,
		LoggedAt: body.LoggedAt,
		Notes:    body.Notes,
		Items:    protoItems,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *GatewayHandler) DeleteNutritionMeal(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	mealID := mux.Vars(r)["id"]
	resp, err := h.nutritionClient.DeleteMealLog(r.Context(), &nutritionv1.DeleteMealLogRequest{
		MealLogId: mealID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) UpdateNutritionItem(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	itemID := mux.Vars(r)["id"]
	var body struct {
		Name                 string  `json:"name"`
		EstimatedWeightGrams float64 `json:"estimated_weight_grams"`
		Calories             float64 `json:"calories"`
		ProteinGrams         float64 `json:"protein_grams"`
		CarbsGrams           float64 `json:"carbs_grams"`
		FatGrams             float64 `json:"fat_grams"`
		FiberGrams           float64 `json:"fiber_grams"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.nutritionClient.UpdateMealItem(r.Context(), &nutritionv1.UpdateMealItemRequest{
		ItemId:               itemID,
		Name:                 body.Name,
		EstimatedWeightGrams: body.EstimatedWeightGrams,
		Calories:             body.Calories,
		ProteinGrams:         body.ProteinGrams,
		CarbsGrams:           body.CarbsGrams,
		FatGrams:             body.FatGrams,
		FiberGrams:           body.FiberGrams,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *GatewayHandler) DeleteNutritionItem(w http.ResponseWriter, r *http.Request) {
	if h.nutritionClient == nil {
		http.Error(w, "nutrition service is not available", http.StatusServiceUnavailable)
		return
	}
	itemID := mux.Vars(r)["id"]
	resp, err := h.nutritionClient.DeleteMealItem(r.Context(), &nutritionv1.DeleteMealItemRequest{
		ItemId: itemID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func NewNutritionClient(cfg *Config) (nutritionv1.NutritionServiceClient, error) {
	target := cfg.NutritionServiceTarget
	if target == "" {
		target = "127.0.0.1:50052"
	}
	pool, err := grpcclient.NewClientPool(grpcclient.PoolConfig{
		Target: target,
		Size:   4,
		DialOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(32 * 1024 * 1024),
				grpc.MaxCallSendMsgSize(32 * 1024 * 1024),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nutrition-service pool: %w", err)
	}
	return nutritionv1.NewNutritionServiceClient(pool.Get()), nil
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
