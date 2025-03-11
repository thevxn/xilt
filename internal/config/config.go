// Package config provides access to configurable parameters affecting the workings of the app (e.g. the path to the log file to parse, the path to the DB file to store the parsed logs in, the batch size or the memory usage limit).
package config

import (
	"flag"
	"fmt"
	"path/filepath"
)

type Config struct {
	BatchSize        int
	InputFilePath    string
	DBFilePath       string
	MaxMemoryUsageMB int
	AverageLogSizeMB float64
	Verbose          bool
	CreateIndexes    bool
}

const (
	defaultInputFilePath    = "access.log"
	defaultDbFilePath       = "logs.db"
	defaultBatchSize        = 5000
	defaultMaxMemUsageMB    = 100
	defaultAverageLogSizeMB = 0.001 // 1 KB
	defaultVerbose          = false
	defaultCreateIndexes    = false
)

func defineFlags(fs *flag.FlagSet, cfg *Config) {
	fs.IntVar(&cfg.BatchSize, "batchSize", defaultBatchSize, "Defines the batch size. Used for calculating the number of goroutines to spin up.")
	fs.IntVar(&cfg.MaxMemoryUsageMB, "maxMemUsage", defaultMaxMemUsageMB, "Defines the maximum allowed memory usage in Megabytes. Used for calculating the number of goroutines to spin up.")
	fs.Float64Var(&cfg.AverageLogSizeMB, "avgLogSize", defaultAverageLogSizeMB, "Defines the average size of one log in MB. Used for calculating the number of goroutines to spin up.")
	fs.BoolVar(&cfg.Verbose, "v", defaultVerbose, "Defines whether verbose mode should be used.")
	fs.BoolVar(&cfg.CreateIndexes, "i", defaultCreateIndexes, "Defines whether indexes should be created in the parsed logs' table.")
}

// Load attempts to parse flags and args and update the config with the parsed values. A default value is returned for each field if no value is specified in a flag/arg. If successful, it returns the updated config. Otherwise, an error is returned.
func Load(fs *flag.FlagSet, args []string) (*Config, error) {
	cfg := &Config{
		BatchSize:        defaultBatchSize,
		InputFilePath:    defaultInputFilePath,
		DBFilePath:       defaultDbFilePath,
		MaxMemoryUsageMB: defaultMaxMemUsageMB,
		AverageLogSizeMB: defaultAverageLogSizeMB,
		Verbose:          defaultVerbose,
		CreateIndexes:    defaultCreateIndexes,
	}

	defineFlags(fs, cfg)

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("error parsing flags: %v", err)
	}

	// Parse args
	parsedArgs := fs.Args()
	if len(parsedArgs) >= 1 {
		cleanPath := filepath.Clean(parsedArgs[0])
		if cleanPath != "." {
			cfg.InputFilePath = cleanPath
		} else {
			return nil, fmt.Errorf("the provided log file path is invalid")
		}
	}
	if len(parsedArgs) >= 2 {
		cleanPath := filepath.Clean(parsedArgs[1])
		if cleanPath != "." {
			cfg.DBFilePath = cleanPath
		} else {
			return nil, fmt.Errorf("the provided DB file path is invalid")
		}
	}

	return cfg, nil
}

// Validate checks that the currently configured values make sense for continuing with log processing.
// TODO: Validate file paths?
func (cfg *Config) Validate() error {
	if cfg.MaxMemoryUsageMB <= 0 {
		return fmt.Errorf("MaxMemoryUsageMB must be greater than 0. Got %d", cfg.MaxMemoryUsageMB)
	}
	if cfg.AverageLogSizeMB <= 0 {
		return fmt.Errorf("AverageLogSizeMB must be greater than 0. Got %f", cfg.AverageLogSizeMB)
	}
	if cfg.BatchSize <= 0 {
		return fmt.Errorf("BatchSize must be greater than 0. Got %d", cfg.BatchSize)
	}

	return nil
}
