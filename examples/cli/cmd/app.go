/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package cmd

import (
	`errors`
	`os`
	`path/filepath`

	`github.com/desertbit/grumble`
	`github.com/fatih/color`

	ebus `github.com/jhuix-go/ebus/client`
)

var App = grumble.New(&grumble.Config{
	Name:                  "ebus",
	Description:           "An event bus CLI.",
	HistoryFile:           filepath.Join(os.TempDir(), ".ebus_history"),
	Prompt:                "ebus§ ",
	PromptColor:           color.New(color.FgGreen, color.Bold),
	HelpHeadlineColor:     color.New(color.FgGreen),
	HelpHeadlineUnderline: true,
	HelpSubCommands:       true,

	Flags: func(f *grumble.Flags) {
		f.String("d", "directory", "", "set an alternative root directory path")
		f.Bool("v", "verbose", false, "enable verbose mode")
	},
})

var ebClt *ebus.Client

var (
	ErrServerNotExist = errors.New("event bus be not run")
	ErrClientNotExist = errors.New("event client be not run")
	ErrPipeNotExist   = errors.New("event pipe be not exist")
	ErrInvalidParam   = errors.New("command: invalid param")
	ErrParamsIsEmpty  = errors.New("command: params is empty")
)

func appClose() error {
	if ebClt != nil {
		ebClt.Stop()
	}

	return nil
}

func init() {
	App.OnClose(appClose)
}
