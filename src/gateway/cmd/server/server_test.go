package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	nutritionv1 "github.com/barathsurya2004/go-code/pkg/proto/nutritionv1"
	pennev1 "github.com/barathsurya2004/go-code/pkg/proto/pennev1"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockPenneServiceClient struct {
	checkHealthFn           func(ctx context.Context, in *pennev1.HealthCheckRequest, opts ...grpc.CallOption) (*pennev1.HealthCheckResponse, error)
	getUserFn               func(ctx context.Context, in *pennev1.GetUserRequest, opts ...grpc.CallOption) (*pennev1.UserResponse, error)
	createTransactionFn     func(ctx context.Context, in *pennev1.CreateTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error)
	getTransactionFn        func(ctx context.Context, in *pennev1.GetTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error)
	getTransactionsByUserFn func(ctx context.Context, in *pennev1.GetTransactionsByUserRequest, opts ...grpc.CallOption) (*pennev1.TransactionListResponse, error)
	updateTransactionFn     func(ctx context.Context, in *pennev1.UpdateTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error)
	deleteTransactionFn     func(ctx context.Context, in *pennev1.DeleteTransactionRequest, opts ...grpc.CallOption) (*pennev1.DeleteResponse, error)
	getEnvelopeFn           func(ctx context.Context, in *pennev1.GetEnvelopeRequest, opts ...grpc.CallOption) (*pennev1.EnvelopeItem, error)
	getEnvelopesByUserFn    func(ctx context.Context, in *pennev1.GetEnvelopesByUserRequest, opts ...grpc.CallOption) (*pennev1.EnvelopeListResponse, error)
}

func (m *mockPenneServiceClient) CheckHealth(ctx context.Context, in *pennev1.HealthCheckRequest, opts ...grpc.CallOption) (*pennev1.HealthCheckResponse, error) {
	if m.checkHealthFn != nil {
		return m.checkHealthFn(ctx, in, opts...)
	}
	return &pennev1.HealthCheckResponse{Status: "SERVING"}, nil
}

func (m *mockPenneServiceClient) GetUser(ctx context.Context, in *pennev1.GetUserRequest, opts ...grpc.CallOption) (*pennev1.UserResponse, error) {
	if m.getUserFn != nil {
		return m.getUserFn(ctx, in, opts...)
	}
	return &pennev1.UserResponse{Uuid: in.GetUserUuid(), Name: "Alice"}, nil
}

func (m *mockPenneServiceClient) CreateTransaction(ctx context.Context, in *pennev1.CreateTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error) {
	if m.createTransactionFn != nil {
		return m.createTransactionFn(ctx, in, opts...)
	}
	return &pennev1.TransactionItem{Id: "tx-1", Amount: in.GetAmount()}, nil
}

func (m *mockPenneServiceClient) GetTransaction(ctx context.Context, in *pennev1.GetTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error) {
	if m.getTransactionFn != nil {
		return m.getTransactionFn(ctx, in, opts...)
	}
	return &pennev1.TransactionItem{Id: in.GetTransactionUuid()}, nil
}

func (m *mockPenneServiceClient) GetTransactionsByUser(ctx context.Context, in *pennev1.GetTransactionsByUserRequest, opts ...grpc.CallOption) (*pennev1.TransactionListResponse, error) {
	if m.getTransactionsByUserFn != nil {
		return m.getTransactionsByUserFn(ctx, in, opts...)
	}
	return &pennev1.TransactionListResponse{Transactions: []*pennev1.TransactionItem{{Id: "tx-1"}}, TotalCount: 1}, nil
}

func (m *mockPenneServiceClient) UpdateTransaction(ctx context.Context, in *pennev1.UpdateTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error) {
	if m.updateTransactionFn != nil {
		return m.updateTransactionFn(ctx, in, opts...)
	}
	return &pennev1.TransactionItem{Id: in.GetId()}, nil
}

func (m *mockPenneServiceClient) DeleteTransaction(ctx context.Context, in *pennev1.DeleteTransactionRequest, opts ...grpc.CallOption) (*pennev1.DeleteResponse, error) {
	if m.deleteTransactionFn != nil {
		return m.deleteTransactionFn(ctx, in, opts...)
	}
	return &pennev1.DeleteResponse{Success: true}, nil
}

func (m *mockPenneServiceClient) GetEnvelope(ctx context.Context, in *pennev1.GetEnvelopeRequest, opts ...grpc.CallOption) (*pennev1.EnvelopeItem, error) {
	if m.getEnvelopeFn != nil {
		return m.getEnvelopeFn(ctx, in, opts...)
	}
	return &pennev1.EnvelopeItem{Id: in.GetEnvelopeId()}, nil
}

func (m *mockPenneServiceClient) GetEnvelopesByUser(ctx context.Context, in *pennev1.GetEnvelopesByUserRequest, opts ...grpc.CallOption) (*pennev1.EnvelopeListResponse, error) {
	if m.getEnvelopesByUserFn != nil {
		return m.getEnvelopesByUserFn(ctx, in, opts...)
	}
	return &pennev1.EnvelopeListResponse{Envelopes: []*pennev1.EnvelopeItem{{Id: "env-1"}}}, nil
}

