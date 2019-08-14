package main

import (
	"github.com/sirupsen/logrus"
)

// Event stores messages to log later, from our standard interface
type Event struct {
	id      int
	message string
}

// StandardLogger enforces specific log message formats
type StandardLogger struct {
	*logrus.Logger
}

// NewLogger initializes the standard logger
func NewLogger() StandardLogger {
	var baseLogger = logrus.New()

	var standardLogger = StandardLogger{baseLogger}
	fm := logrus.FieldMap{
		logrus.FieldKeyTime:  "@timestamp",
		logrus.FieldKeyLevel: "@level",
		logrus.FieldKeyMsg:   "@message",
		logrus.FieldKeyFunc:  "@caller",
	}
	standardLogger.Formatter = &logrus.JSONFormatter{
		FieldMap:    fm,
		PrettyPrint: true,
	}

	return standardLogger
}

// Declare variables to store log messages as new Events
var (
	invalidArgMessage      = Event{1, "Invalid arg: %s"}
	invalidArgValueMessage = Event{2, "Invalid value for argument: %s: %v"}
	missingArgMessage      = Event{3, "Missing arg: %s"}
	infoMessage            = Event{10, "Info message: %s"}
	errorMessage           = Event{99, "Error message:%s"}
	debugMessage           = Event{999, "Debug message:%s"}
)

// InvalidArg is a standard error message
func (l *StandardLogger) InvalidArg(argumentName string) {
	l.Errorf(invalidArgMessage.message, argumentName)
}

// InvalidArgValue is a standard error message
func (l *StandardLogger) InvalidArgValue(argumentName string, argumentValue string) {
	l.Errorf(invalidArgValueMessage.message, argumentName, argumentValue)
}

// MissingArg is a standard error message
func (l *StandardLogger) MissingArg(argumentName string) {
	l.Errorf(missingArgMessage.message, argumentName)
}

// Info is a standard info message
func (l *StandardLogger) Info(message string) {
	l.Infof(infoMessage.message, message)
}

// Error is a standard info message
func (l *StandardLogger) Error(message string) {
	l.Errorf(errorMessage.message, message)
}

// Debug is a standard info message
func (l *StandardLogger) Debug(message string) {
	l.Debugf(debugMessage.message, message)
}
