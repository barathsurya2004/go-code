package main

import (
	"os"
	"time"
)

type Config struct {
	Port               string
	PenneServiceTarget string
	PenneHTTPTarget    string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
}

func NewConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	penneTarget := os.Getenv("PENNE_SERVICE_TARGET")
	if penneTarget == "" {
		penneTarget = "127.0.0.1:50051"
	}

	penneHTTPTarget := os.Getenv("PENNE_HTTP_TARGET")
	if penneHTTPTarget == "" {
		penneHTTPTarget = "http://127.0.0.1:8080"
	}

	return &Config{
		Port:               port,
		PenneServiceTarget: penneTarget,
		PenneHTTPTarget:    penneHTTPTarget,
		ReadTimeout:        15 * time.Second,
		WriteTimeout:       15 * time.Second,
	}
}
