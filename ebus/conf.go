/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package ebus

import (
	"errors"
	"strconv"

	"github.com/jhuix-go/ebus/pkg/app"
	"github.com/jhuix-go/ebus/pkg/discovery"
	"github.com/jhuix-go/ebus/server"
)

type AppConfig struct {
	ServiceId int64 `json:"service_id" yaml:"service_id" toml:"service_id"` // 服务标识（ID）
}

type ServiceConfig struct {
	App       *AppConfig              `json:"app" yaml:"app" toml:"app"`
	Service   *server.Config          `json:"service" yaml:"service" toml:"service"`
	Discovery *discovery.ServerConfig `json:"discovery,omitempty" yaml:"discovery" toml:"discovery"`
}

func (s *Service) ReloadConfig(cfg app.Config) error {
	config := &ServiceConfig{}
	if err := cfg.Unmarshal(config); err != nil {
		return err
	}

	if config.App == nil || config.Service == nil {
		return errors.New("conf is invalid")
	}

	if config.Discovery != nil {
		config.Discovery.DiscoveryType = discovery.TypeConsul
		if len(config.Discovery.RegistryDir) == 0 {
			config.Discovery.RegistryDir = server.SelfName
		}
		config.Discovery.NodeID += strconv.Itoa(int(config.App.ServiceId))
	}
	if s.cfg == nil {
		s.cfg = config
		return nil
	}

	if s.cfg.Service.TraceMessage != config.Service.TraceMessage {
		s.cfg.Service.TraceMessage = config.Service.TraceMessage
		s.svr.SetTraceMessage(config.Service.TraceMessage)
	}
	return nil
}

func (s *Service) InitializeConfig(cfg app.Config) error {
	return s.ReloadConfig(cfg)
}
