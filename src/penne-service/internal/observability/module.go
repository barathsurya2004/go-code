package observability

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func provideUptimeTracker(log *zap.Logger) *UptimeTracker {
	return NewUptimeTracker("", 10*time.Second, log)
}

func provideNetworkProbe(log *zap.Logger) *NetworkProbe {
	return NewNetworkProbe("", 15*time.Second, log)
}

func registerObservabilityLifecycle(lc fx.Lifecycle, ut *UptimeTracker, np *NetworkProbe, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Initializing server uptime and network observability...")
			ut.CheckLastOutage(time.Now())
			ut.StartHeartbeat(context.Background())
			np.Start(context.Background())
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Shutting down observability probes gracefully...")
			np.Stop()
			ut.Stop()
			return nil
		},
	})
}

var Module = fx.Module(
	"observability",
	fx.Provide(
		provideUptimeTracker,
		provideNetworkProbe,
	),
	fx.Invoke(
		registerObservabilityLifecycle,
	),
)
