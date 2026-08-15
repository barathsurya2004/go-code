package main

import (
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
)

func TestCreateDispatcherAndServiceClient(t *testing.T) {
	cfg := &cadence.CadenceConfig{
		Domain:         cadence.Domain,
		ServiceName:    cadence.WorkerServiceName,
		CadenceService: cadence.CadenceService,
		HostPort:       cadence.CadenceHostPort,
	}

	dispatcher, err := createDispatcher(cfg)
	if err != nil {
		t.Fatalf("expected no error creating dispatcher, got %v", err)
	}
	if dispatcher == nil {
		t.Fatal("expected non-nil dispatcher")
	}

	svcClient := initServiceClient(dispatcher, cfg)
	if svcClient == nil {
		t.Fatal("expected non-nil serviceClient")
	}

	if err := dispatcher.Stop(); err != nil {
		t.Errorf("expected clean dispatcher stop, got %v", err)
	}
}
