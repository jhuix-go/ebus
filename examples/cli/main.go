/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package main

import (
	`github.com/desertbit/grumble`

	`github.com/jhuix-go/ebus/examples/cli/cmd`
)

func main() {
	grumble.Main(cmd.App)
}
