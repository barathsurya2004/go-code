package observability

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	// ServerNetworkStatus indicates current internet connectivity (1 = connected, 0 = dropped)
	ServerNetworkStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "penne_server_network_status",
			Help: "Current internet connection status (1 = online, 0 = offline/ISP drop)",
		},
	)

	// ServerNetworkDropoutsTotal counts total ISP / WiFi outages
	ServerNetworkDropoutsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "penne_server_network_dropouts_total",
			Help: "Total number of network/ISP disconnect events detected",
		},
	)

	// ServerLastNetworkDropDurationSeconds records duration of the most recent network outage
	ServerLastNetworkDropDurationSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "penne_server_last_network_drop_duration_seconds",
			Help: "Duration of the last detected network outage in seconds",
		},
	)
)

type ProberFunc func(ctx context.Context) bool

type NetworkProbe struct {
	target        string
	interval      time.Duration
	prober        ProberFunc
	logger        *zap.Logger
	isOnline      bool
	dropStartTime time.Time
	stopCh        chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
}

func defaultProber(target string) ProberFunc {
	return func(ctx context.Context) bool {
		d := net.Dialer{Timeout: 2 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", target)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}
}

func NewNetworkProbe(target string, interval time.Duration, logger *zap.Logger) *NetworkProbe {
	if target == "" {
		target = "1.1.1.1:53"
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}
	return &NetworkProbe{
		target:   target,
		interval: interval,
		prober:   defaultProber(target),
		logger:   logger,
		isOnline: true,
		stopCh:   make(chan struct{}),
	}
}

// SetProber allows injecting mock probe functions for testing
func (np *NetworkProbe) SetProber(p ProberFunc) {
	np.mu.Lock()
	defer np.mu.Unlock()
	np.prober = p
}

// ProbeOnce executes a single check and handles state transitions
func (np *NetworkProbe) ProbeOnce(ctx context.Context, now time.Time) bool {
	np.mu.Lock()
	defer np.mu.Unlock()

	online := np.prober(ctx)
	if online {
		ServerNetworkStatus.Set(1)
		if !np.isOnline {
			// Transition from offline to online
			duration := now.Sub(np.dropStartTime)
			np.logger.Info("🌐 Internet connectivity restored",
				zap.Duration("offline_duration", duration),
				zap.Time("disconnected_at", np.dropStartTime),
				zap.Time("reconnected_at", now),
			)
			ServerNetworkDropoutsTotal.Inc()
			ServerLastNetworkDropDurationSeconds.Set(duration.Seconds())
			np.isOnline = true
		}
	} else {
		ServerNetworkStatus.Set(0)
		if np.isOnline {
			// Transition from online to offline
			np.dropStartTime = now
			np.isOnline = false
			np.logger.Warn("⚠️ Internet connectivity lost (ISP or local WiFi drop detected)",
				zap.Time("disconnected_at", now),
			)
		}
	}
	return online
}

// Start runs background periodic probing
func (np *NetworkProbe) Start(ctx context.Context) {
	// Initial probe immediately
	np.ProbeOnce(ctx, time.Now())

	np.wg.Add(1)
	ticker := time.NewTicker(np.interval)
	go func() {
		defer np.wg.Done()
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				np.ProbeOnce(probeCtx, now)
				cancel()
			case <-np.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (np *NetworkProbe) Stop() {
	np.mu.Lock()
	select {
	case <-np.stopCh:
		np.mu.Unlock()
	default:
		close(np.stopCh)
		np.mu.Unlock()
		np.wg.Wait()
	}
}
