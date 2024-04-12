/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package server

import (
	"time"
)

type Config struct {
	Address       string        `json:"address" yaml:"address" toml:"address"`
	TraceMessage  bool          `json:"trace_message,omitempty" yaml:"trace_message,omitempty" toml:"trace_message,omitempty"`
	ExchangeLines int           `json:"exchange_lines,omitempty" yaml:"exchange_lines,omitempty" toml:"exchange_lines,omitempty"`
	SendChanSize  int           `json:"send_chan_size" yaml:"send_chan_size" toml:"send_chan_size"`
	RecvChanSize  int           `json:"recv_chan_size" yaml:"recv_chan_size" toml:"recv_chan_size"`
	ReadTimeout   time.Duration `json:"read_timeout" yaml:"read_timeout" toml:"read_timeout"`
	// WriteTimeout   time.Duration `json:"write_timeout" yaml:"write_timeout" toml:"write_timeout"`
}

var defaultServerConfig = Config{
	Address:       "tcp://0.0.0.0:8171",
	TraceMessage:  false,
	ExchangeLines: 4,
	SendChanSize:  128,
	RecvChanSize:  128,
	ReadTimeout:   time.Second * 60,
}
