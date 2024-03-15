/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package main

import (
	`github.com/jhuix-go/ebus/app`
	`github.com/jhuix-go/ebus/ebus`
	`github.com/jhuix-go/ebus/server`
)

func main() {
	app.Start(server.SelfName, app.ConfigTypeToml, ebus.NewService())
}
