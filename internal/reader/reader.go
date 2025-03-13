// Package reader provides functionality for reading from a log file and pushing raw logs to a batch channel to be processed.
package reader

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"go.vxn.dev/xilt/internal/config"
	"go.vxn.dev/xilt/pkg/logger"
)

type reader struct {
	logger logger.Logger
	cfg    *config.Config
}

// NewReader returns a new instance of the Reader struct.
func NewReader(l logger.Logger, c *config.Config) *reader {
	return &reader{
		logger: l,
		cfg:    c,
	}
}

// ReadAndBatch reads from the file configured in the Reader struct's config and pushes raw logs into a batch channel for further processing.
func (r *reader) ReadAndBatch(batchChannel chan<- []string) error {
	file, err := os.Open(r.cfg.InputFilePath)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return fmt.Errorf("error: file '%s' does not exist", r.cfg.InputFilePath)
		case errors.Is(err, fs.ErrPermission):
			return fmt.Errorf("error: insufficient permissions to read file '%s'", r.cfg.InputFilePath)
		default:
			return fmt.Errorf("error opening file '%s': %v", r.cfg.InputFilePath, err)
		}
	}

	r.logger.Debug("log file opened...")

	defer func() {
		file.Close()
		r.logger.Debug("log file closed...")
	}()

	scanner := bufio.NewScanner(file)

	batch := make([]string, 0, r.cfg.BatchSize)

	r.logger.Println("beginning reading from log file and the parsing process...")

	// Iterate over the lines in the log file and push them into the batch slice until the batch size or EOF is reached
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			batch = append(batch, line)
			if len(batch) == r.cfg.BatchSize {
				batchChannel <- batch
				batch = make([]string, 0, r.cfg.BatchSize)
			}
		}
	}

	// Push the remaining logs if any
	if len(batch) > 0 {
		batchChannel <- batch
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file '%s': %v", r.cfg.InputFilePath, err)
	}
	return nil
}
