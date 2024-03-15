/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package discovery

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/jhuix-go/ebus/pkg/discovery/registry"
	"github.com/jhuix-go/ebus/pkg/discovery/registry/consul"
)

const defaultTtl = 30 * time.Second

func NewConsulRegistry(cfg *ServerConfig) (registry.Registrar, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, ErrEndpointsInvalid
	}

	config := &api.Config{}
	if err := json.Unmarshal([]byte(cfg.Endpoints[0]), config); err != nil {
		return nil, err
	}

	ttl := cfg.TTL
	if ttl < time.Second {
		ttl = defaultTtl
	}
	if len(cfg.Username) != 0 && len(cfg.Password) != 0 {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}
	r, err := consul.NewRegistrar(
		&consul.Config{
			ConsulCfg:   config,
			RegistryDir: cfg.RegistryDir,
			Ttl:         ttl,
		})
	if err != nil {
		return nil, err
	}

	return r, nil
}
