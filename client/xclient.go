/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package client

import (
	"runtime/debug"
	`sync/atomic`
	"time"

	"go.nanomsg.org/mangos/v3"
	mproto "go.nanomsg.org/mangos/v3/protocol"

	"github.com/jhuix-go/ebus/pkg/discovery"
	"github.com/jhuix-go/ebus/pkg/discovery/watch"
	"github.com/jhuix-go/ebus/pkg/log"
	"github.com/jhuix-go/ebus/pkg/queue"
	`github.com/jhuix-go/ebus/pkg/runtime`
	"github.com/jhuix-go/ebus/protocol"
)

type Options struct {
	SerializeType byte          `json:"serialize_type,omitempty" yaml:"serialize_type,omitempty" toml:"serialize_type,omitempty"`
	CompressType  byte          `json:"compress_type,omitempty" yaml:"compress_type,omitempty" toml:"compress_type,omitempty"`
	NotifyBlock   bool          `json:"notify_block,omitempty" yaml:"notify_block,omitempty" toml:"notify_block,omitempty"`
	IdleTimeout   time.Duration `json:"idle_timeout,omitempty" yaml:"idle_timeout,omitempty" toml:"idle_timeout,omitempty"`
	TraceMessage  bool          `json:"trace_message,omitempty" yaml:"trace_message,omitempty" toml:"trace_message,omitempty"`
}

type XClient struct {
	*Client
	handler PipeHandler
	opt     Options
	watcher watch.Watcher
	pipes   map[string]protocol.Pipe
	pending map[uint64]*Call
	q       *queue.Queue[*mangos.Message]
	seq     atomic.Uint32
	done    chan struct{}
}

func NewXClient(cfg *Config, watchCfg *discovery.ClientConfig, handler PipeHandler) *XClient {
	clt := &XClient{
		handler: handler,
		pipes:   make(map[string]protocol.Pipe),
		q:       queue.NewQueueWithSize[*mangos.Message](defaultQLen, defaultQLen),
	}
	clt.Client = NewClient(cfg, clt)
	if watchCfg != nil && len(watchCfg.Endpoints) > 0 {
		watcher, err := discovery.NewConsulWatch(watchCfg, clt)
		if err != nil {
			log.Errorf("<event> create discovery watch error: %v", err)
		} else {
			clt.watcher = watcher
		}
	}
	return clt
}

func (c *XClient) AddClient(_ string, addr string) {
	_, _ = c.connect(c.event, addr)
}

func (c *XClient) UpdateClient(_ string, addr string, conn any) {
	c.pipes[addr] = conn.(protocol.Pipe)
}

func (c *XClient) RemoveClient(_ string, addr string) {
	pipe, ok := c.pipes[addr]
	if ok {
		pipe.Stop()
	}
}

func (c *XClient) dispatch() {
	defer runtime.HandleCrash(false, func(err any) {
		if err != nil {
			log.Errorf("<event> handle panic: %v\n%s", err, debug.Stack())
		}
	})

	for {
		select {
		case m := <-c.q.DequeueC():
			p := m.Pipe.(mproto.Pipe).GetPrivate().(protocol.Pipe)
			_ = c.handler.OnPipeDataArrived(p, m)
			m.Free()
		case <-c.done:
			return
		}
	}
}

func (c *XClient) handleMessage(msg *mangos.Message) {
	c.q.Enqueue(msg)
}

func (c *XClient) OnPipeConnected(p protocol.Pipe) {
	if c.watcher != nil {
		c.watcher.Update(p.Pipe().Address(), p)
	}
	c.wg.Start(func() {
		c.dispatch()
	})
	c.handler.OnPipeConnected(p)
}

func (c *XClient) OnPipeDataArrived(p protocol.Pipe, msg interface{}) error {
	m, err := c.receive(msg.(*mangos.Message))
	if err != nil {
		return err
	}

	if m == nil {
		return nil
	}

	c.handleMessage(m)
	return nil
}

func (c *XClient) OnPipeClosed(p protocol.Pipe) {
	if c.watcher != nil {
		c.watcher.Remove(p.Pipe().Address())
	}
	c.handler.OnPipeClosed(p)
}

func (c *XClient) OnPipeTimer(p protocol.Pipe) {
	c.handler.OnPipeTimer(p)
}

func (c *XClient) Watch() {
	c.wg.Start(func() {
		c.watcher.Watch()
	})
}

func (c *XClient) Stop() {
	c.watcher.Close()
	close(c.done)
	c.Client.Stop()
	c.wg.Wait()
}

func (c *XClient) Pick(hash string) (protocol.Pipe, error) {
	if c.watcher == nil {
		return nil, protocol.ErrNoWatcher
	}

	conn := c.watcher.Pick(&watch.PickInfo{Hash: hash})
	if conn == nil {
		return nil, protocol.ErrNoSource
	}

	p, _ := conn.(protocol.Pipe)
	return p, nil
}

func (c *XClient) PickSend(srcHash string, dest uint32, destHash uint64, data []byte) error {
	p, err := c.Pick(srcHash)
	if err != nil {
		return err
	}

	src := p.RemoteID()
	return c.SendData(src, dest, destHash, data)
}
