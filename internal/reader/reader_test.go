package reader

import (
	"os"
	"testing"

	"go.vxn.dev/xilt/internal/config"
	"go.vxn.dev/xilt/pkg/logger"
)

func TestReadAndBatch(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-logfile-*.log")
	if err != nil {
		t.Errorf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testData := []string{`10.0.1.100 user-identifier jack [25/Aug/2024:19:33:44 -0700] "GET /images/logo.png HTTP/1.0" 304 0 "https://cache.example.com" "Opera/90.0"`, `192.168.0.55 user-identifier ivan [20/Jul/2024:10:11:12 -0400] "POST /submit/form HTTP/1.1" 201 4321 "https://form.example.com" "Firefox/120.0"`, `172.16.254.1 user-identifier charlie [20/Jan/2024:18:45:22 -0700] "PUT /upload/file.txt HTTP/1.1" 201 9876 "https://dropbox.com" "PostmanRuntime/7.26"`}
	for _, line := range testData {
		_, err := tmpFile.WriteString(line + "\n")
		if err != nil {
			t.Errorf("Failed to write to temp file: %v", err)
		}
	}
	err = tmpFile.Close()
	if err != nil {
		t.Errorf("failed to close temp file: %v", err)
	}

	logger := logger.NewLogger(false)
	cfg := &config.Config{
		InputFilePath: tmpFile.Name(),
		BatchSize:     2,
	}
	r := NewReader(logger, cfg)

	batchChannel := make(chan []string)

	go func(channel chan []string) {
		for log := range channel {
			t.Logf("Value read: %v", log)
		}
	}(batchChannel)

	err = r.ReadAndBatch(batchChannel)
	if err != nil {
		t.Errorf("error reading and batching: %v", err)
	}

	close(batchChannel)
}

func TestReadAndBatch_InvalidFile(t *testing.T) {
	logger := logger.NewLogger(false)
	cfg := &config.Config{
		InputFilePath: "invalid file name",
		BatchSize:     2,
	}
	r := NewReader(logger, cfg)

	// Create a channel to receive batches
	batchChannel := make(chan []string)

	go func(channel chan []string) {
		for log := range channel {
			t.Logf("Value read: %v", log)
		}
	}(batchChannel)

	err := r.ReadAndBatch(batchChannel)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
