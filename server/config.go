/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package server

import (
	`time`
)

type Config struct {
	Address           string        `json:"address" yaml:"address" toml:"address"`
	SendChanSize      int           `json:"send_chan_size" yaml:"send_chan_size" toml:"send_chan_size"`
	RecvChanSize      int           `json:"recv_chan_size" yaml:"recv_chan_size" toml:"recv_chan_size"`
	DataErrorContinue bool          `json:"data_error_continue,omitempty" yaml:"data_error_continue,omitempty" toml:"data_error_continue,omitempty"`
	ReadTimeout       time.Duration `json:"read_timeout" yaml:"read_timeout" toml:"read_timeout"`
	// WriteTimeout   time.Duration `json:"write_timeout" yaml:"write_timeout" toml:"write_timeout"`
}

var defaultServerConfig = Config{
	Address:           "tcp://0.0.0.0:8171",
	SendChanSize:      128,
	RecvChanSize:      128,
	DataErrorContinue: true,
	ReadTimeout:       time.Second * 60,
}
