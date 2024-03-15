/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package log

type Logger interface {
	Errorf(format string, v ...any)
	Warnf(format string, v ...any)
	Infof(format string, v ...any)
	Debugf(format string, v ...any)
}

type EmptyLogger struct{}

func (l *EmptyLogger) Errorf(format string, v ...any) {}
func (l *EmptyLogger) Warnf(format string, v ...any)  {}
func (l *EmptyLogger) Infof(format string, v ...any)  {}
func (l *EmptyLogger) Debugf(format string, v ...any) {}

var (
	defaultEmptyLogger Logger = &EmptyLogger{}
	defaultLogger      Logger = &logger{}
)

func SetLogger(logger Logger) {
	if logger == nil {
		if _, ok := defaultLogger.(*EmptyLogger); !ok {
			defaultLogger = defaultEmptyLogger
		}
		return
	}

	defaultLogger = logger
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
	if l, ok := defaultLogger.(*logger); ok {
		l.Close()
	}
}
