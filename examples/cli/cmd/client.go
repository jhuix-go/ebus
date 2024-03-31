/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package cmd

import (
	"strconv"

	"github.com/desertbit/grumble"
	mproto "go.nanomsg.org/mangos/v3/protocol"

	ebus "github.com/jhuix-go/ebus/client"
	"github.com/jhuix-go/ebus/pkg/log"
	"github.com/jhuix-go/ebus/protocol"
)

type ClientHandler struct{}

func (h *ClientHandler) OnPipeConnected(p protocol.Pipe) {}
func (h *ClientHandler) OnPipeDataArrived(p protocol.Pipe, msg interface{}) error {
	m := msg.(*mproto.Message)
	log.Infof("event %s pipe data arrived: %s %s", protocol.EventName(p.Event()), protocol.StringHeader(m.Header), string(m.Body))
	return nil
}
func (h *ClientHandler) OnPipeClosed(p protocol.Pipe) {}
func (h *ClientHandler) OnPipeTimer(p protocol.Pipe)  {}

var clientCmd = &grumble.Command{
	Name:    "client",
	Help:    "admin a event client",
	Aliases: []string{"clt"},
}

var cltRunCmd = &grumble.Command{
	Name: "run",
	Help: "run a event client",
	Args: func(a *grumble.Args) {
		a.String("name", "event name")
		a.String("address", "connect to bus address: tcp://127.0.0.1:8000 ws://127.0.0.1:8000 etc.",
			grumble.Default("tcp://127.0.0.1:8171"))
	},
	Run: func(c *grumble.Context) error {
		name := c.Args.String("name")
		if len(name) == 0 {
			return ErrParamsIsEmpty
		}

		addr := c.Args.String("address")
		if len(addr) == 0 {
			return ErrParamsIsEmpty
		}

		if ebClt == nil {
			ebClt = ebus.NewClient(nil, &ClientHandler{})
		}

		if err := ebClt.Connect(name, addr); err != nil {
			return err
		}

		printf("result: %s event client connect to %s succeed", name, addr)
		return nil
	},
}

var cltQueryCmd = &grumble.Command{
	Name:    "query",
	Help:    "query event pipes of client",
	Aliases: []string{"q"},
	Args: func(a *grumble.Args) {
		a.Uint("id", "id of event pipe", grumble.Default(uint(0)))
	},
	Run: func(c *grumble.Context) error {
		headColorPrintf("query event pipes of client:")
		if ebClt == nil {
			return ErrClientNotExist
		}

		id := c.Args.Uint("id")
		if id == 0 {
			printHeadline("id              event      remote           address")
			ebClt.RangePipes(func(id uint32, p protocol.Pipe) bool {
				printf("%10d      %s       %10d       %s<->%s",
					p.ID(), protocol.EventName(p.Event()), p.RemoteID(), p.LocalAddr(), p.RemoteAddr())
				return true
			})
			return nil
		}

		p := ebClt.Pipe(uint32(id))
		if p == nil {
			return ErrPipeNotExist
		}

		printHeadline("id              event      remote           address")
		printf("%10d      %s       %10d       %s<->%s",
			id, protocol.EventName(p.Event()), p.RemoteID(), p.LocalAddr(), p.RemoteAddr())
		return nil
	},
}

// var cltSendCmd = &grumble.Command{
// 	Name: "send",
// 	Help: "send event pipe message of client",
// 	Args: func(a *grumble.Args) {
// 		a.Uint("src", "src of event pipe", grumble.Default(uint(0)))
// 		a.Uint("dest", "dest of event pipe", grumble.Default(uint(0)))
// 		a.String("data", "message data for send", grumble.Default(""))
// 	},
// 	Run: func(c *grumble.Context) error {
// 		headColorPrintf("send event pipe message of client:")
// 		if ebClt == nil {
// 			return ErrServerNotExist
// 		}
//
// 		src := c.Args.Uint("src")
// 		if src == 0 {
// 			return ErrParamsIsEmpty
// 		}
//
// 		dest := c.Args.Uint("dest")
// 		if dest == 0 {
// 			return ErrParamsIsEmpty
// 		}
//
// 		data := c.Args.String("data")
// 		if len(data) == 0 {
// 			return ErrParamsIsEmpty
// 		}
//
// 		if err := ebClt.Send(uint32(src), uint32(dest), []byte(data)); err != nil {
// 			return err
// 		}
//
// 		log.Infof("send message succeed: %d->%d %s", src, dest, data)
// 		return nil
// 	},
// }

var cltSendCmd = &grumble.Command{
	Name: "send",
	Help: "send event pipe message of client",
	Flags: func(fs *grumble.Flags) {
		fs.Bool("e", "event", false, "send event data mode")
		fs.Uint64("s", "hash", 0, "use hash for select event")
	},
	Args: func(as *grumble.Args) {
		as.Uint("src", "src of event pipe", grumble.Default(uint(0)))
		as.String("dest", "dest of event pipe", grumble.Default(""))
		// as.Uint64("hash", "hash of event pipe", grumble.Default(uint64(0)))
		as.String("data", "message data for send", grumble.Default(""))
	},
	Run: func(c *grumble.Context) error {
		headColorPrintf("send event pipe message of client:")
		if ebClt == nil {
			return ErrServerNotExist
		}

		src := uint32(c.Args.Uint("src"))
		if src == 0 {
			return ErrParamsIsEmpty
		}

		destArg := c.Args.String("dest")
		if len(destArg) == 0 {
			return ErrParamsIsEmpty
		}

		data := c.Args.String("data")
		if len(data) == 0 {
			return ErrParamsIsEmpty
		}

		var dest uint32
		if c.Flags.Bool("event") {
			dest = protocol.EventNameN(destArg)
		} else {
			v, _ := strconv.Atoi(destArg)
			dest = uint32(v)
		}
		hash := c.Flags.Uint64("hash")
		if err := ebClt.SendData(src, dest, hash, []byte(data)); err != nil {
			return err
		}

		log.Infof("send event message succeed: %d->%s %s", src, dest, data)
		return nil
	},
}

var cltCloseCmd = &grumble.Command{
	Name: "close",
	Help: "close event pipe of client",
	Args: func(a *grumble.Args) {
		a.Uint("id", "id of event pipe", grumble.Default(uint(0)))
	},
	Run: func(c *grumble.Context) error {
		headColorPrintf("close event pipe of client:")
		if ebClt == nil {
			return ErrClientNotExist
		}

		id := c.Args.Uint("id")
		if id == 0 {
			return ErrParamsIsEmpty
		}

		p := ebClt.Pipe(uint32(id))
		if p == nil {
			return ErrPipeNotExist
		}

		localAddr := p.LocalAddr()
		if err := p.Close(); err != nil {
			return err
		}

		printf("close %d event pipe succeed: %s", id, localAddr)
		return nil
	},
}

func init() {
	clientCmd.AddCommand(cltRunCmd)
	clientCmd.AddCommand(cltQueryCmd)
	clientCmd.AddCommand(cltSendCmd)
	clientCmd.AddCommand(cltCloseCmd)
	App.AddCommand(clientCmd)
}
