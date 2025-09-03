/*
The MIT License (MIT)

Copyright © 2025 linuxdaemon <linuxdaemon.irc@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package logger provides logging functionality for the application.
// It implements a structured logging interface using zap.
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger
type Logger struct {
	*zap.Logger
}

// Error logs a message at Error level
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// New creates a new logger with the specified log level
func New(level string) (*Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{logger}, nil
}

// NewDefault creates a new logger with default settings (INFO level)
func NewDefault() *Logger {
	logger, _ := New("info")
	return logger
}

// With creates a child logger with the given fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{l.Logger.With(fields...)}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// SyncQuietly flushes any buffered log entries and ignores errors
func (l *Logger) SyncQuietly() {
	// Ignore sync errors so we exit cleanly
	_ = l.Logger.Sync()
}

// Fatal logs a message at Fatal level and then calls os.Exit(1)
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
	os.Exit(1)
}