func TestNewConfig(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("PENNE_SERVICE_TARGET")
	cfg := NewConfig()
	if cfg.Port != "8080" {
		t.Errorf("expected 8080, got %s", cfg.Port)
	}
	if cfg.PenneServiceTarget != "127.0.0.1:50051" {
		t.Errorf("expected default target, got %s", cfg.PenneServiceTarget)
	}

	os.Setenv("PORT", "9090")
	os.Setenv("PENNE_SERVICE_TARGET", "backend:50051")
	cfg2 := NewConfig()
	if cfg2.Port != "9090" || cfg2.PenneServiceTarget != "backend:50051" {
		t.Errorf("expected custom env values, got %+v", cfg2)
	}
	os.Unsetenv("PORT")
	os.Unsetenv("PENNE_SERVICE_TARGET")
}

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Preflight OPTIONS with custom requested headers & Origin
	reqOpt := httptest.NewRequest(http.MethodOptions, "/test", nil)
	reqOpt.Header.Set("Origin", "http://localhost:5173")
	reqOpt.Header.Set("Access-Control-Request-Headers", "authorization, ngrok-skip-browser-warning, cache-control")
	recOpt := httptest.NewRecorder()
	handler.ServeHTTP(recOpt, reqOpt)
	if recOpt.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", recOpt.Code)
	}
	if recOpt.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("expected reflected origin, got %s", recOpt.Header().Get("Access-Control-Allow-Origin"))
	}
	if recOpt.Header().Get("Access-Control-Allow-Headers") != "authorization, ngrok-skip-browser-warning, cache-control" {
		t.Errorf("expected reflected headers")
	}

	// Normal request with Origin
	reqGet := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqGet.Header.Set("Origin", "http://localhost:5173")
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recGet.Code)
	}
	if recGet.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("expected http://localhost:5173 CORS header")
	}
}

func TestAuthAndCorrelationMiddleware(t *testing.T) {
	var capturedUser string
	var capturedReqID string

	handler := AuthAndCorrelationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReqID = r.Header.Get("x-request-id")
		w.WriteHeader(http.StatusOK)
	}))

	// Bearer token fallback
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer user-token-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedReqID == "" {
		t.Errorf("expected generated request ID")
	}

	// Custom x-user-uuid and x-request-id
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("x-user-uuid", "u-456")
	req2.Header.Set("x-request-id", "req-789")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	_ = capturedUser
}

func TestProtocolMultiplexer(t *testing.T) {
	client := &mockPenneServiceClient{}
	h := NewGatewayHandler(client, zap.NewNop())
	router := NewRouter(h, &Config{})

	multiplexer := &ProtocolMultiplexer{
		router:     router,
		grpcServer: grpc.NewServer(),
	}

	// HTTP request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	multiplexer.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// gRPC simulation request
	reqGRPC := httptest.NewRequest(http.MethodPost, "/penne.v1.PenneService/CheckHealth", nil)
	reqGRPC.ProtoMajor = 2
	reqGRPC.Header.Set("Content-Type", "application/grpc")
	recGRPC := httptest.NewRecorder()
	multiplexer.ServeHTTP(recGRPC, reqGRPC)
}

func TestHealthCheckHandler(t *testing.T) {
	// Healthy
	client := &mockPenneServiceClient{}
	h := NewGatewayHandler(client, zap.NewNop())
	router := NewRouter(h, &Config{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Downstream error
	clientErr := &mockPenneServiceClient{
		checkHealthFn: func(ctx context.Context, in *pennev1.HealthCheckRequest, opts ...grpc.CallOption) (*pennev1.HealthCheckResponse, error) {
			return nil, errors.New("downstream down")
		},
	}
	routerErr := NewRouter(NewGatewayHandler(clientErr, zap.NewNop()), &Config{})
	recErr := httptest.NewRecorder()
	routerErr.ServeHTTP(recErr, req)
	if recErr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recErr.Code)
	}
}

func TestUserHandlers(t *testing.T) {
	client := &mockPenneServiceClient{
		getUserFn: func(ctx context.Context, in *pennev1.GetUserRequest, opts ...grpc.CallOption) (*pennev1.UserResponse, error) {
			if in.GetUserUuid() == "u-1" {
				return &pennev1.UserResponse{Uuid: "u-1", Name: "Bob"}, nil
			}
			return nil, status.Error(codes.NotFound, "user not found")
		},
	}
	router := NewRouter(NewGatewayHandler(client, zap.NewNop()), &Config{})

	// Success
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user?id=u-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Not found
	reqNF := httptest.NewRequest(http.MethodGet, "/api/v1/user?id=unknown", nil)
	recNF := httptest.NewRecorder()
	router.ServeHTTP(recNF, reqNF)
	if recNF.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recNF.Code)
	}
}

