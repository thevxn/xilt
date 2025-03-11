package main

import (
	"bufio"
	"errors"
	"flag"
	"io/fs"

	"log"
	"math"
	"os"
	"sync"
	"time"

	"go.vxn.dev/xilt/internal/config"
	"go.vxn.dev/xilt/internal/database"
	"go.vxn.dev/xilt/internal/parser"
	"go.vxn.dev/xilt/pkg/logger"
)

func main() {
	// Start timer
	start := time.Now()

	// Load config
	cfg, err := config.Load(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalln("error parsing flags and arguments:", err)
	}

	l := logger.NewLogger(cfg.Verbose)

	l.Debug("config loaded...")

	// Validate that configured values make sense
	if err := cfg.Validate(); err != nil {
		log.Fatalln("invalid configuration: ", err)
	}

	l.Debug("config is valid...")

	db := database.NewDB(l, cfg)
	if err := db.Init(); err != nil {
		log.Fatalln("error initializing database: ", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			l.Println("error closing database:", err)
		}
		l.Debug("database closed...")
	}()

	// Logs are distributed to parsing routines in batches via this channel
	batchChannel := make(chan []string)
	// Parsing routines distribute batches of parsed logs to the single writing routine via this channel
	parsedLogChannel := make(chan []parser.Log)

	var batchWg sync.WaitGroup
	var insertWg sync.WaitGroup

	// Spin up routines to parse logs
	// Max number of routines is equal to the maximum memory usage limit / the average size of a batch, taking into account the configured average log size and  batch size
	// -2 = one write routine is running concurrently and at the same time another batch is being put together by reading from the log file, which also has to be taken into account
	routineCount := int(math.Max(1, math.Floor(float64(cfg.MaxMemoryUsageMB)/(cfg.AverageLogSizeMB*float64(cfg.BatchSize)))-2))

	parser, err := parser.NewParser(l, nil)
	if err != nil {
		l.Println("error creating parser: ", err)
		return
	}

	l.Debugf("spinning up %d log parsing routines...", routineCount)

	for i := range routineCount {
		batchWg.Add(1)
		go parser.ParseBatch(i, batchChannel, parsedLogChannel, &batchWg)
	}

	l.Debug("log parsing routines spawned...")

	// There is only one DB write routine due to SQLite's single-writer model
	insertWg.Add(1)
	go db.InsertBatch(parsedLogChannel, &insertWg)

	l.Debug("batch insert routine spawned...")

	file, err := os.Open(cfg.InputFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			l.Printf("error: file '%s' does not exist", cfg.InputFilePath)
			return
		} else if errors.Is(err, fs.ErrPermission) {
			l.Printf("error: insufficient permissions to read file '%s'", cfg.InputFilePath)
			return
		} else {
			l.Printf("error opening file '%s': %v", cfg.InputFilePath, err)
			return
		}
	}

	l.Debug("log file opened...")

	defer func() {
		file.Close()
		l.Debug("log file closed...")
	}()

	scanner := bufio.NewScanner(file)

	batch := make([]string, 0, cfg.BatchSize)

	l.Println("beginning reading from log file and the parsing process...")

	// Iterate over the lines in the log file and push them into the batch slice until the batch size or EOF is reached
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			batch = append(batch, line)
			if len(batch) == cfg.BatchSize {
				batchChannel <- batch
				batch = make([]string, 0, cfg.BatchSize)
			}
		}
	}

	// Push the remaining logs if any
	if len(batch) > 0 {
		batchChannel <- batch
	}

	if err := scanner.Err(); err != nil {
		l.Println("error reading from file:", err)
		return
	}

	close(batchChannel)
	batchWg.Wait()

	close(parsedLogChannel)
	insertWg.Wait()

	if err := db.CreateIndexes(); err != nil {
		l.Println("error creating table indexes: ", err)
	}

	// Stop timer & print duration
	end := time.Now()

	l.Println("log parsing finished")
	l.Printf("elapsed time: %s", end.Sub(start))
}
