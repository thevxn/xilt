package parser

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
)

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Println(v ...any) {
	logMsg := fmt.Sprintln(v...)
	m.logs = append(m.logs, strings.TrimSpace(logMsg))
}
func (m *mockLogger) Printf(format string, v ...any) {
	logMsg := fmt.Sprintln(v...)
	m.logs = append(m.logs, strings.TrimSpace(logMsg))
}
func (m *mockLogger) Debug(v ...any) {
	logMsg := fmt.Sprintln(v...)
	m.logs = append(m.logs, strings.TrimSpace(logMsg))
}
func (m *mockLogger) Debugf(format string, v ...any) {
	logMsg := fmt.Sprintln(v...)
	m.logs = append(m.logs, strings.TrimSpace(logMsg))
}

const (
	validCombinedLog = `127.0.0.1 user-identifier frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif?param1=test HTTP/1.0" 200 2326 "referrer" "agent"`
	validCommonLog   = `127.0.0.1 user-identifier frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif?param1=test HTTP/1.0" 200 2326`
)

func TestNewParserWithRegex(t *testing.T) {
	compiledDefaultRegex, err := regexp.Compile(defaultRegex)
	if err != nil {
		t.Errorf("error compiling regex: %v", err)
	}
	expected := &parser{logger: &mockLogger{}, regex: compiledDefaultRegex}
	actual, err := NewParser(&mockLogger{}, &defaultRegex)
	if err != nil {
		t.Errorf("error creating new parser: %v", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected: %v, got: %v", expected, actual)
	}
}

func TestNewParserWithoutRegex(t *testing.T) {
	compiledDefaultRegex, err := regexp.Compile(defaultRegex)
	if err != nil {
		t.Errorf("error compiling regex: %v", err)
	}

	expected := &parser{logger: &mockLogger{}, regex: compiledDefaultRegex}

	actual, err := NewParser(&mockLogger{}, nil)
	if err != nil {
		t.Errorf("error creating new parser: %v", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected: %v, got: %v", expected, actual)
	}
}

func TestNewParserWithInvalidRegex(t *testing.T) {
	invalidRegex := "(abc"

	_, err := NewParser(&mockLogger{}, &invalidRegex)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

}

func TestParser_ParseLogValidCombined(t *testing.T) {

	p, err := NewParser(&mockLogger{}, &defaultRegex)
	if err != nil {
		t.Errorf("error creating new parser: %v", err)
	}

	if _, err := p.parseLog(validCombinedLog); err != nil {
		t.Errorf("did not expect error, got %v", err)
	}
}

func TestParser_ParseLogValidCommon(t *testing.T) {

	p, err := NewParser(&mockLogger{}, &defaultRegex)
	if err != nil {
		t.Errorf("error creating new parser: %v", err)
	}

	if _, err := p.parseLog(validCommonLog); err != nil {
		t.Errorf("did not expect error, got %v", err)
	}
}

func TestParser_ParseLogNoParams(t *testing.T) {
	log := `127.0.0.1 user-identifier frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326`

	p, err := NewParser(&mockLogger{}, &defaultRegex)
	if err != nil {
		t.Errorf("error creating new parser: %v", err)
	}

	if _, err := p.parseLog(log); err != nil {
		t.Errorf("did not expect error, got %v", err)
	}
}

func TestParser_ParseLogInvalidFormat(t *testing.T) {
	log := `abc12345`

	m := &mockLogger{}

	p, err := NewParser(m, &defaultRegex)
	if err != nil {
		t.Errorf("error creating new parser: %v", err)
	}

	if _, err := p.parseLog(log); err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestParser_ParseLogInvalidResponseCode(t *testing.T) {

	p, err := NewParser(&mockLogger{}, &defaultRegex)
	if err != nil {
		t.Errorf("error creating parser: %v", err)
	}

	if _, err := p.parseLog(validCombinedLog); err != nil {
		t.Errorf("did not expect error, got %v", err)
	}
}

func TestParser_ParseLogInvalidBytesSent(t *testing.T) {
	log := `127.0.0.1 user-identifier frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif?param1=test HTTP/1.0" 200 abc "referrer" "agent"`

	l := &mockLogger{}

	p, err := NewParser(l, &defaultRegex)
	if err != nil {
		t.Errorf("error creating parser: %v", err)
	}

	if _, err := p.parseLog(log); err != nil {
		t.Errorf("did not expect error, got %v", err)
	}

	expectedLog := `strconv.ParseUint: parsing "abc": invalid syntax`

	allLogs := strings.Join(l.logs, " ")

	if !strings.Contains(allLogs, expectedLog) {
		t.Errorf("expected an error parsing response code to be logged. logs: %s", allLogs)
	}
}

func TestParser_ParseLogInvalidTime(t *testing.T) {
	log := `127.0.0.1 user-identifier frank [99/Abc/2000:13:55:36 -9999] "GET /apache_pb.gif HTTP/1.0" abc 2326`

	l := &mockLogger{}

	p, err := NewParser(l, &defaultRegex)
	if err != nil {
		t.Errorf("error creating parser: %v", err)
	}

	if _, err := p.parseLog(log); err != nil {
		t.Errorf("did not expect error, got %v", err)
	}

	expectedLog := `error parsing response code:  strconv.ParseUint: parsing "abc": invalid syntax error parsing time: parsing time "99/Abc/2000:13:55:36 -9999" as "02/Jan/2006:15:04:05 -0700": cannot parse "Abc/2000:13:55:36 -9999" as "Jan"`

	allLogs := strings.Join(l.logs, " ")

	if !strings.Contains(allLogs, expectedLog) {
		t.Errorf("expected an error parsing response code to be logged. logs: %s", allLogs)
	}

}

func TestParser_ParseBatchValid(t *testing.T) {
	logs := []string{`127.0.0.1 user-identifier frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif?param1=test HTTP/1.0" 200 2326 "referrer" "agent"`, `127.0.0.1 user-identifier frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif?param1=test HTTP/1.0" 200 2326 "referrer" "agent"`}

	l := &mockLogger{}

	p, err := NewParser(l, &defaultRegex)
	if err != nil {
		t.Errorf("error creating parser: %v", err)
	}

	batchChan := make(chan []string, 1)
	parsedLogChan := make(chan []Log, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go p.ParseBatch(1, batchChan, parsedLogChan, &wg)

	batchChan <- logs
	close(batchChan)

	wg.Wait()

	parsedLogs := <-parsedLogChan
	close(parsedLogChan)

	expected := []Log{{
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

	if !reflect.DeepEqual(&expected, &parsedLogs) {
		t.Errorf("expected %v, got %v", expected, parsedLogs)
	}
}

func TestParser_ParseBatchInvalid(t *testing.T) {
	logs := []string{`Invalid log format`}

	l := &mockLogger{}

	p, err := NewParser(l, &defaultRegex)
	if err != nil {
		t.Errorf("error creating parser: %v", err)
	}

	batchChan := make(chan []string, 1)
	parsedLogChan := make(chan []Log, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go p.ParseBatch(1, batchChan, parsedLogChan, &wg)

	batchChan <- logs
	close(batchChan)

	wg.Wait()

	<-parsedLogChan
	close(parsedLogChan)

	expectedLog := "Invalid log format"

	allLogs := strings.Join(l.logs, " ")

	t.Log(logs[0])

	if !strings.Contains(allLogs, expectedLog) {
		t.Errorf("expected an error parsing response code to be logged. logs: %s", allLogs)
	}

}
