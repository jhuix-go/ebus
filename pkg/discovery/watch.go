/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package discovery

import (
	`encoding/json`

	`github.com/hashicorp/consul/api`

	`github.com/jhuix-go/ebus/pkg/discovery/watch`
	`github.com/jhuix-go/ebus/pkg/discovery/watch/consul`
)

func NewConsulWatch(cfg *ClientConfig, handler watch.ClientManagerHandler) (watch.Watcher, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, ErrEndpointsInvalid
	}

	config := &api.Config{}
	if err := json.Unmarshal([]byte(cfg.Endpoints[0]), config); err != nil {
		return nil, err
	}

	if len(cfg.Username) != 0 && len(cfg.Password) != 0 {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}
	w, err := consul.NewWatcher(&consul.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.ServiceVersion,
		RegistryDir:    cfg.RegistryDir,
		Balancer:       cfg.Balancer,
	}, config, handler)
	if err != nil {
		return nil, err
	}

	return w, nil
}