func TestTransactionHandlers(t *testing.T) {
	client := &mockPenneServiceClient{
		createTransactionFn: func(ctx context.Context, in *pennev1.CreateTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error) {
			if in.GetAmount() < 0 {
				return nil, status.Error(codes.InvalidArgument, "negative amount")
			}
			return &pennev1.TransactionItem{Id: "t-1", Amount: in.GetAmount()}, nil
		},
		getTransactionFn: func(ctx context.Context, in *pennev1.GetTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error) {
			if in.GetTransactionUuid() == "t-1" {
				return &pennev1.TransactionItem{Id: "t-1"}, nil
			}
			return nil, status.Error(codes.NotFound, "not found")
		},
		getTransactionsByUserFn: func(ctx context.Context, in *pennev1.GetTransactionsByUserRequest, opts ...grpc.CallOption) (*pennev1.TransactionListResponse, error) {
			return &pennev1.TransactionListResponse{Transactions: []*pennev1.TransactionItem{{Id: "t-1"}}, TotalCount: 1}, nil
		},
		updateTransactionFn: func(ctx context.Context, in *pennev1.UpdateTransactionRequest, opts ...grpc.CallOption) (*pennev1.TransactionItem, error) {
			if in.GetId() == "t-1" {
				return &pennev1.TransactionItem{Id: "t-1"}, nil
			}
			return nil, status.Error(codes.NotFound, "not found")
		},
		deleteTransactionFn: func(ctx context.Context, in *pennev1.DeleteTransactionRequest, opts ...grpc.CallOption) (*pennev1.DeleteResponse, error) {
			if in.GetTransactionUuid() == "t-1" {
				return &pennev1.DeleteResponse{Success: true}, nil
			}
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	router := NewRouter(NewGatewayHandler(client, zap.NewNop()), &Config{})

	// Create success
	body := map[string]interface{}{"user_uuid": "u-1", "amount": 15.5}
	raw, _ := json.Marshal(body)
	reqC := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(raw))
	recC := httptest.NewRecorder()
	router.ServeHTTP(recC, reqC)
	if recC.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", recC.Code)
	}

	// Create bad body
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader([]byte("invalid-json")))
	recBad := httptest.NewRecorder()
	router.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", recBad.Code)
	}

	// Create invalid argument gRPC error
	bodyErr := map[string]interface{}{"user_uuid": "u-1", "amount": -5.0}
	rawErr, _ := json.Marshal(bodyErr)
	reqCErr := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewReader(rawErr))
	recCErr := httptest.NewRecorder()
	router.ServeHTTP(recCErr, reqCErr)
	if recCErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative amount, got %d", recCErr.Code)
	}

	// Get success
	reqG := httptest.NewRequest(http.MethodGet, "/api/v1/transaction?id=t-1", nil)
	recG := httptest.NewRecorder()
	router.ServeHTTP(recG, reqG)
	if recG.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recG.Code)
	}

	// Get not found
	reqGNF := httptest.NewRequest(http.MethodGet, "/api/v1/transaction?id=unknown", nil)
	recGNF := httptest.NewRecorder()
	router.ServeHTTP(recGNF, reqGNF)
	if recGNF.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recGNF.Code)
	}

	// Get list with query params
	reqL := httptest.NewRequest(http.MethodGet, "/api/v1/transactions?user_id=u-1&limit=10", nil)
	recL := httptest.NewRecorder()
	router.ServeHTTP(recL, reqL)
	if recL.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recL.Code)
	}

	// Get list with invalid and negative limit (falls back to default)
	reqLInv := httptest.NewRequest(http.MethodGet, "/api/v1/transactions?user_id=u-1&limit=abc", nil)
	recLInv := httptest.NewRecorder()
	router.ServeHTTP(recLInv, reqLInv)
	if recLInv.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recLInv.Code)
	}

	reqLNeg := httptest.NewRequest(http.MethodGet, "/api/v1/transactions?user_id=u-1&limit=-5", nil)
	recLNeg := httptest.NewRecorder()
	router.ServeHTTP(recLNeg, reqLNeg)
	if recLNeg.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recLNeg.Code)
	}

	// Get list downstream gRPC error
	clientTxErr := &mockPenneServiceClient{
		getTransactionsByUserFn: func(ctx context.Context, in *pennev1.GetTransactionsByUserRequest, opts ...grpc.CallOption) (*pennev1.TransactionListResponse, error) {
			return nil, status.Error(codes.Internal, "db failure")
		},
	}
	routerTxErr := NewRouter(NewGatewayHandler(clientTxErr, zap.NewNop()), &Config{})
	reqLTxErr := httptest.NewRequest(http.MethodGet, "/api/v1/transactions?user_id=u-1", nil)
	recLTxErr := httptest.NewRecorder()
	routerTxErr.ServeHTTP(recLTxErr, reqLTxErr)
	if recLTxErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", recLTxErr.Code)
	}

	// Update success
	bodyUp := map[string]interface{}{"id": "t-1", "user_uuid": "u-1", "amount": 20.0}
	rawUp, _ := json.Marshal(bodyUp)
	reqU := httptest.NewRequest(http.MethodPut, "/api/v1/transaction", bytes.NewReader(rawUp))
	recU := httptest.NewRecorder()
	router.ServeHTTP(recU, reqU)
	if recU.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recU.Code)
	}

	// Update bad json
	reqUBad := httptest.NewRequest(http.MethodPut, "/api/v1/transaction", bytes.NewReader([]byte("{bad")))
	recUBad := httptest.NewRecorder()
	router.ServeHTTP(recUBad, reqUBad)
	if recUBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", recUBad.Code)
	}

	// Update not found
	bodyUpNF := map[string]interface{}{"id": "unknown", "user_uuid": "u-1"}
	rawUpNF, _ := json.Marshal(bodyUpNF)
	reqUNF := httptest.NewRequest(http.MethodPut, "/api/v1/transaction", bytes.NewReader(rawUpNF))
	recUNF := httptest.NewRecorder()
	router.ServeHTTP(recUNF, reqUNF)
	if recUNF.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recUNF.Code)
	}

	// Delete success
	reqD := httptest.NewRequest(http.MethodDelete, "/api/v1/transaction?id=t-1", nil)
	recD := httptest.NewRecorder()
	router.ServeHTTP(recD, reqD)
	if recD.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recD.Code)
	}

	// Delete not found
	reqDNF := httptest.NewRequest(http.MethodDelete, "/api/v1/transaction?id=unknown", nil)
	recDNF := httptest.NewRecorder()
	router.ServeHTTP(recDNF, reqDNF)
	if recDNF.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recDNF.Code)
	}
}

