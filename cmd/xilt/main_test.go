package main

import (
	"testing"

	"go.vxn.dev/xilt/internal/config"
)

func TestGetRoutineCount(t *testing.T) {

	cfg := &config.Config{
		MaxMemoryUsageMB: 100,
		AverageLogSizeMB: 0.001,
		BatchSize:        5000,
	}

	expected := 18
	actual := getRoutineCount(cfg)

	if expected != actual {
		t.Errorf("expected %d, got %d", expected, actual)
	}
}
