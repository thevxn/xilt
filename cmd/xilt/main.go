package main

import (
	"flag"

	"log"
	"math"
	"os"
	"sync"
	"time"

	"go.vxn.dev/xilt/internal/config"
	"go.vxn.dev/xilt/internal/database"
	"go.vxn.dev/xilt/internal/parser"
	"go.vxn.dev/xilt/internal/reader"
	"go.vxn.dev/xilt/pkg/logger"
)

const (
	reservedRoutines = 2
)

func getRoutineCount(cfg *config.Config) int {
	// Max number of routines is equal to the maximum memory usage limit / the average size of a batch, taking into account the configured average log size and  batch size
	// reservedRoutines are subtracted because one write routine is running concurrently and at the same time another batch is being put together by reading from the log file, which also has to be taken into account
	return int(math.Max(1, math.Floor(float64(cfg.MaxMemoryUsageMB)/(cfg.AverageLogSizeMB*float64(cfg.BatchSize)))-reservedRoutines))
}

func main() {
	// Start timer
	start := time.Now()

	// Load config
	cfg, err := config.Load(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalln("error loading config:", err)
	}

	l := logger.NewLogger(cfg.Verbose)

	l.Debug("config loaded...")

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

	parser, err := parser.NewParser(l, nil)
	if err != nil {
		l.Println("error creating parser: ", err)
		return
	}

	// Spin up routines to parse logs
	routineCount := getRoutineCount(cfg)

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

	// Instantiate a reader which will read from the configured input file and push raw logs into the batchChannel for parsing
	reader := reader.NewReader(l, cfg)

	if err := reader.ReadAndBatch(batchChannel); err != nil {
		l.Printf("error while reading from log file and batching: %v", err)
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
