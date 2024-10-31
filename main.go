/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package main

import (
	"github.com/jhuix-go/ebus/ebus"
	"github.com/jhuix-go/ebus/pkg/app"
	"github.com/jhuix-go/ebus/server"
)

func init() {
	app.Version = Version
	if len(GoVersion) > 0 {
		app.GoVersion = GoVersion
	}
	if len(BuildTime) > 0 {
		app.BuildTime = BuildTime
	}
	if len(CommitHash) > 0 {
		app.CommitHash = CommitHash
	}
}

func main() {
	app.Start(server.SelfName, app.ConfigTypeToml, ebus.NewService())
}
