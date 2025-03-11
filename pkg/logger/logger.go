// Package logger provides a wrapper around the log library. It enables the usage of the Debug and Debugf methods which log only if the boolean provided to NewLogger is set to true.
package logger

import (
	"log"
)

type Logger interface {
	Println(v ...any)
	Printf(format string, v ...any)
	Debug(v ...any)
	Debugf(format string, v ...any)
}

type logger struct {
	verbose bool
}

// NewLogger returns a new instance of a logger.
func NewLogger(v bool) *logger {
	return &logger{
		verbose: v,
	}
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Println].
func (l *logger) Println(v ...any) {
	log.Println(v...)
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Printf].
func (l *logger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}

// Debug calls Output to print to the standard logger if the verbose flag is enabled in the config.
// Arguments are handled in the manner of [fmt.Println].
func (l *logger) Debug(v ...any) {
	if l.verbose {
		log.Println(v...)
	}
}

// Debugf calls Output to print to the standard logger if the verbose flag is enabled in the config.
// Arguments are handled in the manner of [fmt.Printf].
func (l *logger) Debugf(format string, v ...any) {
	if l.verbose {
		log.Printf(format, v...)
	}
}
