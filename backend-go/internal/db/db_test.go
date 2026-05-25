package db

import (
	"strings"
	"testing"
)

func TestNewRejectsMalformedDSN(t *testing.T) {
	_, err := New(t.Context(), "postgres://bad host:5432/db")
	if err == nil {
		t.Fatal("expected parse error for malformed DSN")
	}
	if !strings.Contains(err.Error(), "parse db dsn") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

func TestNewAppliesPoolDefaults(t *testing.T) {
	pool := integrationPool(t)
	cfg := pool.Config()
	if cfg.MaxConns != 10 {
		t.Errorf("MaxConns: want 10, got %d", cfg.MaxConns)
	}
	if cfg.MaxConnIdleTime.Minutes() != 5 {
		t.Errorf("MaxConnIdleTime: want 5m, got %s", cfg.MaxConnIdleTime)
	}
	if cfg.HealthCheckPeriod.Minutes() != 1 {
		t.Errorf("HealthCheckPeriod: want 1m, got %s", cfg.HealthCheckPeriod)
	}
}
