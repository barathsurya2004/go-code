package cadence

import (
	"os"
)

// Cadence system constants matching system configuration
const (
	Domain            = "nouri-lifecycle-workers"
	CadenceService    = "cadence-frontend"
	CadenceHostPort   = "localhost:7933"
	ClientServiceName = "nouri-nutrition-client"
	WorkerServiceName = "nouri-nutrition-worker"
	TaskListName      = "nouri-tasklist"
)

// CadenceConfig holds configuration for the Cadence client and worker connection.
type CadenceConfig struct {
	Domain         string
	ServiceName    string
	CadenceService string
	HostPort       string
	TaskListName   string
}

// NewCadenceConfig provides default configuration with environment overrides.
func NewCadenceConfig() *CadenceConfig {
	hostPort := os.Getenv("CADENCE_HOST_PORT")
	if hostPort == "" {
		hostPort = os.Getenv("CADENCE_ADDR")
	}
	if hostPort == "" {
		hostPort = CadenceHostPort
	}

	domain := os.Getenv("CADENCE_DOMAIN")
	if domain == "" {
		domain = Domain
	}

	return &CadenceConfig{
		Domain:         domain,
		ServiceName:    ClientServiceName,
		CadenceService: CadenceService,
		HostPort:       hostPort,
		TaskListName:   TaskListName,
	}
}
