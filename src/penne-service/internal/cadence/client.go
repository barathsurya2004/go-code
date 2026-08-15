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

// Cadence system constants matching system configuration
const (
	Domain            = "penne-service"
	CadenceService    = "cadence-frontend"
	CadenceHostPort   = "localhost:7933"
	ClientServiceName = "penne-service-client"
	WorkerServiceName = "penne-service-worker"
	TaskListName      = "onboarding-task-list"
)

// CadenceConfig holds configuration for the Cadence client connection.
type CadenceConfig struct {
	Domain         string
	ServiceName    string
	CadenceService string
	HostPort       string
}

// NewCadenceConfig provides default configuration.
func NewCadenceConfig() *CadenceConfig {
	return &CadenceConfig{
		Domain:         Domain,
		ServiceName:    ClientServiceName,
		CadenceService: CadenceService,
		HostPort:       CadenceHostPort,
	}
}

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

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping YARPC dispatcher for Cadence client...")
			return dispatcher.Stop()
		},
	})

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

// Module exports the Cadence dependency injection module for Uber Fx (used by HTTP server).
var Module = fx.Module(
	"cadence",
	fx.Provide(
		NewCadenceConfig,
		NewCadenceServiceClient,
		NewCadenceClient,
	),
)
