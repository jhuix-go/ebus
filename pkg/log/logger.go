/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package log

import (
	"time"
)

const (
	NoneLevel = iota
	// PanicLevel level, the highest level of severity. will call panic() if the logging level <= PanicLevel.
	PanicLevel
	// FatalLevel level. Logs and then calls `logger.Exit(1)`. It will exit even if the
	// logging level <= FatalLevel.
	FatalLevel
	// ErrorLevel level. Runtime errors. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// NoticeLevel level Uncommon events
	NoticeLevel
	// InfoLevel level. Examples: User logs in, SQL logs.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
)

type FieldCall func(v any) any

type Field struct {
	Value any
	Call  FieldCall
}

type Fields []Field

type Entry interface {
	WithLevel(level int) Entry
	WithTime(t time.Time) Entry
	WithFormat(f string) Entry
	WithField(v any, call FieldCall) Entry
	WithFields(Fields) Entry
	Log()
}

type EmptyEntry struct{}

func (l *EmptyEntry) WithLevel(int) Entry            { return l }
func (l *EmptyEntry) WithTime(time.Time) Entry       { return l }
func (l *EmptyEntry) WithFormat(string) Entry        { return l }
func (l *EmptyEntry) WithField(any, FieldCall) Entry { return l }
func (l *EmptyEntry) WithFields(Fields) Entry        { return l }
func (l *EmptyEntry) Log()                           {}

var DefaultEmptyEntry = &EmptyEntry{}

type Logger interface {
	WithError(format string) Entry
	WithWarn(format string) Entry
	WithInfo(format string) Entry
	WithDebug(format string) Entry
	Errorf(format string, v ...any)
	Warnf(format string, v ...any)
	Infof(format string, v ...any)
	Debugf(format string, v ...any)
	SetLevel(level int)
}

type EmptyLogger struct{}

func (l *EmptyLogger) WithError(string) Entry { return DefaultEmptyEntry }
func (l *EmptyLogger) WithWarn(string) Entry  { return DefaultEmptyEntry }
func (l *EmptyLogger) WithInfo(string) Entry  { return DefaultEmptyEntry }
func (l *EmptyLogger) WithDebug(string) Entry { return DefaultEmptyEntry }
func (l *EmptyLogger) Errorf(string, ...any)  {}
func (l *EmptyLogger) Warnf(string, ...any)   {}
func (l *EmptyLogger) Infof(string, ...any)   {}
func (l *EmptyLogger) Debugf(string, ...any)  {}
func (l *EmptyLogger) SetLevel(int)           {}

var (
	defaultEmptyLogger Logger = &EmptyLogger{}
	defaultLogger             = defaultEmptyLogger
)

func SetLogger(nl Logger) {
	Close()
	if nl == nil {
		return
	}

	defaultLogger = nl
}

func SetAsync(async bool) {
	if l, ok := defaultLogger.(*logger); ok {
		l.SetAsync(async)
	}
}

func WithError(format string) Entry {
	return defaultLogger.WithError(format)
}
func WithWarn(format string) Entry {
	return defaultLogger.WithWarn(format)
}
func WithInfo(format string) Entry {
	return defaultLogger.WithInfo(format)
}
func WithDebug(format string) Entry {
	return defaultLogger.WithDebug(format)
}

func Errorf(format string, v ...any) {
	defaultLogger.Errorf(format, v...)
}
func Warnf(format string, v ...any) {
	defaultLogger.Warnf(format, v...)
}
func Infof(format string, v ...any) {
	defaultLogger.Infof(format, v...)
}
func Debugf(format string, v ...any) {
	defaultLogger.Debugf(format, v...)
}

func Close() {
	var old Logger
	old, defaultLogger = defaultLogger, defaultEmptyLogger
	if l, ok := old.(*logger); ok {
		l.Close()
	}
}