func TestEnvelopeHandlers(t *testing.T) {
	client := &mockPenneServiceClient{
		getEnvelopeFn: func(ctx context.Context, in *pennev1.GetEnvelopeRequest, opts ...grpc.CallOption) (*pennev1.EnvelopeItem, error) {
			if in.GetEnvelopeId() == "e-1" {
				return &pennev1.EnvelopeItem{Id: "e-1", Name: "Groceries"}, nil
			}
			return nil, status.Error(codes.NotFound, "not found")
		},
		getEnvelopesByUserFn: func(ctx context.Context, in *pennev1.GetEnvelopesByUserRequest, opts ...grpc.CallOption) (*pennev1.EnvelopeListResponse, error) {
			if in.GetUserUuid() == "u-1" {
				return &pennev1.EnvelopeListResponse{Envelopes: []*pennev1.EnvelopeItem{{Id: "e-1"}}}, nil
			}
			return nil, status.Error(codes.Internal, "db failure")
		},
	}
	router := NewRouter(NewGatewayHandler(client, zap.NewNop()), &Config{})

	// Get envelope success
	req := httptest.NewRequest(http.MethodGet, "/api/v1/envelope?id=e-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Get envelope not found
	reqNF := httptest.NewRequest(http.MethodGet, "/api/v1/envelope?id=bad", nil)
	recNF := httptest.NewRecorder()
	router.ServeHTTP(recNF, reqNF)
	if recNF.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recNF.Code)
	}

	// Get envelopes by user success
	reqU := httptest.NewRequest(http.MethodGet, "/api/v1/envelopes?user_id=u-1", nil)
	recU := httptest.NewRecorder()
	router.ServeHTTP(recU, reqU)
	if recU.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recU.Code)
	}

	// Get envelopes by user error
	reqUErr := httptest.NewRequest(http.MethodGet, "/api/v1/envelopes?user_id=bad", nil)
	recUErr := httptest.NewRecorder()
	router.ServeHTTP(recUErr, reqUErr)
	if recUErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", recUErr.Code)
	}
}

