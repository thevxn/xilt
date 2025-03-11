package logger

import (
	"bytes"
	"log"
	"testing"
)

func TestLogger_PrintLn(t *testing.T) {

	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Remnove time, date and source file information from the log
	log.SetFlags(0)
	defer func() {
		// Reset log output
		log.SetOutput(nil)
	}()

	logger := NewLogger(false)

	logger.Println("test message")

	expected := "test message\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestLogger_Printf(t *testing.T) {

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(nil)
	}()

	logger := NewLogger(false)

	logger.Printf("test %s", "message")

	expected := "test message\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestLogger_Debug(t *testing.T) {

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	defer func() {
		log.SetOutput(nil)
	}()

	logger := NewLogger(true)

	logger.Debug("debug message")

	expected := "debug message\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}

	logger = NewLogger(false)
	buf.Reset()

	logger.Debug("debug message")

	if buf.String() != "" {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestLogger_Debugf(t *testing.T) {

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	defer func() {
		log.SetOutput(nil)
	}()

	logger := NewLogger(true)

	logger.Debugf("debug %s", "message")

	expected := "debug message\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}

	logger = NewLogger(false)
	buf.Reset()
	logger.Debugf("debug %s", "message")

	if buf.String() != "" {
		t.Errorf("expected no output, got %q", buf.String())
	}
}
