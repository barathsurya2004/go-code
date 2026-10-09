package observability

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestMetricsMiddleware(t *testing.T) {
	r := mux.NewRouter()
	r.Use(MetricsMiddleware())
	r.HandleFunc("/test/{id}", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}).Methods("GET")

	req := httptest.NewRequest("GET", "/test/123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	// Test fallback when route template is not matched
	unmatchedReq := httptest.NewRequest("POST", "/unknown", nil)
	unmatchedRR := httptest.NewRecorder()
	MetricsMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(unmatchedRR, unmatchedReq)

	assert.Equal(t, http.StatusOK, unmatchedRR.Code)
}

func TestUptimeTracker_CheckLastOutage(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "uptime.json")

	// 1. Missing file
	tracker := NewUptimeTracker(filePath, 5*time.Second, logger)
	tracker.CheckLastOutage(time.Now())

	// 2. Corrupt file
	require.NoError(t, os.WriteFile(filePath, []byte("invalid json"), 0644))
	tracker.CheckLastOutage(time.Now())

	// 3. Small gap (no outage)
	now := time.Now()
	recentState := HeartbeatState{
		LastHeartbeat: now.Add(-5 * time.Second),
		CleanShutdown: false,
	}
	recentBytes, _ := json.Marshal(recentState)
	require.NoError(t, os.WriteFile(filePath, recentBytes, 0644))
	tracker.CheckLastOutage(now)

	// 4. Large gap with unclean shutdown (powercut or crash)
	outageState := HeartbeatState{
		LastHeartbeat: now.Add(-10 * time.Minute),
		CleanShutdown: false,
	}
	outageBytes, _ := json.Marshal(outageState)
	require.NoError(t, os.WriteFile(filePath, outageBytes, 0644))
	tracker.CheckLastOutage(now)

	// 5. Large gap with clean shutdown
	cleanState := HeartbeatState{
		LastHeartbeat: now.Add(-10 * time.Minute),
		CleanShutdown: true,
	}
	cleanBytes, _ := json.Marshal(cleanState)
	require.NoError(t, os.WriteFile(filePath, cleanBytes, 0644))
	tracker.CheckLastOutage(now)
}

func TestUptimeTracker_StartAndStop(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sub", "uptime.json")

	tracker := NewUptimeTracker(filePath, 20*time.Millisecond, logger)

	ctx, cancel := context.WithCancel(context.Background())
	tracker.StartHeartbeat(ctx)

	time.Sleep(60 * time.Millisecond)

	// Verify file was written
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	var state HeartbeatState
	require.NoError(t, json.Unmarshal(data, &state))
	assert.False(t, state.CleanShutdown)

	// Graceful stop
	tracker.Stop()
	tracker.Stop() // Test idempotency

	time.Sleep(20 * time.Millisecond)
	data, err = os.ReadFile(filePath)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &state))
	assert.True(t, state.CleanShutdown)

	cancel()
}

func TestUptimeTracker_ContextCancel(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "uptime.json")

	tracker := NewUptimeTracker(filePath, 20*time.Millisecond, logger)

	ctx, cancel := context.WithCancel(context.Background())
	tracker.StartHeartbeat(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel() // Cancel via context
	time.Sleep(30 * time.Millisecond)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	var state HeartbeatState
	require.NoError(t, json.Unmarshal(data, &state))
	assert.True(t, state.CleanShutdown)
}

func TestUptimeTracker_EnvDefaults(t *testing.T) {
	t.Setenv("UPTIME_STATE_FILE", "custom_path.json")
	tr := NewUptimeTracker("", -1, zap.NewNop())
	assert.Equal(t, "custom_path.json", tr.filePath)
	assert.Equal(t, 10*time.Second, tr.interval)

	t.Setenv("UPTIME_STATE_FILE", "")
	tr2 := NewUptimeTracker("", 0, zap.NewNop())
	assert.Equal(t, "data/uptime_state.json", tr2.filePath)
}

func TestNetworkProbe_StateTransitions(t *testing.T) {
	logger := zap.NewNop()
	probe := NewNetworkProbe("", -1, logger)
	assert.Equal(t, "1.1.1.1:53", probe.target)
	assert.Equal(t, 15*time.Second, probe.interval)

	mockState := true
	probe.SetProber(func(ctx context.Context) bool {
		return mockState
	})

	ctx := context.Background()
	now := time.Now()

	// 1. Initial state: online
	assert.True(t, probe.ProbeOnce(ctx, now))

	// 2. Drop connection
	mockState = false
	dropTime := now.Add(5 * time.Second)
	assert.False(t, probe.ProbeOnce(ctx, dropTime))
	assert.False(t, probe.isOnline)

	// 3. Stays offline
	assert.False(t, probe.ProbeOnce(ctx, dropTime.Add(2*time.Second)))

	// 4. Connection restored
	mockState = true
	restoreTime := dropTime.Add(10 * time.Second)
	assert.True(t, probe.ProbeOnce(ctx, restoreTime))
	assert.True(t, probe.isOnline)

	// 5. Stays online
	assert.True(t, probe.ProbeOnce(ctx, restoreTime.Add(2*time.Second)))
}

func TestNetworkProbe_StartAndStop(t *testing.T) {
	logger := zap.NewNop()
	probe := NewNetworkProbe("test:80", 20*time.Millisecond, logger)

	probe.SetProber(func(ctx context.Context) bool {
		return true
	})

	ctx, cancel := context.WithCancel(context.Background())
	probe.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	probe.Stop()
	probe.Stop() // Idempotency
	cancel()
}

func TestNetworkProbe_DefaultProber(t *testing.T) {
	// Start a local TCP listener to verify defaultProber works
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	p := defaultProber(listener.Addr().String())
	assert.True(t, p(context.Background()))

	// Test failing prober on closed port
	pFail := defaultProber("127.0.0.1:1")
	assert.False(t, pFail(context.Background()))
}

func TestObservabilityModule(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("UPTIME_STATE_FILE", filepath.Join(tmpDir, "state.json"))

	app := fxtest.New(
		t,
		fx.Provide(func() *zap.Logger { return zap.NewNop() }),
		Module,
	)

	app.RequireStart()
	time.Sleep(30 * time.Millisecond)
	app.RequireStop()
}
