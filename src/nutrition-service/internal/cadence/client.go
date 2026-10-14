package cadence

import (
	"context"

	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/client"
	"go.uber.org/fx"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

// NewCadenceServiceClient creates the low-level YARPC RPC client interface for Cadence.
func NewCadenceServiceClient(lc fx.Lifecycle, cfg *CadenceConfig, logger *zap.Logger) (workflowserviceclient.Interface, error) {
	ch, err := tchannel.NewChannelTransport(tchannel.ServiceName(cfg.ServiceName))
	if err != nil {
		logger.Error("Failed to create TChannel transport for Cadence client", zap.Error(err))
		return nil, err
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: cfg.ServiceName,
		Outbounds: yarpc.Outbounds{
			cfg.CadenceService: {Unary: ch.NewSingleOutbound(cfg.HostPort)},
		},
	})

	if err := dispatcher.Start(); err != nil {
		logger.Error("Failed to start YARPC dispatcher for Cadence client", zap.Error(err))
		return nil, err
	}

	if lc != nil {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				logger.Info("Stopping YARPC dispatcher for Cadence client...")
				return dispatcher.Stop()
			},
		})
	}

	return workflowserviceclient.New(dispatcher.ClientConfig(cfg.CadenceService)), nil
}

// NewCadenceClient initializes the high-level Cadence workflow client.
func NewCadenceClient(serviceClient workflowserviceclient.Interface, cfg *CadenceConfig) client.Client {
	return client.NewClient(
		serviceClient,
		cfg.Domain,
		&client.Options{},
	)
}
