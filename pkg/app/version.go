/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package app

import (
	`runtime`
	`time`
)

var (
	// Version is app or service version
	Version    = "v0.0.1"
	GoVersion  = runtime.Version()
	BuildTime  = time.Now().Format("2006-01-02 15:04:05.000")
	CommitHash = "localhost"
)