func TestWriteGRPCErrorCodes(t *testing.T) {
	testCases := []struct {
		err          error
		expectedCode int
	}{
		{status.Error(codes.Unauthenticated, "unauthenticated"), http.StatusUnauthorized},
		{status.Error(codes.PermissionDenied, "denied"), http.StatusForbidden},
		{status.Error(codes.AlreadyExists, "exists"), http.StatusConflict},
		{status.Error(codes.Unknown, "unknown"), http.StatusInternalServerError},
		{errors.New("plain non-grpc error"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		rec := httptest.NewRecorder()
		writeGRPCError(rec, tc.err)
		if rec.Code != tc.expectedCode {
			t.Errorf("for error %v, expected %d, got %d", tc.err, tc.expectedCode, rec.Code)
		}
	}
}

func TestNewPenneClient(t *testing.T) {
	// Empty target error
	_, _, err := NewPenneClient(&Config{PenneServiceTarget: ""})
	if err == nil {
		t.Fatal("expected error for empty target")
	}

	// Valid target
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer lis.Close()

	client, pool, err := NewPenneClient(&Config{PenneServiceTarget: lis.Addr().String()})
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	defer pool.Close()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestStartServer_Lifecycle(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	_, port, _ := net.SplitHostPort(lis.Addr().String())
	lis.Close()

	cfg := &Config{
		Port:               port,
		PenneServiceTarget: "127.0.0.1:50051",
		ReadTimeout:        5 * time.Second,
		WriteTimeout:       5 * time.Second,
	}

	client := &mockPenneServiceClient{}
	handler := NewGatewayHandler(client, zap.NewNop())
	router := NewRouter(handler, cfg)

	app := fx.New(
		fx.NopLogger,
		fx.Provide(
			func() *Config { return cfg },
			func() *mux.Router { return router },
			zap.NewNop,
		),
		fx.Invoke(StartServer),
	)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		t.Fatalf("app start error: %v", err)
	}

	// Verify server responds
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStop()
	if err := app.Stop(stopCtx); err != nil {
		t.Fatalf("app stop error: %v", err)
	}
}

func TestReverseProxyFallback(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"proxied successfully"}`))
	}))
	defer backend.Close()

	cfg := &Config{PenneHTTPTarget: backend.URL}
	client := &mockPenneServiceClient{}
	h := NewGatewayHandler(client, zap.NewNop())
	router := NewRouter(h, cfg)

	req := httptest.NewRequest(http.MethodGet, "/envelope-groups?user_uuid=test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPulseProxyForwarding(t *testing.T) {
	pulseBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","message":"pulse proxy received"}`))
	}))
	defer pulseBackend.Close()

	cfg := &Config{PulseHTTPTarget: pulseBackend.URL}
	client := &mockPenneServiceClient{}
	h := NewGatewayHandler(client, zap.NewNop())
	router := NewRouter(h, cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/sync", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

type mockNutritionServiceClient struct {
	processImageFn    func(ctx context.Context, in *nutritionv1.ProcessImageRequest, opts ...grpc.CallOption) (*nutritionv1.JobStatus, error)
	getJobStatusFn    func(ctx context.Context, in *nutritionv1.GetJobStatusRequest, opts ...grpc.CallOption) (*nutritionv1.JobStatus, error)
	getDailyLogFn     func(ctx context.Context, in *nutritionv1.GetDailyLogRequest, opts ...grpc.CallOption) (*nutritionv1.DailyNutritionSummary, error)
	getGoalsFn        func(ctx context.Context, in *nutritionv1.GetGoalsRequest, opts ...grpc.CallOption) (*nutritionv1.NutritionGoals, error)
	setGoalsFn        func(ctx context.Context, in *nutritionv1.SetGoalsRequest, opts ...grpc.CallOption) (*nutritionv1.NutritionGoals, error)
	addManualMealFn   func(ctx context.Context, in *nutritionv1.AddManualMealRequest, opts ...grpc.CallOption) (*nutritionv1.MealLog, error)
	deleteMealLogFn   func(ctx context.Context, in *nutritionv1.DeleteMealLogRequest, opts ...grpc.CallOption) (*nutritionv1.DeleteResponse, error)
	updateMealItemFn  func(ctx context.Context, in *nutritionv1.UpdateMealItemRequest, opts ...grpc.CallOption) (*nutritionv1.MealItem, error)
	deleteMealItemFn  func(ctx context.Context, in *nutritionv1.DeleteMealItemRequest, opts ...grpc.CallOption) (*nutritionv1.DeleteResponse, error)
}

func (m *mockNutritionServiceClient) CheckHealth(ctx context.Context, in *nutritionv1.HealthCheckRequest, opts ...grpc.CallOption) (*nutritionv1.HealthCheckResponse, error) {
	return &nutritionv1.HealthCheckResponse{Status: "SERVING"}, nil
}
func (m *mockNutritionServiceClient) ProcessImage(ctx context.Context, in *nutritionv1.ProcessImageRequest, opts ...grpc.CallOption) (*nutritionv1.JobStatus, error) {
	if m.processImageFn != nil {
		return m.processImageFn(ctx, in, opts...)
	}
	return &nutritionv1.JobStatus{JobId: "job-1", Status: "PENDING"}, nil
}
func (m *mockNutritionServiceClient) GetJobStatus(ctx context.Context, in *nutritionv1.GetJobStatusRequest, opts ...grpc.CallOption) (*nutritionv1.JobStatus, error) {
	if m.getJobStatusFn != nil {
		return m.getJobStatusFn(ctx, in, opts...)
	}
	return &nutritionv1.JobStatus{JobId: in.GetJobId(), Status: "COMPLETED"}, nil
}
func (m *mockNutritionServiceClient) GetDailyLog(ctx context.Context, in *nutritionv1.GetDailyLogRequest, opts ...grpc.CallOption) (*nutritionv1.DailyNutritionSummary, error) {
	if m.getDailyLogFn != nil {
		return m.getDailyLogFn(ctx, in, opts...)
	}
	return &nutritionv1.DailyNutritionSummary{Date: in.GetDate(), TotalCalories: 1500}, nil
}
func (m *mockNutritionServiceClient) GetGoals(ctx context.Context, in *nutritionv1.GetGoalsRequest, opts ...grpc.CallOption) (*nutritionv1.NutritionGoals, error) {
	if m.getGoalsFn != nil {
		return m.getGoalsFn(ctx, in, opts...)
	}
	return &nutritionv1.NutritionGoals{CaloriesTarget: 2200}, nil
}
func (m *mockNutritionServiceClient) SetGoals(ctx context.Context, in *nutritionv1.SetGoalsRequest, opts ...grpc.CallOption) (*nutritionv1.NutritionGoals, error) {
	if m.setGoalsFn != nil {
		return m.setGoalsFn(ctx, in, opts...)
	}
	return in.GetGoals(), nil
}
func (m *mockNutritionServiceClient) AddManualMeal(ctx context.Context, in *nutritionv1.AddManualMealRequest, opts ...grpc.CallOption) (*nutritionv1.MealLog, error) {
	if m.addManualMealFn != nil {
		return m.addManualMealFn(ctx, in, opts...)
	}
	return &nutritionv1.MealLog{Id: "meal-1", TotalCalories: 500}, nil
}
func (m *mockNutritionServiceClient) DeleteMealLog(ctx context.Context, in *nutritionv1.DeleteMealLogRequest, opts ...grpc.CallOption) (*nutritionv1.DeleteResponse, error) {
	if m.deleteMealLogFn != nil {
		return m.deleteMealLogFn(ctx, in, opts...)
	}
	return &nutritionv1.DeleteResponse{Success: true}, nil
}
func (m *mockNutritionServiceClient) UpdateMealItem(ctx context.Context, in *nutritionv1.UpdateMealItemRequest, opts ...grpc.CallOption) (*nutritionv1.MealItem, error) {
	if m.updateMealItemFn != nil {
		return m.updateMealItemFn(ctx, in, opts...)
	}
	return &nutritionv1.MealItem{Id: in.GetItemId(), Calories: in.GetCalories()}, nil
}
func (m *mockNutritionServiceClient) DeleteMealItem(ctx context.Context, in *nutritionv1.DeleteMealItemRequest, opts ...grpc.CallOption) (*nutritionv1.DeleteResponse, error) {
	if m.deleteMealItemFn != nil {
		return m.deleteMealItemFn(ctx, in, opts...)
	}
	return &nutritionv1.DeleteResponse{Success: true}, nil
}

func TestNutritionEndpoints(t *testing.T) {
	penneClient := &mockPenneServiceClient{}
	nutriClient := &mockNutritionServiceClient{}
	h := NewGatewayHandlerWithNutrition(penneClient, nutriClient, zap.NewNop())
	router := NewRouter(h, &Config{})

	// 1. Upload meal image with direct binary (clipboard paste)
	reqImg := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", bytes.NewReader([]byte("png-binary")))
	reqImg.Header.Set("Content-Type", "image/png")
	recImg := httptest.NewRecorder()
	router.ServeHTTP(recImg, reqImg)
	if recImg.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", recImg.Code, recImg.Body.String())
	}

	// 2. Upload with multipart form data using 'image' field
	mpBody := &bytes.Buffer{}
	writer := multipart.NewWriter(mpBody)
	part, _ := writer.CreateFormFile("image", "meal.jpg")
	part.Write([]byte("fake-jpeg-bytes"))
	writer.WriteField("meal_type", "lunch")
	writer.WriteField("notes", "healthy meal")
	writer.Close()

	reqMP := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", mpBody)
	reqMP.Header.Set("Content-Type", writer.FormDataContentType())
	recMP := httptest.NewRecorder()
	router.ServeHTTP(recMP, reqMP)
	if recMP.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for multipart 'image', got %d: %s", recMP.Code, recMP.Body.String())
	}

	// 3. Upload with multipart form data using 'file' field fallback
	mpBodyF := &bytes.Buffer{}
	writerF := multipart.NewWriter(mpBodyF)
	partF, _ := writerF.CreateFormFile("file", "meal.png")
	partF.Write([]byte("fake-png-bytes"))
	writerF.Close()

	reqMPF := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", mpBodyF)
	reqMPF.Header.Set("Content-Type", writerF.FormDataContentType())
	recMPF := httptest.NewRecorder()
	router.ServeHTTP(recMPF, reqMPF)
	if recMPF.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for multipart 'file', got %d: %s", recMPF.Code, recMPF.Body.String())
	}

	// 4. Upload with multipart form missing both 'image' and 'file'
	mpBodyMissing := &bytes.Buffer{}
	writerMissing := multipart.NewWriter(mpBodyMissing)
	writerMissing.WriteField("notes", "missing file")
	writerMissing.Close()

	reqMPMissing := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", mpBodyMissing)
	reqMPMissing.Header.Set("Content-Type", writerMissing.FormDataContentType())
	recMPMissing := httptest.NewRecorder()
	router.ServeHTTP(recMPMissing, reqMPMissing)
	if recMPMissing.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing file field, got %d", recMPMissing.Code)
	}

	// 5. Upload with multipart form containing empty file data
	mpBodyEmpty := &bytes.Buffer{}
	writerEmpty := multipart.NewWriter(mpBodyEmpty)
	_, _ = writerEmpty.CreateFormFile("image", "empty.jpg")
	writerEmpty.Close()

	reqMPEmpty := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", mpBodyEmpty)
	reqMPEmpty.Header.Set("Content-Type", writerEmpty.FormDataContentType())
	recMPEmpty := httptest.NewRecorder()
	router.ServeHTTP(recMPEmpty, reqMPEmpty)
	if recMPEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty multipart file data, got %d", recMPEmpty.Code)
	}

	// 6. Upload with malformed multipart boundary
	reqBadMP := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", bytes.NewReader([]byte("not valid body")))
	reqBadMP.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
	recBadMP := httptest.NewRecorder()
	router.ServeHTTP(recBadMP, reqBadMP)
	if recBadMP.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed multipart, got %d", recBadMP.Code)
	}

	// 7. Upload with empty image/png body
	reqEmptyDirect := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", bytes.NewReader([]byte("")))
	reqEmptyDirect.Header.Set("Content-Type", "image/png")
	recEmptyDirect := httptest.NewRecorder()
	router.ServeHTTP(recEmptyDirect, reqEmptyDirect)
	if recEmptyDirect.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty image binary, got %d", recEmptyDirect.Code)
	}

	// 8. Upload with invalid content type
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", bytes.NewReader([]byte("text")))
	reqBad.Header.Set("Content-Type", "text/plain")
	recBad := httptest.NewRecorder()
	router.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", recBad.Code)
	}

	// 9. Get job status
	reqJob := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition/jobs/job-1", nil)
	recJob := httptest.NewRecorder()
	router.ServeHTTP(recJob, reqJob)
	if recJob.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recJob.Code)
	}

	// 10. Get daily nutrition
	reqDaily := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition/daily?date=2026-10-13", nil)
	recDaily := httptest.NewRecorder()
	router.ServeHTTP(recDaily, reqDaily)
	if recDaily.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recDaily.Code)
	}

	// 11. Goals GET and POST
	reqGetGoals := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition/goals", nil)
	recGetGoals := httptest.NewRecorder()
	router.ServeHTTP(recGetGoals, reqGetGoals)
	if recGetGoals.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recGetGoals.Code)
	}

	goalsBody, _ := json.Marshal(map[string]interface{}{
		"calories_target": 2400,
		"protein_grams":   180,
	})
	reqSetGoals := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/goals", bytes.NewReader(goalsBody))
	recSetGoals := httptest.NewRecorder()
	router.ServeHTTP(recSetGoals, reqSetGoals)
	if recSetGoals.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recSetGoals.Code)
	}

	// 12. Manual meal POST and DELETE
	mealBody, _ := json.Marshal(map[string]interface{}{
		"meal_type": "lunch",
		"items": []map[string]interface{}{
			{"name": "Steak", "calories": 400, "protein_grams": 45},
		},
	})
	reqMeal := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/meals", bytes.NewReader(mealBody))
	recMeal := httptest.NewRecorder()
	router.ServeHTTP(recMeal, reqMeal)
	if recMeal.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", recMeal.Code)
	}

	reqDelMeal := httptest.NewRequest(http.MethodDelete, "/api/v1/nutrition/meals/meal-1", nil)
	recDelMeal := httptest.NewRecorder()
	router.ServeHTTP(recDelMeal, reqDelMeal)
	if recDelMeal.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recDelMeal.Code)
	}

	// 13. Meal item PUT and DELETE
	itemBody, _ := json.Marshal(map[string]interface{}{
		"name":     "Updated Steak",
		"calories": 450,
	})
	reqItem := httptest.NewRequest(http.MethodPut, "/api/v1/nutrition/items/item-1", bytes.NewReader(itemBody))
	recItem := httptest.NewRecorder()
	router.ServeHTTP(recItem, reqItem)
	if recItem.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recItem.Code)
	}

	reqDelItem := httptest.NewRequest(http.MethodDelete, "/api/v1/nutrition/items/item-1", nil)
	recDelItem := httptest.NewRecorder()
	router.ServeHTTP(recDelItem, reqDelItem)
	if recDelItem.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recDelItem.Code)
	}
}

