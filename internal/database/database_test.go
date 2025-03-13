package database

import (
	"sync"
	"testing"

	"go.vxn.dev/xilt/internal/config"
	"go.vxn.dev/xilt/internal/parser"
)

type mockLogger struct{}

func (m *mockLogger) Println(v ...any)               {}
func (m *mockLogger) Printf(format string, v ...any) {}
func (m *mockLogger) Debug(v ...any)                 {}
func (m *mockLogger) Debugf(format string, v ...any) {}

func TestNewDB(t *testing.T) {
	config := &config.Config{
		Verbose: false,
	}

	logger := &mockLogger{}

	db := NewDB(logger, config)

	if db.conn != nil {
		t.Errorf("expected conn to be nil, got %v", db.conn)
	}

	if db.config != config {
		t.Errorf("expected config %v, got %v", config, db.config)
	}

	if db.logger != logger {
		t.Errorf("expected logger %v, got %v", logger, db.logger)
	}

}

func TestDB_Init(t *testing.T) {
	config := &config.Config{
		Verbose:    false,
		DBFilePath: ":memory:?cache=shared",
	}

	logger := &mockLogger{}

	db := NewDB(logger, config)

	err := db.Init()
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}
	defer db.Close()

	if db.conn == nil {
		t.Error("expected db.conn to be non-nil after Init")
	}

	// Verify that the log table was created
	rows, err := db.conn.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='logs';")
	if err != nil {
		t.Errorf("failed to query sqlite_master: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("expected 'logs' table to exist, but it does not")
	}

}

func TestDB_InitInvalidFilePath(t *testing.T) {
	config := &config.Config{
		Verbose:    false,
		DBFilePath: "invalidpath//....invalidfilename..../test",
	}

	logger := &mockLogger{}

	db := NewDB(logger, config)
	defer db.Close()

	err := db.Init()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDB_Close(t *testing.T) {
	config := &config.Config{
		Verbose:    false,
		DBFilePath: ":memory:?cache=shared",
	}

	logger := &mockLogger{}

	db := NewDB(logger, config)
	if err := db.Close(); err != nil {
		t.Errorf("expected nil, got error: %v", err)
	}

}

func TestDB_CreateIndexes(t *testing.T) {
	config := &config.Config{
		Verbose:       false,
		DBFilePath:    ":memory:?cache=shared",
		CreateIndexes: true,
	}

	logger := &mockLogger{}

	db := NewDB(logger, config)
	defer db.Close()

	if err := db.Init(); err != nil {
		t.Errorf("Init failed: %v", err)
	}

	if err := db.CreateIndexes(); err != nil {
		t.Errorf("index creation failed: %v", err)
	}

	rows, err := db.conn.Query("select name FROM sqlite_master WHERE type='index'")
	if err != nil {
		t.Errorf("error querying indexes: %v", err)
	}
	defer rows.Close()

	indexes := make([]string, 0)

	for rows.Next() {
		var idx string

		if err := rows.Scan(&idx); err != nil {
			t.Errorf("error scanning index name: %v", err)
		}
		indexes = append(indexes, idx)
	}

	if len(indexes) < 1 {
		t.Error("no indexes created, expected indexes to be created")
	}

}

func TestDB_CreateIndexesDisabled(t *testing.T) {
	config := &config.Config{
		Verbose:       false,
		DBFilePath:    ":memory:?cache=shared",
		CreateIndexes: false,
	}

	logger := &mockLogger{}

	db := NewDB(logger, config)
	defer db.Close()

	if err := db.Init(); err != nil {
		t.Errorf("Init failed: %v", err)
	}

	if err := db.CreateIndexes(); err != nil {
		t.Errorf("expected nil to be returned, got: %v", err)
	}

}

func TestDB_InsertBatch(t *testing.T) {
	config := &config.Config{
		Verbose:    false,
		DBFilePath: ":memory:?cache=shared",
	}

	logger := &mockLogger{}

	db := NewDB(logger, config)
	defer db.Close()

	if err := db.Init(); err != nil {
		t.Errorf("Init failed: %v", err)
	}

	parsedLogs := []parser.Log{{
		IP:           "127.0.0.1",
		Identity:     "user-identifier",
		User:         "frank",
		Time:         "10/Oct/2000:13:55:36 -0700",
		TimestampUTC: "2000-10-10T20:55:36Z",
		Method:       "GET",
		Route:        "/apache_pb.gif",
		Params:       "param1=test",
		ResponseCode: 200,
		BytesSent:    2326,
		Referer:      "referrer",
		Agent:        "agent",
	}, {
		IP:           "127.0.0.1",
		Identity:     "user-identifier",
		User:         "frank",
		Time:         "10/Oct/2000:13:55:36 -0700",
		TimestampUTC: "2000-10-10T20:55:36Z",
		Method:       "GET",
		Route:        "/apache_pb.gif",
		Params:       "param1=test",
		ResponseCode: 200,
		BytesSent:    2326,
		Referer:      "referrer",
		Agent:        "agent",
	}}

	parsedLogChan := make(chan []parser.Log)

	var wg sync.WaitGroup
	wg.Add(1)

	go db.InsertBatch(parsedLogChan, &wg)

	parsedLogChan <- parsedLogs
	close(parsedLogChan)

	wg.Wait()

	rows, err := db.conn.Query("SELECT id FROM logs;")
	if err != nil {
		t.Errorf("error querying logs: %v", err)
	}

	logIDs := make([]int8, 0)

	for rows.Next() {
		var logID int8

		if err := rows.Scan(&logID); err != nil {
			t.Error("error scanning log rows")
		}

		logIDs = append(logIDs, logID)
	}

	if len(logIDs) < 2 {
		t.Errorf("expected 2 log IDs to be returned from DB, got %d ID(s) instead", len(logIDs))
	}
}
