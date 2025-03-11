// Package parser provides functionality enabling parsing of batches of logs in raw string format (Combined and Common Log Formats are currently supported), returning a batch of parsed logs in a structured format for further processing and/or storage. It is designed for efficient batch processing and concurrent handling of log data.
package parser

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.vxn.dev/xilt/pkg/logger"
)

type Log struct {
	IP           string
	Identity     string
	User         string
	Time         string
	TimestampUTC string
	Method       string
	Route        string
	Params       string
	ResponseCode uint16
	BytesSent    uint32
	Referer      string
	Agent        string
}

type Parser interface {
	parseLog(l string) (*Log, error)
	ParseBatch(id int, batchChan <-chan []string, parsedLogChan chan<- []Log, wg *sync.WaitGroup)
}

type parser struct {
	logger logger.Logger
	regex  *regexp.Regexp
}

const (
	defaultParams        = "-"
	defaultBytesSent     = 0
	defaultReferer       = "-"
	defaultAgent         = "-"
	defaultLogTimeLayout = "02/Jan/2006:15:04:05 -0700"
	timestampUTCLayout   = "2006-01-02T15:04:05Z07:00"
)

var (
	defaultRegex = `^(?<ip>\S*).* (?<identity>\S*) (?<user>\S*) \[(?<timestamp>.*)\]\s"(?<method>\S*)\s(?<route>\S*)\s(?<protocol>[^"]*)"\s(?<response>\S*)\s(?<bytes>\S*)\s?"?(?<referrer>[^"]*)"?\s?"?(?<agent>[^"]*)"?\s*$`
)

// NewParser returns a new Parser instance. It takes a logger instance implementing the Logger interface and a regex pattern string. If the regex pattern is not passed (passing a nil pointer instead), the default regex pattern is used to create the Parser instance.
func NewParser(l logger.Logger, r *string) (*parser, error) {
	if r == nil {
		r = &defaultRegex
	}

	regex, err := regexp.Compile(*r)
	if err != nil {
		return nil, err
	}

	return &parser{
		logger: l,
		regex:  regex,
	}, nil
}

// parseLog takes a single log in raw form and returns a parsed Log struct for further manipulation.
func (p *parser) parseLog(l string) (*Log, error) {

	matches := p.regex.FindStringSubmatch(l)
	if matches == nil {
		return nil, errors.New(("invalid log format: " + l))
	}

	// Initialize the Log struct with default values where needed. If the log being parsed contains the values (not "-"), the default values will be replaced.
	parsedLog := Log{
		Params:  defaultParams,
		Referer: defaultReferer,
		Agent:   defaultAgent,
	}

	// TODO: use net.ParseIP?
	parsedLog.IP = matches[1]
	parsedLog.Identity = matches[2]
	parsedLog.User = matches[3]
	parsedLog.Time = matches[4]
	parsedLog.Method = matches[5]

	// Parse route and params
	uri := strings.Split(matches[6], "?")

	parsedLog.Route = uri[0]

	if len(uri) > 1 {
		parsedLog.Params = uri[1]
	}

	// Parse response code
	responseCode, err := strconv.ParseUint(matches[8], 10, 16)
	if err != nil {
		p.logger.Println("error parsing response code: ", err)
	} else {
		parsedLog.ResponseCode = uint16(responseCode)
	}

	// Parse UTC timestamp
	parsedTime, err := time.Parse(defaultLogTimeLayout, matches[4])
	if err != nil {
		p.logger.Println("error parsing time:", err)
	} else {
		parsedLog.TimestampUTC = parsedTime.UTC().Format(timestampUTCLayout)
	}

	// Parse bytes sent
	bytesSent, err := strconv.ParseUint(matches[9], 10, 64)
	if err != nil {
		// If there is an error parsing as int and the value is not "-", there is some issue with the format. If the value is "-", it is a valid value, therefore no error is logged and the default value of "-" is used.
		if matches[9] != "-" {
			p.logger.Debugf("error parsing bytes sent: %v. proceeding with default value", err)
		}
	} else {
		parsedLog.BytesSent = uint32(bytesSent)
	}

	// Parse Referer if present
	if len(matches) > 9 && matches[10] != "" {
		parsedLog.Referer = matches[10]
	}

	// Parse Agent if present
	if len(matches) > 10 && matches[11] != "" {
		parsedLog.Agent = matches[11]
	}

	return &parsedLog, nil
}

// ParseBatch reads batches of raw logs from an input channel, parses each log in the batch, and sends successfully parsed logs from the batch to an output channel for further processing or storage. It is designed to run concurrently as part of a goroutine, and only valid logs are included in the output batch.
func (p *parser) ParseBatch(id int, batchChan <-chan []string, parsedLogChan chan<- []Log, wg *sync.WaitGroup) {
	defer wg.Done()
	for batch := range batchChan {
		p.logger.Debugf("routine %d beginning to parse a batch of %d logs", id, len(batch))

		parsedLogs := make([]Log, 0, len(batch))

		for _, logEntry := range batch {
			parsedLog, err := p.parseLog(logEntry)
			if err != nil {
				p.logger.Println("error parsing log: ", err)
				continue
			}
			parsedLogs = append(parsedLogs, *parsedLog)
		}

		p.logger.Debugf("routine %d successfully parsed a batch", id)

		parsedLogChan <- parsedLogs
	}
}
