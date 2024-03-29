/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package cmd

import (
	`fmt`
	`time`

	`github.com/jhuix-go/ebus/pkg/log`
)

func sprintf(format string, v ...any) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000000000")
	format = "[" + timestamp + "]" + " " + format
	_, _ = App.Println(fmt.Sprintf(format, v...))
}

type logger struct{}

func (l *logger) WithError(format string) log.Entry { return log.DefaultEmptyEntry }
func (l *logger) WithWarn(format string) log.Entry  { return log.DefaultEmptyEntry }
func (l *logger) WithInfo(format string) log.Entry  { return log.DefaultEmptyEntry }
func (l *logger) WithDebug(format string) log.Entry { return log.DefaultEmptyEntry }
func (l *logger) Errorf(format string, v ...any)    { sprintf(format, v...) }
func (l *logger) Warnf(format string, v ...any)     { sprintf(format, v...) }
func (l *logger) Infof(format string, v ...any)     { sprintf(format, v...) }
func (l *logger) Debugf(format string, v ...any)    { sprintf(format, v...) }
func (l *logger) SetLevel(level int)                {}

var defaultLogger = &logger{}

func printf(format string, v ...any) {
	_, _ = App.Println(fmt.Sprintf(format, v...))
}

func headColorPrintf(format string, v ...any) {
	_, _ = App.Config().HelpHeadlineColor.Fprintln(App, fmt.Sprintf(format, v...))
}

func headlinePrinter() func(v ...interface{}) (int, error) {
	if App.Config().NoColor || App.Config().HelpHeadlineColor == nil {
		return App.Println
	}
	return func(v ...interface{}) (int, error) {
		return App.Config().HelpHeadlineColor.Fprintln(App, v...)
	}
}

func printHeadline(s string) {
	hp := headlinePrinter()
	if App.Config().HelpHeadlineUnderline {
		_, _ = hp(s)
		u := ""
		for i := 0; i < len(s); i++ {
			u += "="
		}
		_, _ = hp(u)
	} else {
		_, _ = hp(s)
	}
}

func init() {
	log.SetLogger(defaultLogger)
}
