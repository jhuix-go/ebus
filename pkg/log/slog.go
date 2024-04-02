/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package log

import (
	"path"
	"runtime"
	"strconv"
	`strings`
	"sync"
	"sync/atomic"
	"time"

	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	"github.com/gookit/slog/rotatefile"

	"github.com/jhuix-go/ebus/pkg/queue"
)

func getCaller(callerSkip int) (string, int) {
	pcs := make([]uintptr, 1) // alloc 1 times
	num := runtime.Callers(callerSkip, pcs)
	if num < 1 {
		return "", 0
	}

	f, _ := runtime.CallersFrames(pcs).Next()
	if f.PC == 0 {
		return "", 0
	}

	return f.File, f.Line
}

const (
	argNormal = iota
	argJson
)

type Record struct {
	q         *queue.Queue[*Record]
	level     slog.Level
	timestamp time.Time
	fileName  string
	lineNum   int
	format    string
	args      []Field
}

func newRecord(q *queue.Queue[*Record]) *Record {
	return &Record{q: q}
}

func (e *Record) WithLevel(level int) Entry {
	e.level = slog.Level(level * int(slog.PanicLevel))
	return e
}

func (e *Record) WithTime(t time.Time) Entry {
	e.timestamp = t
	return e
}

func (e *Record) WithFormat(f string) Entry {
	e.format = f
	return e
}

func (e *Record) WithField(v any, call FieldCall) Entry {
	e.args = append(e.args, Field{v, call})
	return e
}

func (e *Record) WithFields(fields Fields) Entry {
	e.args = append(e.args, fields...)
	return e
}

func (e *Record) withCaller(callerSkip int) Entry {
	e.fileName, e.lineNum = getCaller(callerSkip)
	return e
}

func (e *Record) record() {
	var (
		args   []any
		format string
	)
	for _, arg := range e.args {
		if arg.Call != nil {
			args = append(args, arg.Call(arg.Value))
		} else {
			args = append(args, arg.Value)
		}
	}
	if len(e.fileName) != 0 {
		format = e.fileName + ":" + strconv.Itoa(e.lineNum) + " "
	}
	if !e.timestamp.IsZero() {
		format = "[" + e.timestamp.Format(slog.DefaultTimeFormat) + "] " + format
	}
	format += e.format
	switch e.level {
	case slog.ErrorLevel:
		slog.Errorf(format, args...)
	case slog.WarnLevel:
		slog.Warnf(format, args...)
	case slog.InfoLevel:
		slog.Infof(format, args...)
	case slog.DebugLevel:
		slog.Debugf(format, args...)
	default:
		slog.Tracef(format, args...)
	}
}

func (e *Record) Log() {
	if e.q != nil {
		e.q.Enqueue(e)
		return
	}

	e.record()
}

var recordPool = sync.Pool{
	New: func() interface{} {
		return &Record{}
	},
}

type logger struct {
	q     *queue.Queue[*Record]
	wg    sync.WaitGroup
	done  chan struct{}
	async atomic.Bool
	level slog.Level
}

func (l *logger) newRecord() *Record {
	q := l.q
	if !l.async.Load() {
		q = nil
	}
	r := recordPool.Get().(*Record)
	r.q = q
	return r
}

func releaseRecord(r *Record) {
	r.q = nil
	r.args = nil
	r.format = ""
	r.level = slog.Level(0)
	recordPool.Put(r)
}

func newLogger() *logger {
	l := &logger{
		q:     queue.NewQueueWithSize[*Record](1024, 128),
		done:  make(chan struct{}),
		level: slog.InfoLevel,
	}
	return l
}

func record(e *Record) {
	e.record()
	releaseRecord(e)
}

func (l *logger) dispatch() {
	l.async.Store(true)
	if l.done == nil {
		l.done = make(chan struct{})
	}
	q := l.q
	done := l.done
	defer l.wg.Done()

	tick := time.NewTicker(5 * time.Second)

	for {
		select {
		case e := <-q.DequeueC():
			record(e)
		case <-tick.C:
			slog.Flush()
		case <-done:
			tick.Stop()
			l.async.Store(false)
			// processing the remaining logs
			for {
				select {
				case e := <-l.q.DequeueC():
					record(e)
				default:
					return
				}
			}
		}
	}
}

func (l *logger) startAsync() {
	l.wg.Add(1)
	go l.dispatch()
}

func (l *logger) closeAsync() {
	var done chan struct{}
	done, l.done = l.done, nil
	if done != nil {
		close(done)
	}
	l.wg.Wait()
}

func (l *logger) SetAsync(async bool) {
	if l.async.CompareAndSwap(!async, async) {
		if l.async.Load() {
			l.startAsync()
		} else {
			l.closeAsync()
		}
	}
}

var callerSkip = 5

func (l *logger) WithError(format string) Entry {
	if !l.level.ShouldHandling(slog.ErrorLevel) {
		return DefaultEmptyEntry
	}

	return l.newRecord().withCaller(callerSkip).WithTime(time.Now()).WithLevel(ErrorLevel).WithFormat(format)
}

