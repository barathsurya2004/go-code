package main

import (
	"fmt"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

func createDispatcher(cfg *cadence.CadenceConfig) (*yarpc.Dispatcher, error) {
	ch, err := tchannel.NewChannelTransport(tchannel.ServiceName(cfg.ServiceName))
	if err != nil {
		return nil, err
	}
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: cfg.ServiceName,
		Outbounds: yarpc.Outbounds{
			cfg.CadenceService: {Unary: ch.NewSingleOutbound(cfg.HostPort)},
		},
	})
	if err := dispatcher.Start(); err != nil {
		return nil, err
	}
	return dispatcher, nil
}

func initServiceClient(dispatcher *yarpc.Dispatcher, cfg *cadence.CadenceConfig) workflowserviceclient.Interface {
	return workflowserviceclient.New(dispatcher.ClientConfig(cfg.CadenceService))
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg := cadence.NewCadenceConfig()
	cfg.ServiceName = cadence.WorkerServiceName

	dispatcher, err := createDispatcher(cfg)
	if err != nil {
		logger.Fatal("Failed to create dispatcher", zap.Error(err))
	}
	defer dispatcher.Stop()

	serviceClient := initServiceClient(dispatcher, cfg)
	w, err := cadence.StartWorker(serviceClient, cfg, logger)
	if err != nil {
		logger.Fatal("Failed to start worker", zap.Error(err))
	}
	defer w.Stop()

	fmt.Println("Worker successfully started and polling Cadence...")
	select {}
}
