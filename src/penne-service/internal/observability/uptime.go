package observability

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	// ServerOutagesTotal tracks unexpected power cuts/crashes vs clean restarts
	ServerOutagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "penne_server_outages_total",
			Help: "Total number of server restarts grouped by clean vs sudden outage/power cut",
		},
		[]string{"type"}, // "powercut_or_crash" or "clean_restart"
	)

	// LastOutageDurationSeconds records how long the server was offline before this boot
	LastOutageDurationSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "penne_server_last_outage_duration_seconds",
			Help: "Duration of the most recently detected downtime in seconds",
		},
	)

	// ServerUptimeSeconds tracks running uptime for the active process
	ServerUptimeSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "penne_server_uptime_seconds",
			Help: "Current process uptime in seconds",
		},
	)
)

// HeartbeatState stores persistent state to disk
type HeartbeatState struct {
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CleanShutdown bool      `json:"clean_shutdown"`
}

type UptimeTracker struct {
	filePath  string
	interval  time.Duration
	startTime time.Time
	logger    *zap.Logger
	stopCh    chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
}

func NewUptimeTracker(filePath string, interval time.Duration, logger *zap.Logger) *UptimeTracker {
	if filePath == "" {
		filePath = os.Getenv("UPTIME_STATE_FILE")
		if filePath == "" {
			filePath = "data/uptime_state.json"
		}
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &UptimeTracker{
		filePath:  filePath,
		interval:  interval,
		startTime: time.Now(),
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// CheckLastOutage reads the previous heartbeat from disk to evaluate previous downtime
func (ut *UptimeTracker) CheckLastOutage(now time.Time) {
	data, err := os.ReadFile(ut.filePath)
	if err != nil {
		ut.logger.Info("No existing uptime state file found (first run or state reset)", zap.String("path", ut.filePath))
		return
	}

	var state HeartbeatState
	if err := json.Unmarshal(data, &state); err != nil {
		ut.logger.Warn("Failed to decode previous uptime state file", zap.Error(err))
		return
	}

	gap := now.Sub(state.LastHeartbeat)
	if gap > 3*ut.interval {
		if !state.CleanShutdown {
			ut.logger.Error("🚨 UNEXPECTED OUTAGE DETECTED (Power cut, kernel crash, or hard kill)",
				zap.Duration("downtime", gap),
				zap.Time("last_heartbeat", state.LastHeartbeat),
				zap.Time("resumed_at", now),
			)
			ServerOutagesTotal.WithLabelValues("powercut_or_crash").Inc()
		} else {
			ut.logger.Info("Server restarted after clean shutdown",
				zap.Duration("offline_duration", gap),
				zap.Time("shutdown_at", state.LastHeartbeat),
				zap.Time("restarted_at", now),
			)
			ServerOutagesTotal.WithLabelValues("clean_restart").Inc()
		}
		LastOutageDurationSeconds.Set(gap.Seconds())
	}
}

// StartHeartbeat begins periodically writing heartbeat timestamps to disk
func (ut *UptimeTracker) StartHeartbeat(ctx context.Context) {
	ut.writeState(time.Now(), false)

	ut.wg.Add(1)
	ticker := time.NewTicker(ut.interval)
	go func() {
		defer ut.wg.Done()
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				ut.writeState(now, false)
				ServerUptimeSeconds.Set(time.Since(ut.startTime).Seconds())
			case <-ut.stopCh:
				ut.writeState(time.Now(), true)
				return
			case <-ctx.Done():
				ut.writeState(time.Now(), true)
				return
			}
		}
	}()
}

// Stop signals graceful shutdown and marks clean exit on disk
func (ut *UptimeTracker) Stop() {
	ut.mu.Lock()
	select {
	case <-ut.stopCh:
		ut.mu.Unlock()
	default:
		close(ut.stopCh)
		ut.mu.Unlock()
		ut.wg.Wait()
	}
}

func (ut *UptimeTracker) writeState(now time.Time, clean bool) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	state := HeartbeatState{
		LastHeartbeat: now,
		CleanShutdown: clean,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	dir := filepath.Dir(ut.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	// Atomic write via temp file
	tmpFile := ut.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err == nil {
		_ = os.Rename(tmpFile, ut.filePath)
	}
}
