package cadence

import (
	"testing"
)

func TestCadenceConfig(t *testing.T) {
	cfg := NewCadenceConfig()
	if cfg == nil {
		t.Fatal("expected non-nil CadenceConfig")
	}
	if cfg.Domain != Domain {
		t.Errorf("expected domain %s, got %s", Domain, cfg.Domain)
	}
	if cfg.ServiceName != ClientServiceName {
		t.Errorf("expected service name %s, got %s", ClientServiceName, cfg.ServiceName)
	}
	if cfg.CadenceService != CadenceService {
		t.Errorf("expected cadence service %s, got %s", CadenceService, cfg.CadenceService)
	}
	if cfg.HostPort != CadenceHostPort {
		t.Errorf("expected host port %s, got %s", CadenceHostPort, cfg.HostPort)
	}
}

func TestNewCadenceClient(t *testing.T) {
	cfg := NewCadenceConfig()
	cli := NewCadenceClient(nil, cfg)
	if cli == nil {
		t.Fatal("expected non-nil Cadence client")
	}
}
