package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

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

	// Preflight OPTIONS
	reqOpt := httptest.NewRequest(http.MethodOptions, "/test", nil)
	recOpt := httptest.NewRecorder()
	handler.ServeHTTP(recOpt, reqOpt)
	if recOpt.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", recOpt.Code)
	}

	// Normal request
	reqGet := httptest.NewRequest(http.MethodGet, "/test", nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recGet.Code)
	}
	if recGet.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected * CORS header")
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
