/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package log

import (
	`github.com/gookit/slog`
)

type Options struct {
	OutDir     string `json:"out_dir" yaml:"out_dir" toml:"out_dir"`
	Filename   string `json:"filename" yaml:"filename" toml:"filename"`
	MaxSize    int    `json:"max_size" yaml:"max_size" toml:"max_size"`
	MaxBackups int    `json:"max_backups" yaml:"max_backups" toml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age" toml:"max_age"`
	Compress   bool   `json:"compress" yaml:"compress" toml:"compress"`
	Level      int    `json:"level" yaml:"level" toml:"level"`
	Async      bool   `json:"async" yaml:"async" toml:"async"`
}

var defaultOptions = Options{
	OutDir:     "../logs",
	Filename:   ".logs",
	MaxSize:    50,
	MaxBackups: 30,
	MaxAge:     15,
	Compress:   true,
	Level:      int(slog.InfoLevel),
	Async:      false,
}

func NewOptions() Options {
	return defaultOptions
}

type WithOption func(o *Options)

func WithOutDirOption(outDir string, filterDefault bool) WithOption {
	return WithOption(func(o *Options) {
		if !filterDefault || len(outDir) > 0 {
			o.OutDir = outDir
		}
	})
}

func WithFileNameOption(filename string, filterDefault bool) WithOption {
	return WithOption(func(o *Options) {
		if !filterDefault || len(filename) > 0 {
			o.Filename = filename
		}
	})
}

func WithMaxSizeOption(maxSize int, filterDefault bool) WithOption {
	return WithOption(func(o *Options) {
		if !filterDefault || maxSize > 0 {
			o.MaxSize = maxSize
		}
	})
}

func WithMaxBackupsOption(maxBackups int, filterDefault bool) WithOption {
	return WithOption(func(o *Options) {
		if !filterDefault || maxBackups > 0 {
			o.MaxBackups = maxBackups
		}
	})
}

func WithMaxAgeOption(maxAge int, filterDefault bool) WithOption {
	return WithOption(func(o *Options) {
		if !filterDefault || maxAge > 0 {
			o.MaxAge = maxAge
		}
	})
}

func WithLevelOption(level int, filterDefault bool) WithOption {
	return WithOption(func(o *Options) {
		if !filterDefault || level != 0 {
			o.Level = level
		}
	})
}

func WithCompressOption(compress bool) WithOption {
	return WithOption(func(o *Options) {
		o.Compress = compress
	})
}

func WithAsyncOption(async bool) WithOption {
	return WithOption(func(o *Options) {
		o.Async = async
	})
}
