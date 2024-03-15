/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package log

import (
	`path`

	`github.com/gookit/slog`
	`github.com/gookit/slog/handler`
	`github.com/gookit/slog/rotatefile`
)

type logger struct {
}

func (l *logger) Errorf(format string, v ...any) {
	slog.Errorf(format, v...)
}
func (l *logger) Warnf(format string, v ...any) {
	slog.Warnf(format, v...)
}
func (l *logger) Infof(format string, v ...any) {
	slog.Infof(format, v...)
}
func (l *logger) Debugf(format string, v ...any) {
	slog.Debugf(format, v...)
}
func (l *logger) Close() {
	_ = slog.Close()
}

var logConfig *handler.Config

// newRotateFileHandler instance. It supports splitting log files by time and size
func newRotateFileHandler(logfile string, level slog.Level, rt rotatefile.RotateTime, fns ...handler.ConfigFn) (*handler.SyncCloseHandler, error) {
	logConfig = handler.NewConfig(fns...).With(handler.WithLogfile(logfile), handler.WithLogLevel(level),
		handler.WithLevelMode(handler.LevelModeValue), handler.WithRotateTime(rt))
	writer, err := logConfig.RotateWriter()
	if err != nil {
		return nil, err
	}

	h := handler.SyncCloserWithMaxLevel(writer, logConfig.Level)
	return h, nil
}

type WithConfig func(*handler.Config)

func SetConfig(ops ...WithConfig) {
	for _, op := range ops {
		op(logConfig)
	}
}

func SetLevel(level int) {
	slog.SetLogLevel(slog.Level(level))
}

func WithLevelConfig(level int, filterDefault bool) WithConfig {
	return WithConfig(func(o *handler.Config) {
		if !filterDefault || level > 0 {
			o.Level = slog.Level(level)
		}
	})
}

func WithFileNameConfig(filename string, filterDefault bool) WithConfig {
	return WithConfig(func(o *handler.Config) {
		if !filterDefault || len(filename) > 0 {
			o.Logfile = filename
		}
	})
}

func WithMaxSizeConfig(maxSize int, filterDefault bool) WithConfig {
	return WithConfig(func(o *handler.Config) {
		if !filterDefault || maxSize > 0 {
			o.MaxSize = uint64(maxSize) * rotatefile.OneMByte
		}
	})
}

func WithMaxBackupsConfig(maxBackups int, filterDefault bool) WithConfig {
	return WithConfig(func(o *handler.Config) {
		if !filterDefault || maxBackups > 0 {
			o.BackupTime = uint(rotatefile.RotateTime(maxBackups) * rotatefile.EveryDay)
		}
	})
}

func WithMaxAgeConfig(maxAge int, filterDefault bool) WithConfig {
	return WithConfig(func(o *handler.Config) {
		if !filterDefault || maxAge > 0 {
			o.RotateTime = rotatefile.RotateTime(maxAge) * rotatefile.EveryDay
		}
	})
}

func WithCompressConfig(compress bool) WithConfig {
	return WithConfig(func(o *handler.Config) {
		o.Compress = compress
	})
}

func InitLogger(appName string, o *Options) {
	opt := defaultOptions
	ops := make([]WithOption, 0, 7)
	ops = append(ops, WithOutDirOption(o.OutDir, true), WithFileNameOption(o.Filename, true),
		WithMaxSizeOption(o.MaxSize, true), WithMaxBackupsOption(o.MaxBackups, true),
		WithMaxAgeOption(o.MaxAge, true), WithLevelOption(o.Level, true), WithCompressOption(o.Compress))
	for _, op := range ops {
		op(&opt)
	}
	slog.DefaultTimeFormat = "2006/01/02 15:04:05.000000"
	slog.DefaultChannelName = appName
	slog.Std().ChannelName = slog.DefaultChannelName
	slog.Std().CallerSkip = 8
	slog.Std().CallerFlag = slog.CallerFlagFpLine
	if tf, ok := slog.Std().Formatter.(*slog.TextFormatter); ok {
		tf.TimeFormat = slog.DefaultTimeFormat
		tf.SetTemplate("[{{datetime}}] [{{channel}}] [{{level}}] {{caller}} {{message}} {{data}} {{extra}}\n")
	}
	fileName := path.Join(opt.OutDir, appName+opt.Filename)
	maxSize := uint64(opt.MaxSize) * rotatefile.OneMByte
	bt := rotatefile.RotateTime(opt.MaxBackups) * rotatefile.EveryDay
	rt := rotatefile.RotateTime(opt.MaxAge) * rotatefile.EveryDay
	level := slog.Level(o.Level)
	h1, _ := newRotateFileHandler(fileName, level, rt, handler.WithMaxSize(maxSize), handler.WithBackupTime(uint(bt)))
	// h2 := handler.ConsoleWithMaxLevel(slog.Level(o.Level))
	slog.PushHandler(h1)
}
