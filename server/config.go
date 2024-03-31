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
	Address      string        `json:"address" yaml:"address" toml:"address"`
	TraceMessage bool          `json:"trace_message,omitempty" yaml:"trace_message,omitempty" toml:"trace_message,omitempty"`
	SendChanSize int           `json:"send_chan_size" yaml:"send_chan_size" toml:"send_chan_size"`
	RecvChanSize int           `json:"recv_chan_size" yaml:"recv_chan_size" toml:"recv_chan_size"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" toml:"read_timeout"`
	// WriteTimeout   time.Duration `json:"write_timeout" yaml:"write_timeout" toml:"write_timeout"`
}

var defaultServerConfig = Config{
	Address:      "tcp://0.0.0.0:8171",
	SendChanSize: 128,
	RecvChanSize: 128,
	TraceMessage: false,
	ReadTimeout:  time.Second * 60,
}
