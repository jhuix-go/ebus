/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package app

import (
	"context"
	`errors`
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	`github.com/jhuix-go/ebus/log`
)

const defaultPprofAddress = "0.0.0.0:6060"

type PprofConfig struct {
	Trace   bool   `json:"trace,omitempty" yaml:"trace,omitempty" toml:"trace,omitempty"`
	Address string `json:"address,omitempty" yaml:"address,omitempty" toml:"address,omitempty"`
}

type Pprof struct {
	cfg *PprofConfig
	srv *http.Server
	wg  *sync.WaitGroup
}

func NewPprofConfig() *PprofConfig {
	return &PprofConfig{false, defaultPprofAddress}
}

func NewPprof(cfg *PprofConfig) *Pprof {
	if len(cfg.Address) == 0 {
		cfg.Address = defaultPprofAddress
	}
	return &Pprof{cfg: cfg, wg: &sync.WaitGroup{}}
}

func (p *Pprof) Start() {
	if !p.cfg.Trace {
		return
	}

	log.Infof("pprof running: %s", p.cfg.Address)
	if p.srv == nil {
		p.srv = &http.Server{Addr: p.cfg.Address}
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("pprof ListenAndServe: %v", err)
			p.srv = nil
		}
	}()
}

func (p *Pprof) Stop() {
	if p.srv == nil {
		return
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		if err := p.srv.Shutdown(ctx); err != nil {
			log.Errorf("pprof Shutdown: %v", err)
		}
	}()
	p.wg.Wait()
	p.srv = nil
	log.Infof("pprof stopped: %s", p.cfg.Address)
}

func (p *Pprof) SetConfig(cfg *PprofConfig) {
	if len(cfg.Address) == 0 {
		cfg.Address = defaultPprofAddress
	}
	p.cfg = cfg
}

func (p *Pprof) Restart() {
	p.Stop()
	p.Start()
}
