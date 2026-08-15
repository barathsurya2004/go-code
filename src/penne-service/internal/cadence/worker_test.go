package cadence

import (
	"testing"

	"go.uber.org/zap"
)

func TestRegisterWorkflowsAndActivities(t *testing.T) {
	// Ensures workflow and activity registration logic executes successfully
	RegisterWorkflowsAndActivities()
}

func TestStartWorker_NilClientError(t *testing.T) {
	logger := zap.NewNop()
	cfg := NewCadenceConfig()
	_, err := StartWorker(nil, cfg, logger)
	if err == nil {
		t.Error("expected error when serviceClient is nil, got nil")
	}
}
