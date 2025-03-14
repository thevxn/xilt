package config

import (
	"flag"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	args := []string{}

	cfg, err := Load(fs, args)
	if err != nil {
		t.Errorf("error loading config: %v", err)
	}

	expected := &Config{
		BatchSize:        defaultBatchSize,
		InputFilePath:    defaultInputFilePath,
		DBFilePath:       defaultDbFilePath,
		MaxMemoryUsageMB: defaultMaxMemUsageMB,
		AverageLogSizeMB: defaultAverageLogSizeMB,
		Verbose:          defaultVerbose,
		CreateIndexes:    defaultCreateIndexes,
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("expected %+v, got %+v", expected, cfg)
	}
}

func TestLoad_Flags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Set flags to be tested
	args := []string{
		"-batchSize=1234",
		"-maxMemUsage=250",
		"-avgLogSize=750",
		"-v",
		"-i",
	}

	cfg, err := Load(fs, args)
	if err != nil {
		t.Errorf("error loading config: %v", err)
	}

	expected := &Config{
		BatchSize:        1234,
		InputFilePath:    defaultInputFilePath,
		DBFilePath:       defaultDbFilePath,
		MaxMemoryUsageMB: 250,
		AverageLogSizeMB: 750,
		Verbose:          true,
		CreateIndexes:    true,
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("expected %+v, got %+v", expected, cfg)
	}

}

func TestLoad_TwoArgs(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Set args to be tested
	args := []string{"test.log", "test.db"}

	cfg, err := Load(fs, args)
	if err != nil {
		t.Errorf("error loading config: %v", err)
	}

	expected := &Config{
		BatchSize:        defaultBatchSize,
		InputFilePath:    filepath.Clean("test.log"),
		DBFilePath:       filepath.Clean("test.db"),
		MaxMemoryUsageMB: defaultMaxMemUsageMB,
		AverageLogSizeMB: defaultAverageLogSizeMB,
		Verbose:          defaultVerbose,
		CreateIndexes:    defaultCreateIndexes,
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("expected %+v, got %+v", expected, cfg)
	}

}

func TestLoad_OneArg(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Set args to be tested
	args := []string{"test.log"}

	cfg, err := Load(fs, args)
	if err != nil {
		t.Errorf("error loading config: %v", err)
	}

	expected := &Config{
		BatchSize:        defaultBatchSize,
		InputFilePath:    filepath.Clean("test.log"),
		DBFilePath:       defaultDbFilePath,
		MaxMemoryUsageMB: defaultMaxMemUsageMB,
		AverageLogSizeMB: defaultAverageLogSizeMB,
		Verbose:          defaultVerbose,
		CreateIndexes:    defaultCreateIndexes,
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("expected %+v, got %+v", expected, cfg)
	}

}

func TestLoad_InvalidLogFilePath(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Set args to be tested
	args := []string{"."}

	_, err := Load(fs, args)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

}

func TestLoad_InvalidDBFilePath(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Set args to be tested
	args := []string{"test.log", "."}

	_, err := Load(fs, args)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

}

// TODO: test invalid log file & DB file formats?

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			cfg: Config{
				BatchSize:        100,
				MaxMemoryUsageMB: 100,
				AverageLogSizeMB: 0.001,
			},
			expectError: false,
		},
		{
			name: "invalid MaxMemoryUsageMB",
			cfg: Config{
				BatchSize:        100,
				MaxMemoryUsageMB: 0,
				AverageLogSizeMB: 0.001,
			},
			expectError: true,
			errorMsg:    "MaxMemoryUsageMB must be greater than 0. Got 0",
		},
		{
			name: "invalid AverageLogSizeMB",
			cfg: Config{
				BatchSize:        100,
				MaxMemoryUsageMB: 100,
				AverageLogSizeMB: 0,
			},
			expectError: true,
			errorMsg:    "AverageLogSizeMB must be greater than 0. Got 0.000000",
		},
		{
			name: "invalid BatchSize",
			cfg: Config{
				BatchSize:        0,
				MaxMemoryUsageMB: 100,
				AverageLogSizeMB: 0.001,
			},
			expectError: true,
			errorMsg:    "BatchSize must be greater than 0. Got 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError %v", err, tt.expectError)
			}
			if tt.expectError && err.Error() != tt.errorMsg {
				t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

func TestLoad_InvalidFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	args := []string{
		"-invalidFlag",
	}

	if _, err := Load(fs, args); err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	args := []string{"-batchSize=0"}

	_, err := Load(fs, args)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

}
