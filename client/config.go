/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package client

import (
	`time`
)

type Config struct {
	ServiceId         int           `json:"service_id" yaml:"service_id" toml:"service_id"`
	EventName         string        `json:"event" yaml:"event" toml:"event"`
	Address           string        `json:"address" yaml:"address" toml:"address"`
	SendChanSize      int           `json:"send_chan_size,omitempty" yaml:"send_chan_size,omitempty" toml:"send_chan_size,omitempty"`
	RecvChanSize      int           `json:"recv_chan_size,omitempty" yaml:"recv_chan_size,omitempty" toml:"recv_chan_size,omitempty"`
	DataErrorContinue bool          `json:"data_error_continue,omitempty" yaml:"data_error_continue,omitempty" toml:"data_error_continue,omitempty"`
	Reconnect         bool          `json:"reconnect,omitempty" yaml:"reconnect,omitempty" toml:"reconnect,omitempty"`
	Interval          time.Duration `json:"interval,omitempty" yaml:"interval,omitempty" toml:"interval,omitempty"`
	ReadTimeout       time.Duration `json:"read_timeout,omitempty" yaml:"read_timeout,omitempty" toml:"read_timeout,omitempty"`
	WriteTimeout      time.Duration `json:"write_timeout,omitempty" yaml:"write_timeout,omitempty" toml:"write_timeout,omitempty"`
}

const (
	minReconnectTime = time.Second
	maxReconnectTime = time.Second * 10
)

var defaultClientConfig = Config{
	Address:           "tcp://127.0.0.1:8171",
	SendChanSize:      128,
	RecvChanSize:      128,
	DataErrorContinue: true,
	Reconnect:         true,
	Interval:          time.Second * 30,
	ReadTimeout:       time.Second * 60,
	WriteTimeout:      time.Second * 10,
}
