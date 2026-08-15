package main

import (
	"fmt"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	cfg := &cadence.CadenceConfig{
		Domain:         cadence.Domain,
		ServiceName:    cadence.WorkerServiceName,
		CadenceService: cadence.CadenceService,
		HostPort:       cadence.CadenceHostPort,
	}

	// 1. Setup transport & dispatcher for worker
	ch, err := tchannel.NewChannelTransport(tchannel.ServiceName(cfg.ServiceName))
	if err != nil {
		logger.Fatal("Failed to setup transport", zap.Error(err))
	}
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: cfg.ServiceName,
		Outbounds: yarpc.Outbounds{
			cfg.CadenceService: {Unary: ch.NewSingleOutbound(cfg.HostPort)},
		},
	})
	if err := dispatcher.Start(); err != nil {
		logger.Fatal("Failed to start dispatcher", zap.Error(err))
	}
	defer dispatcher.Stop()

	serviceClient := workflowserviceclient.New(dispatcher.ClientConfig(cfg.CadenceService))

	// 2. Start the Cadence worker (registers workflows & activities from internal/cadence)
	w, err := cadence.StartWorker(serviceClient, cfg, logger)
	if err != nil {
		logger.Fatal("Failed to start worker", zap.Error(err))
	}
	defer w.Stop()

	fmt.Println("Worker successfully started and polling Cadence...")

	select {}
}