func (l *logger) WithWarn(format string) Entry {
	if !l.level.ShouldHandling(slog.WarnLevel) {
		return DefaultEmptyEntry
	}

	return l.newRecord().withCaller(callerSkip).WithTime(time.Now()).WithLevel(WarnLevel).WithFormat(format)
}

func (l *logger) WithInfo(format string) Entry {
	if !l.level.ShouldHandling(slog.InfoLevel) {
		return DefaultEmptyEntry
	}

	return l.newRecord().withCaller(callerSkip).WithTime(time.Now()).WithLevel(InfoLevel).WithFormat(format)
}

func (l *logger) WithDebug(format string) Entry {
	if !l.level.ShouldHandling(slog.DebugLevel) {
		return DefaultEmptyEntry
	}

	return l.newRecord().withCaller(callerSkip).WithTime(time.Now()).WithLevel(DebugLevel).WithFormat(format)
}

func (l *logger) Errorf(format string, v ...any) {
	if !l.level.ShouldHandling(slog.ErrorLevel) {
		return
	}

	e := l.newRecord()
	e.withCaller(callerSkip).WithTime(time.Now()).WithLevel(ErrorLevel).WithFormat(format)
	for _, a := range v {
		e.WithField(a, nil)
	}
	e.Log()
}
func (l *logger) Warnf(format string, v ...any) {
	if !l.level.ShouldHandling(slog.WarnLevel) {
		return
	}

	e := l.newRecord()
	e.withCaller(callerSkip).WithTime(time.Now()).WithLevel(WarnLevel).WithFormat(format)
	for _, a := range v {
		e.WithField(a, nil)
	}
	e.Log()
}
func (l *logger) Infof(format string, v ...any) {
	if !l.level.ShouldHandling(slog.InfoLevel) {
		return
	}

	e := l.newRecord()
	e.withCaller(callerSkip).WithTime(time.Now()).WithLevel(InfoLevel).WithFormat(format)
	for _, a := range v {
		e.WithField(a, nil)
	}
	e.Log()
}
func (l *logger) Debugf(format string, v ...any) {
	if !l.level.ShouldHandling(slog.DebugLevel) {
		return
	}

	e := l.newRecord()
	e.withCaller(callerSkip).WithTime(time.Now()).WithLevel(DebugLevel).WithFormat(format)
	for _, a := range v {
		e.WithField(a, nil)
	}
	e.Log()
}

func (l *logger) SetLevel(level int) {
	l.level = slog.Level(level * int(slog.PanicLevel))
	slog.SetLogLevel(l.level)
}
func (l *logger) Close() {
	l.closeAsync()
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
	if tf, ok := h.Formatter().(*slog.TextFormatter); ok {
		tf.TimeFormat = slog.DefaultTimeFormat
		tf.SetTemplate("[{{datetime}}] [{{channel}}] [{{level}}] {{message}} {{data}} {{extra}}\n")
	}
	return h, nil
}

type WithConfig func(*handler.Config)

func SetConfig(ops ...WithConfig) {
	for _, op := range ops {
		op(logConfig)
	}
}

func SetLevel(level int) {
	if level > NoneLevel {
		if _, ok := defaultLogger.(*EmptyLogger); ok {
			defaultLogger = newLogger()
		}
		defaultLogger.SetLevel(level)
		return
	}

	Close()
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
		WithMaxAgeOption(o.MaxAge, true), WithLevelOption(o.Level, true),
		WithCompressOption(o.Compress), WithAsyncOption(o.Async))
	for _, op := range ops {
		op(&opt)
	}
	slog.DefaultTimeFormat = "2006/01/02 15:04:05.000000"
	slog.DefaultChannelName = strings.ToUpper(appName)
	slog.Std().ChannelName = slog.DefaultChannelName
	slog.Std().CallerSkip = 8
	slog.Std().CallerFlag = slog.CallerFlagFpLine
	slog.Std().ReportCaller = false
	if tf, ok := slog.Std().Formatter.(*slog.TextFormatter); ok {
		tf.TimeFormat = slog.DefaultTimeFormat
		tf.SetTemplate("[{{channel}}] [{{level}}] {{message}} {{data}} {{extra}}\n")
	}
	fileName := path.Join(opt.OutDir, appName+opt.Filename)
	maxSize := uint64(opt.MaxSize) * rotatefile.OneMByte
	bt := rotatefile.RotateTime(opt.MaxBackups) * rotatefile.EveryDay
	rt := rotatefile.RotateTime(opt.MaxAge) * rotatefile.EveryDay
	level := slog.Level(opt.Level * int(slog.PanicLevel))
	h1, _ := newRotateFileHandler(fileName, level, rt, handler.WithMaxSize(maxSize), handler.WithBackupTime(uint(bt)))
	slog.PushHandler(h1)
	if opt.Level > NoneLevel {
		l := newLogger()
		l.SetAsync(opt.Async)
		defaultLogger = l
		defaultLogger.SetLevel(opt.Level)
	}
}