func TestNutritionErrorBranches(t *testing.T) {
	penneClient := &mockPenneServiceClient{}
	nutriErrClient := &mockNutritionServiceClient{
		processImageFn: func(ctx context.Context, in *nutritionv1.ProcessImageRequest, opts ...grpc.CallOption) (*nutritionv1.JobStatus, error) {
			return nil, status.Error(codes.Internal, "inference error")
		},
		getJobStatusFn: func(ctx context.Context, in *nutritionv1.GetJobStatusRequest, opts ...grpc.CallOption) (*nutritionv1.JobStatus, error) {
			return nil, status.Error(codes.NotFound, "job not found")
		},
		getDailyLogFn: func(ctx context.Context, in *nutritionv1.GetDailyLogRequest, opts ...grpc.CallOption) (*nutritionv1.DailyNutritionSummary, error) {
			return nil, status.Error(codes.Internal, "daily log error")
		},
		getGoalsFn: func(ctx context.Context, in *nutritionv1.GetGoalsRequest, opts ...grpc.CallOption) (*nutritionv1.NutritionGoals, error) {
			return nil, status.Error(codes.Internal, "goals error")
		},
		setGoalsFn: func(ctx context.Context, in *nutritionv1.SetGoalsRequest, opts ...grpc.CallOption) (*nutritionv1.NutritionGoals, error) {
			return nil, status.Error(codes.Internal, "set goals error")
		},
		addManualMealFn: func(ctx context.Context, in *nutritionv1.AddManualMealRequest, opts ...grpc.CallOption) (*nutritionv1.MealLog, error) {
			return nil, status.Error(codes.Internal, "add meal error")
		},
		deleteMealLogFn: func(ctx context.Context, in *nutritionv1.DeleteMealLogRequest, opts ...grpc.CallOption) (*nutritionv1.DeleteResponse, error) {
			return nil, status.Error(codes.NotFound, "meal not found")
		},
		updateMealItemFn: func(ctx context.Context, in *nutritionv1.UpdateMealItemRequest, opts ...grpc.CallOption) (*nutritionv1.MealItem, error) {
			return nil, status.Error(codes.NotFound, "item not found")
		},
		deleteMealItemFn: func(ctx context.Context, in *nutritionv1.DeleteMealItemRequest, opts ...grpc.CallOption) (*nutritionv1.DeleteResponse, error) {
			return nil, status.Error(codes.NotFound, "item not found")
		},
	}
	router := NewRouter(NewGatewayHandlerWithNutrition(penneClient, nutriErrClient, zap.NewNop()), &Config{})

	// 1. Process image error
	reqImg := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/upload", bytes.NewReader([]byte("png-binary")))
	reqImg.Header.Set("Content-Type", "image/png")
	recImg := httptest.NewRecorder()
	router.ServeHTTP(recImg, reqImg)
	if recImg.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", recImg.Code)
	}

	// 2. Get job status error
	reqJob := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition/jobs/job-err", nil)
	recJob := httptest.NewRecorder()
	router.ServeHTTP(recJob, reqJob)
	if recJob.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recJob.Code)
	}

	// 3. Get daily log error
	reqDaily := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition/daily?date=2026-10-13", nil)
	recDaily := httptest.NewRecorder()
	router.ServeHTTP(recDaily, reqDaily)
	if recDaily.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", recDaily.Code)
	}

	// 4. Get goals error
	reqGoals := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition/goals", nil)
	recGoals := httptest.NewRecorder()
	router.ServeHTTP(recGoals, reqGoals)
	if recGoals.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", recGoals.Code)
	}

	// 5. Set goals bad JSON and error
	reqSetGoalsBad := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/goals", bytes.NewReader([]byte("{bad-json")))
	recSetGoalsBad := httptest.NewRecorder()
	router.ServeHTTP(recSetGoalsBad, reqSetGoalsBad)
	if recSetGoalsBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", recSetGoalsBad.Code)
	}

	goalsBody, _ := json.Marshal(map[string]interface{}{"calories_target": 2000})
	reqSetGoalsErr := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/goals", bytes.NewReader(goalsBody))
	recSetGoalsErr := httptest.NewRecorder()
	router.ServeHTTP(recSetGoalsErr, reqSetGoalsErr)
	if recSetGoalsErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", recSetGoalsErr.Code)
	}

	// 6. Add manual meal bad JSON and error
	reqMealBad := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/meals", bytes.NewReader([]byte("{bad-json")))
	recMealBad := httptest.NewRecorder()
	router.ServeHTTP(recMealBad, reqMealBad)
	if recMealBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", recMealBad.Code)
	}

	mealBody, _ := json.Marshal(map[string]interface{}{"meal_type": "lunch"})
	reqMealErr := httptest.NewRequest(http.MethodPost, "/api/v1/nutrition/meals", bytes.NewReader(mealBody))
	recMealErr := httptest.NewRecorder()
	router.ServeHTTP(recMealErr, reqMealErr)
	if recMealErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", recMealErr.Code)
	}

	// 7. Delete meal error
	reqDelMeal := httptest.NewRequest(http.MethodDelete, "/api/v1/nutrition/meals/meal-err", nil)
	recDelMeal := httptest.NewRecorder()
	router.ServeHTTP(recDelMeal, reqDelMeal)
	if recDelMeal.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recDelMeal.Code)
	}

	// 8. Update item bad JSON and error
	reqItemBad := httptest.NewRequest(http.MethodPut, "/api/v1/nutrition/items/item-1", bytes.NewReader([]byte("{bad-json")))
	recItemBad := httptest.NewRecorder()
	router.ServeHTTP(recItemBad, reqItemBad)
	if recItemBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", recItemBad.Code)
	}

	itemBody, _ := json.Marshal(map[string]interface{}{"name": "item"})
	reqItemErr := httptest.NewRequest(http.MethodPut, "/api/v1/nutrition/items/item-1", bytes.NewReader(itemBody))
	recItemErr := httptest.NewRecorder()
	router.ServeHTTP(recItemErr, reqItemErr)
	if recItemErr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recItemErr.Code)
	}

	// 9. Delete item error
	reqDelItem := httptest.NewRequest(http.MethodDelete, "/api/v1/nutrition/items/item-err", nil)
	recDelItem := httptest.NewRecorder()
	router.ServeHTTP(recDelItem, reqDelItem)
	if recDelItem.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", recDelItem.Code)
	}

	// 10. Nil nutrition client on all routes
	hNil := NewGatewayHandler(penneClient, zap.NewNop())
	rNil := NewRouter(hNil, &Config{})
	nilRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/nutrition/upload"},
		{http.MethodGet, "/api/v1/nutrition/jobs/1"},
		{http.MethodGet, "/api/v1/nutrition/daily"},
		{http.MethodGet, "/api/v1/nutrition/goals"},
		{http.MethodPost, "/api/v1/nutrition/goals"},
		{http.MethodPost, "/api/v1/nutrition/meals"},
		{http.MethodDelete, "/api/v1/nutrition/meals/1"},
		{http.MethodPut, "/api/v1/nutrition/items/1"},
		{http.MethodDelete, "/api/v1/nutrition/items/1"},
	}
	for _, nr := range nilRoutes {
		req := httptest.NewRequest(nr.method, nr.path, nil)
		rec := httptest.NewRecorder()
		rNil.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("for %s %s with nil client, expected 503, got %d", nr.method, nr.path, rec.Code)
		}
	}
}

