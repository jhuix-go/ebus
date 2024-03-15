/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package app

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	`github.com/jhuix-go/ebus/pkg/log`
)

type ServiceInstance interface {
	ParseCommandArgs()
	Initialize(Config) error
	ReloadConfig(Config) error
	RunLoop() error
	Destroy()
}

type ClosureInstance interface {
	Closure()
}

var (
	ServiceName     string
	ApplicationName string
	ServiceApp      ServiceInstance
	ConfigPath      string
	ServiceMode     bool
	OutLogPath      = "../logs"
	ch              = make(chan os.Signal, 1)
)

func SetDefaultOutLogPath(defaultPath string) {
	OutLogPath = defaultPath
}

func ChDir() error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}

	return os.Chdir(dir)
}

func GetApplicationName() string {
	cmd, err := os.Executable()
	if err != nil {
		return ""
	}

	_, exec := filepath.Split(cmd)
	for i := len(exec) - 1; i >= 0; i-- {
		if exec[i] == '.' {
			return exec[:i]
		}
	}

	return exec
}

type WaterConfigProxy struct {
	instance    ServiceInstance
	options     *log.Options
	pprofConfig *PprofConfig
	pprof       *Pprof
}

func (w *WaterConfigProxy) OnConfigChange(cfg Config) {
	if w.options != nil {
		options := initLogOption(cfg)
		ops := make([]log.WithConfig, 0, 4)
		if options.Level != w.options.Level {
			w.options.Level = options.Level
			log.SetLevel(options.Level)
		}
		if options.MaxBackups != w.options.MaxBackups {
			w.options.MaxBackups = options.MaxBackups
			ops = append(ops, log.WithMaxBackupsConfig(options.MaxBackups, true))
		}
		if options.MaxSize != w.options.MaxSize {
			w.options.MaxSize = options.MaxSize
			ops = append(ops, log.WithMaxSizeConfig(options.MaxSize, true))
		}
		if options.MaxAge != w.options.MaxAge {
			w.options.MaxAge = options.MaxAge
			ops = append(ops, log.WithMaxAgeConfig(options.MaxAge, true))
		}
		if options.Compress != w.options.Compress {
			w.options.Compress = options.Compress
			ops = append(ops, log.WithCompressConfig(options.Compress))
		}
		if len(ops) > 0 {
			log.SetConfig(ops...)
		}
	}

	if w.pprofConfig != nil {
		pprofConfig := NewPprofConfig()
		_ = cfg.Sub("pprof").Unmarshal(pprofConfig)
		if pprofConfig.Trace != w.pprofConfig.Trace {
			w.pprofConfig.Trace = pprofConfig.Trace
			if !w.pprofConfig.Trace {
				w.pprof.Stop()
			} else {
				w.pprofConfig.Address = pprofConfig.Address
				w.pprof.SetConfig(w.pprofConfig)
				w.pprof.Start()
			}
		}
		if pprofConfig.Address != w.pprofConfig.Address {
			w.pprofConfig.Address = pprofConfig.Address
			w.pprof.SetConfig(w.pprofConfig)
			if w.pprofConfig.Trace {
				w.pprof.Restart()
			}
		}
	}

	if err := w.instance.ReloadConfig(cfg); err != nil {
		log.Errorf("service config reload failed: %v", err)
	}
}

func onlyShowVersion() bool {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version for "+ServiceName+" service.")
	flag.Parse()
	if showVersion {
		println(ApplicationName + " service version is " + Version + ", build in " + CommitHash + "," + BuildTime)
		println("Compiled by " + GoVersion)
	}
	return showVersion
}

func initLogOption(cfg Config) log.Options {
	options := log.NewOptions()
	options.OutDir = OutLogPath
	if cfg != nil {
		_ = cfg.Sub("log").Unmarshal(options)
	}
	if len(options.Filename) == 0 {
		options.Filename = strings.ToLower(ApplicationName) + ".info.log"
	}
	return options
}

func Start(serviceName string, cfgType string, instance ServiceInstance) {
	ServiceName = serviceName
	ApplicationName = GetApplicationName()
	if instance == nil {
		println(ApplicationName + " app version is " + Version + ", build in " + CommitHash + "," + BuildTime)
		println("Compiled by " + GoVersion)
		println(ApplicationName + " app is nil, will exit.")
		return
	}

	// 强制设置执行文件目录为工作目录
	_ = ChDir()
	// 全局唯一ServiceApp变量
	ServiceApp = instance
	defaultConfig := "../conf/"
	defaultConfig += ApplicationName + "." + cfgType
	flag.StringVar(&ConfigPath, "conf", defaultConfig, ServiceName+" app config path")
	flag.BoolVar(&ServiceMode, "svc", false, "use service mode")
	ServiceApp.ParseCommandArgs()
	if onlyShowVersion() {
		return
	}

	// 拆解配置文件名和相应目录
	cfgName := ApplicationName
	cfgPath, cfgFile := filepath.Split(ConfigPath)
	if cfgFile != "" {
		cfgFileName := strings.Split(cfgFile, ".")
		if len(cfgFileName) > 0 {
			cfgName = cfgFileName[0]
		}
	}
	cfgWater := &WaterConfigProxy{ServiceApp, nil, nil, nil}
	cfg, err := NewConfig(cfgPath, cfgName, cfgType, cfgWater)
	if err != nil {
		println(err.Error())
		if cfg == nil {
			println("read " + ApplicationName + " config be failed, will exit.")
			return
		}
	}

	options := initLogOption(cfg)
	cfgWater.options = &options
	log.InitLogger(ApplicationName, &options)
	defer log.Close()

	pprofConfig := NewPprofConfig()
	if cfg != nil {
		_ = cfg.Sub("pprof").Unmarshal(pprofConfig)
	}
	cfgWater.pprofConfig = pprofConfig
	pprof := NewPprof(pprofConfig)
	cfgWater.pprof = pprof
	pprof.Start()
	defer pprof.Stop()

	log.Infof("app initialize for version is %s, build in %s, %s", Version, CommitHash, BuildTime)
	err = ServiceApp.Initialize(cfg)
	if err != nil {
		log.Infof("app initialize error: %v", err)
		ServiceApp.Destroy()
		return
	}

	cfg.Water()
	ch = make(chan os.Signal, 1)
	defer close(ch)

	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	log.Infof("app starting...")
	err = ServiceApp.RunLoop()
	if err != nil {
		log.Infof("app run error: %v", err)
		return
	}

	for {
		s := <-ch
		log.Infof("get a signal %v", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGINT, syscall.SIGKILL:
			ServiceApp.Destroy()
			time.Sleep(time.Second)
			log.Infof("app exit from signal %d", s)
			return
		case syscall.SIGTERM:
			if ci, ok := ServiceApp.(ClosureInstance); ok {
				ci.Closure()
				log.Infof("app closure from signal %d", s)
			} else {
				ServiceApp.Destroy()
				time.Sleep(time.Second)
				log.Infof("app exit from signal %d", s)
			}
			return
		case syscall.SIGHUP:
		default:
			log.Infof("app exit from default %d", s)
			return
		}
	}
}

func Quit() {
	ch <- syscall.SIGTERM
}
