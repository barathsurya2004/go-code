package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	origPort := os.Getenv("PORT")
	origDB := os.Getenv("DATABASE_URL")
	defer func() {
		os.Setenv("PORT", origPort)
		os.Setenv("DATABASE_URL", origDB)
	}()

	t.Run("default config values", func(t *testing.T) {
		os.Unsetenv("PORT")
		os.Unsetenv("DATABASE_URL")

		cfg := NewConfig()
		assert.Equal(t, "8082", cfg.Port)
		assert.Contains(t, cfg.DatabaseURL, "pulse_app")
	})

	t.Run("custom env config values", func(t *testing.T) {
		os.Setenv("PORT", "9999")
		os.Setenv("DATABASE_URL", "postgres://test/db")

		cfg := NewConfig()
		assert.Equal(t, "9999", cfg.Port)
		assert.Equal(t, "postgres://test/db", cfg.DatabaseURL)
	})
}
