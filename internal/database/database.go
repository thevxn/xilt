// Package database provides methods for interacting with the database to store parsed logs.
package database

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"go.vxn.dev/xilt/internal/config"
	"go.vxn.dev/xilt/internal/parser"
	"go.vxn.dev/xilt/pkg/logger"
)

const (
	createLogTableScript = `CREATE TABLE "logs" ("ID" INTEGER NOT NULL, "IP"	TEXT, "Identity" TEXT,"UserID"	TEXT, "Time"	TEXT, "TimestampUTC" TEXT , "Method"	TEXT, "Route"	TEXT, "Params"	TEXT,  "ResponseCode"	INTEGER, "BytesSent"	INTEGER, "Referer" TEXT, "Agent" TEXT, PRIMARY KEY("id" AUTOINCREMENT));`
	insertLogStatement   = "INSERT INTO logs (IP, Identity, UserID, Time, TimestampUTC, Method, Route, Params, ResponseCode, BytesSent, Referer, Agent) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	createIndexesScript  = `
	CREATE INDEX idx_logs_ip ON logs(IP);
	CREATE INDEX idx_logs_ts ON logs(TimestampUTC);
	CREATE INDEX idx_logs_method ON logs(Method);
	CREATE INDEX idx_logs_route ON logs(Route);
	CREATE INDEX idx_logs_referer ON logs(Referer);
	CREATE INDEX idx_logs_ts_ip ON logs(TimestampUTC, IP);
	`
)

type db struct {
	conn   *sql.DB
	logger logger.Logger
	config *config.Config
}

type Database interface {
	Init(cfg *config.Config) error
	Close() error
	InsertBatch(id int, parsedLogChan <-chan []parser.Log, wg *sync.WaitGroup)
}

// NewDB returns a new instance of a DB struct initialized with the provided logger and config.
func NewDB(l logger.Logger, c *config.Config) *db {
	return &db{
		logger: l,
		config: c,
	}
}

// Init initializes the DB struct. It attempts to connect to the database, configure it to better optimize write performance and creates a table for storage of parsed logs.
func (d *db) Init() error {
	// Connect to DB
	db, err := sql.Open("sqlite3", "file:"+d.config.DBFilePath)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}

	d.conn = db

	// A wrapper to close the DB connection after an error and also handle a possible DB closing error
	// Used for the SQLite queries below
	handleFailure := func(originalErr error) error {
		if closeErr := d.conn.Close(); closeErr != nil {
			d.logger.Println("failed to close DB after error: ", closeErr)
		}
		return originalErr
	}

	// Optimize SQLite for writes
	_, err = d.conn.Exec("PRAGMA synchronous = OFF;")
	if err != nil {
		return handleFailure(fmt.Errorf("failed to set PRAGMA synchronous: %w", err))
	}

	// WAL mode has issues with saving big (50k+) batch sizes and creating indexes on tables containing large amounts of logs, therefore the default rollback journal method is used instead.
	// https://github.com/Tencent/wcdb/issues/243
	// _, err = d.conn.Exec("PRAGMA journal_mode = WAL;")
	// if err != nil {
	// 	return handleFailure(fmt.Errorf("failed to set journal_mode: %w", err))
	// }

	// Create a new table to store parsed logs
	_, err = d.conn.Exec(createLogTableScript)
	if err != nil {
		return handleFailure(fmt.Errorf("failed to create log table: %w", err))
	}

	d.logger.Debug("DB initialized...")

	return nil
}

// Close closes the connection of the DB struct if it is not nil.
func (d *db) Close() error {
	if d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

// InsertBatch inserts a batch of logs from an output channel to the database. It is optimized for concurrent usage in goroutines.
func (d *db) InsertBatch(parsedLogChan <-chan []parser.Log, wg *sync.WaitGroup) {
	defer wg.Done()

	for parsedLogBatch := range parsedLogChan {
		d.logger.Debug("write routine beginning insert")
		tx, err := d.conn.Begin()
		if err != nil {
			d.logger.Printf("write routine failed to start transaction: %v", err)
			continue
		}

		stmt, err := tx.Prepare(insertLogStatement)
		if err != nil {
			d.logger.Printf("write routine failed to prepare statement: %v", err)
			if err = tx.Rollback(); err != nil {
				d.logger.Printf("write routine failed to roll back transaction: %v", err)
			}
			continue
		}

		for _, parsedLog := range parsedLogBatch {
			_, err := stmt.Exec(parsedLog.IP, parsedLog.Identity, parsedLog.User, parsedLog.Time, parsedLog.TimestampUTC, parsedLog.Method, parsedLog.Route, parsedLog.Params, parsedLog.ResponseCode, parsedLog.BytesSent, parsedLog.Referer, parsedLog.Agent)
			if err != nil {
				d.logger.Printf("write routine failed to insert: %v", err)
				if err = tx.Rollback(); err != nil {
					d.logger.Printf("write routine failed to roll back transaction: %v", err)
				}
				break
			}
		}

		if err := stmt.Close(); err != nil {
			d.logger.Println("write routine failed to close statement: %v", err)
		}

		if err := tx.Commit(); err != nil {
			d.logger.Println("write routine failed to commit transaction: %v", err)
		} else {
			d.logger.Debugf("write routine successfully inserted batch of %d logs", len(parsedLogBatch))
		}
	}
}

// CreateIndexes creates indexes on the log table if enabled in the config provided to the DB struct.
func (d *db) CreateIndexes() error {
	if d.config.CreateIndexes {
		d.logger.Println("creating table indexes...")
		if _, err := d.conn.Exec(createIndexesScript); err != nil {
			return err
		}
		d.logger.Println("table indexes created...")
		return nil
	}
	return nil
}