func TestNewGatewayHandlerFX(t *testing.T) {
	penneClient := &mockPenneServiceClient{}
	nutriClient := &mockNutritionServiceClient{}
	h := NewGatewayHandlerFX(GatewayHandlerParams{
		PenneClient:     penneClient,
		Log:             zap.NewNop(),
		NutritionClient: nutriClient,
	})
	if h == nil || h.nutritionClient == nil {
		t.Fatal("expected non-nil gateway handler with nutrition client")
	}
}

func TestNewNutritionClient(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer lis.Close()

	client, err := NewNutritionClient(&Config{NutritionServiceTarget: lis.Addr().String()})
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Empty target defaults to 127.0.0.1:50052
	clientDef, err := NewNutritionClient(&Config{NutritionServiceTarget: ""})
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	if clientDef == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestStartServer_NilLCAndError(t *testing.T) {
	cfg := &Config{Port: "8080"}
	router := mux.NewRouter()
	srv, err := StartServer(cfg, router, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	// Port collision error on Start
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer lis.Close()
	_, port, _ := net.SplitHostPort(lis.Addr().String())

	cfgErr := &Config{Port: port}
	app := fx.New(
		fx.NopLogger,
		fx.Provide(
			func() *Config { return cfgErr },
			func() *mux.Router { return router },
			zap.NewNop,
		),
		fx.Invoke(StartServer),
	)
	startCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err == nil {
		t.Fatal("expected port collision error on app.Start, got nil")
	}
}

