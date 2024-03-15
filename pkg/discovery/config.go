/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package discovery

import (
	`errors`
	"time"
)

const (
	TypeConsul = "consul"
)

var ErrEndpointsInvalid = errors.New("endpoints invalid")

type ServerConfig struct {
	DiscoveryType  string        `json:"type,omitempty" yaml:"type,omitempty" toml:"type,omitempty"`
	ServiceName    string        `json:"name,omitempty" yaml:"name,omitempty" toml:"name,omitempty"`
	ServiceVersion string        `json:"version,omitempty" yaml:"version,omitempty" toml:"version,omitempty"`
	RegistryDir    string        `json:"registry_dir,omitempty" yaml:"registry_dir,omitempty" toml:"registry_dir,omitempty"`
	NodeID         string        `json:"id,omitempty" yaml:"node_id,omitempty" toml:"id,omitempty"`
	Address        string        `json:"address" yaml:"address" toml:"address"`
	Endpoints      []string      `json:"endpoints" yaml:"endpoints" toml:"endpoints"`
	Username       string        `json:"username,omitempty" yaml:"username,omitempty" toml:"username,omitempty"`
	Password       string        `json:"password,omitempty" yaml:"password,omitempty" toml:"password,omitempty"`
	Interval       time.Duration `json:"interval,omitempty" yaml:"interval,omitempty" toml:"interval,omitempty"`
	TTL            time.Duration `json:"ttl,omitempty" yaml:"ttl,omitempty" toml:"ttl,omitempty"`
}

type ClientConfig struct {
	DiscoveryType  string   `json:"type" yaml:"type" toml:"type"`
	ServiceName    string   `json:"name" yaml:"name" toml:"name"`
	ServiceVersion string   `json:"version,omitempty" yaml:"version,omitempty" toml:"version,omitempty"`
	RegistryDir    string   `json:"registry_dir,omitempty" yaml:"registry_dir,omitempty" toml:"registry_dir,omitempty"`
	Balancer       string   `json:"balancer,omitempty" yaml:"balancer,omitempty" toml:"balancer,omitempty"`
	Endpoints      []string `json:"endpoints" yaml:"endpoints" toml:"endpoints"`
	Username       string   `json:"username,omitempty" yaml:"username,omitempty" toml:"username,omitempty"`
	Password       string   `json:"password,omitempty" yaml:"password,omitempty" toml:"password,omitempty"`
}
