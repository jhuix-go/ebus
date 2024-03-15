/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package client

import (
	`errors`
	`fmt`
	`net`
	`runtime/debug`
	`sync`
	`time`

	`go.nanomsg.org/mangos/v3`
	mproto `go.nanomsg.org/mangos/v3/protocol`
	`go.nanomsg.org/mangos/v3/transport/all`

	`github.com/jhuix-go/ebus/log`
	`github.com/jhuix-go/ebus/protocol`
)

type PipeHandler interface {
	OnPipeConnected(p protocol.Pipe)
	OnPipeDataArrived(p protocol.Pipe, msg interface{}) error
	OnPipeClosed(p protocol.Pipe)
	OnPipeTimer(p protocol.Pipe)
}

type Client struct {
	cfg       *Config
	proto     *Protocol
	event     uint32
	socket    protocol.Socket
	handler   PipeHandler
	closed    bool
	establish bool
	times     map[uint32]*time.Timer
	wg        sync.WaitGroup
	sync.Mutex
}

func NewClient(cfg *Config, handler PipeHandler) *Client {
	config := defaultClientConfig
	if cfg != nil {
		config.ServiceId = cfg.ServiceId
		config.EventName = cfg.EventName
		if len(cfg.Address) > 0 {
			config.Address = cfg.Address
		}
		if cfg.SendChanSize > 0 {
			config.SendChanSize = cfg.SendChanSize
		}
		if cfg.RecvChanSize > 0 {
			config.RecvChanSize = cfg.RecvChanSize
		}
		if cfg.ReadTimeout > 0 {
			config.ReadTimeout = cfg.ReadTimeout
		}
		if cfg.WriteTimeout > 0 {
			config.WriteTimeout = cfg.WriteTimeout
		}
		if cfg.Interval > 0 {
			config.Interval = cfg.Interval
		}
		config.DataErrorContinue = cfg.DataErrorContinue
		config.Reconnect = cfg.Reconnect
	}
	cfg = &config
	all.AddTransports(nil)
	proto := NewProtocol()
	socket := protocol.MakeSocket(proto, proto.pipeEventHook)
	if cfg.SendChanSize > 0 {
		_ = socket.SetOption(mangos.OptionWriteQLen, cfg.SendChanSize)
	}
	if cfg.RecvChanSize > 0 {
		_ = socket.SetOption(mangos.OptionReadQLen, cfg.RecvChanSize)
	}
	if cfg.ReadTimeout > 0 {
		_ = socket.SetOption(mangos.OptionRecvDeadline, cfg.ReadTimeout)
	}
	if cfg.WriteTimeout > 0 {
		_ = socket.SetOption(mangos.OptionSendDeadline, cfg.WriteTimeout)
	}
	if cfg.Interval > 0 {
		_ = socket.SetOption(protocol.OptionHeartTime, cfg.Interval)
	}
	if cfg.Reconnect {
		_ = socket.SetOption(mangos.OptionReconnectTime, minReconnectTime)
		_ = socket.SetOption(mangos.OptionMaxReconnectTime, maxReconnectTime)
		_ = socket.SetOption(protocol.OptionReconnect, true)
	}
	return &Client{
		cfg:     cfg,
		proto:   proto,
		event:   protocol.EventNameN(cfg.EventName),
		socket:  socket,
		handler: handler,
		times:   make(map[uint32]*time.Timer),
	}
}

func (c *Client) SetAutoReconnect(v bool) {
	c.Lock()
	c.cfg.Reconnect = v
	_ = c.socket.SetOption(protocol.OptionReconnect, v)
	c.Unlock()
}

func (c *Client) SetDataErrorContinue(v bool) {
	c.cfg.DataErrorContinue = v
}

func (c *Client) SetSendSize(v int) {
	if v > 0 && c.cfg.SendChanSize != v {
		c.cfg.SendChanSize = v
		if c.socket != nil {
			_ = c.socket.SetOption(mangos.OptionWriteQLen, v)
		}
	}
}

func (c *Client) SetRecvSize(v int) {
	if v > 0 && c.cfg.RecvChanSize != v {
		c.cfg.RecvChanSize = v
		if c.socket != nil {
			_ = c.socket.SetOption(mangos.OptionReadQLen, v)
		}
	}
}

func (c *Client) SetInterval(v time.Duration) {
	if v > 0 && c.cfg.Interval != v {
		c.cfg.Interval = v
		if c.socket != nil {
			_ = c.socket.SetOption(protocol.OptionHeartTime, v)
		}
	}
}

func (c *Client) SetReadTimeout(v time.Duration) {
	if v > 0 && c.cfg.ReadTimeout != v {
		c.cfg.ReadTimeout = v
		if c.socket != nil {
			_ = c.socket.SetOption(mangos.OptionRecvDeadline, v)
		}
	}
}

func (c *Client) SetWriteTimeout(v time.Duration) {
	if v > 0 && c.cfg.WriteTimeout != v {
		c.cfg.WriteTimeout = v
		if c.socket != nil {
			_ = c.socket.SetOption(mangos.OptionSendDeadline, v)
		}
	}
}

func (c *Client) Link(p mangos.Pipe) string {
	addr := "->" + c.cfg.Address
	if p != nil {
		var lAddr, rAddr string
		if v, err := p.GetOption(mangos.OptionLocalAddr); err == nil {
			lAddr = v.(net.Addr).String()
		}
		if v, err := p.GetOption(mangos.OptionRemoteAddr); err == nil {
			rAddr = v.(net.Addr).String()
		}
		addr = lAddr + "<->" + rAddr
	}
	return addr
}

func (c *Client) String() string {
	info := c.socket.Info()
	return fmt.Sprintf("%d: {\"event_name\":\"%s\",\"self\":%d,\"self_name\":\"%s\",\"peer\":%d,\"peer_name\":\"%s\"}",
		len(c.proto.pipes), c.cfg.EventName, info.Self, info.SelfName, info.Peer, info.PeerName)
}

func (c *Client) pipeEventHook(pe mangos.PipeEvent, pp protocol.Pipe) interface{} {
	eventId := pp.Event()
	switch pe {
	case mangos.PipeEventAttaching:
		c.onConnection(pp)
		log.Infof("<event> %s connection attaching: %s", protocol.StringEvent(eventId), protocol.Link(pp))
		if !c.establish {
			c.establish = true
			c.wg.Add(1)
			go c.establishConnection()
		}
	case mangos.PipeEventAttached:
		c.registerEvent(pp)
		log.Infof("<event> %s connection attached: %s", protocol.StringEvent(eventId), protocol.Link(pp))
	case mangos.PipeEventDetached:
		log.Infof("<event> %s connection closed: %s", protocol.StringEvent(eventId), protocol.Link(pp))
		c.onClose(pp)
	case protocol.PipeEventRegistered:
		log.Infof("<event> %s connection register remote: %s", protocol.StringEvent(eventId), protocol.Link(pp))
	case protocol.PipeEventHeartBeat:
		return c.pipeHeartHook(pp)
	default:
	}
	return nil
}

func (c *Client) onClose(p protocol.Pipe) {
	if c.handler != nil {
		c.handler.OnPipeClosed(p)
	}
	// c.Reconnect(p.Event())
}

type pipeEventHook struct {
	c      *Client
	event  uint32
	result any
}

func (h *pipeEventHook) PipeEventHook(pe mangos.PipeEvent, pp protocol.Pipe) interface{} {
	if pe == mangos.PipeEventAttaching {
		pp.SetEvent(h.event)
		// reset pipe event hook
		h.c.socket.SetPipeEventHook(h.c.pipeEventHook)
		h.result = pp
	}

	return h.c.pipeEventHook(pe, pp)
}

func (c *Client) pipeHeartHook(p protocol.Pipe) *mproto.Message {
	m := mangos.NewMessage(0)
	id := p.ID()
	m.Header = protocol.PutHeader(m.Header, id, protocol.SignallingControl|protocol.SignallingHeart, protocol.PipeEbus)
	if c.handler != nil {
		c.handler.OnPipeTimer(p)
	}
	return m
}

func (c *Client) Connect(eventName string, addr string) error {
	ev := c.event
	if len(eventName) > 0 {
		ev = protocol.EventNameN(eventName)
	}
	if ev == 0 {
		return protocol.ErrBadEventName
	}

	_, err := c.connect(ev, addr)
	return err
}

func (c *Client) connect(ev uint32, addr string) (any, error) {
	if len(addr) == 0 {
		addr = c.cfg.Address
	}
	if len(addr) == 0 {
		err := protocol.ErrBadAddress
		log.Errorf("<event> connect error: %v", err)
		return nil, err
	}

	c.Lock()
	c.closed = false
	// c.autoReconnect = c.cfg.Reconnect
	c.Unlock()
	hook := &pipeEventHook{c, ev, nil}
	c.socket.SetPipeEventHook(hook.PipeEventHook)
	if err := c.socket.Dial(addr); err != nil {
		log.Errorf("<event> connect to %s error: %v", c.cfg.Address, err)
		return nil, err
	}

	log.Infof("<event> connect to %s succeed", addr)
	return hook.result, nil
}

// func (c *Client) Reconnect(ev uint32) {
// 	c.Lock()
// 	defer c.Unlock()
//
// 	if !c.autoReconnect {
// 		return
// 	}
//
// 	log.Infof("auto reconnect server %s after 5 second", c)
// 	t, ok := c.times[ev]
// 	if ok && t != nil {
// 		t.Stop()
// 	}
// 	t = time.AfterFunc(5*time.Second, func() {
// 		log.Warnf("auto reconnect server %s", c)
// 		_ = c.connect(ev, "")
// 	})
// 	c.times[ev] = t
// }

func (c *Client) registerEvent(p protocol.Pipe) {
	m := mangos.NewMessage(0)
	m.Header = protocol.PutHeader(m.Header, p.ID(), protocol.SignallingControl|protocol.SignallingRegisterEvent, p.Event())
	if err := c.socket.SendMsg(m); err != nil {
		m.Free()
		log.Errorf("<event> %s, register event error: %s", c, err)
	}
}

func (c *Client) onConnection(p protocol.Pipe) {
	if c.handler != nil {
		c.handler.OnPipeConnected(p)
	}
}

func (c *Client) establishConnection() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("<event> handle panic: %v\n%s", err, debug.Stack())
		} else {
			log.Infof("<event> event client closed")
		}
		_ = c.Close()
		c.wg.Done()
	}()

	for {
		m, err := c.socket.RecvMsg()
		if err != nil {
			log.Warnf("<event> failed to receive request: %v", err)
			if errors.Is(err, mangos.ErrClosed) || !c.cfg.DataErrorContinue {
				break
			}

			continue
		}

		h := protocol.Header{Data: m.Header}
		if c.handler != nil {
			p := protocol.LocalPipe(m.Pipe)
			if p == nil {
				p = c.Pipe(h.Dest())
			}
			if p != nil {
				if err = c.handler.OnPipeDataArrived(p, m); err != nil {
					if c.cfg.DataErrorContinue {
						continue
					}

					break
				}
			}
		}
	}
}

func (c *Client) Stop() {
	_ = c.Close()
	c.proto.WaitAllPipe()
	c.wg.Wait()
	c.handler = nil
	log.Infof("<event> event client stopped")
}

func (c *Client) SendEvent(src uint32, eventId uint32, data []byte) error {
	m := mangos.NewMessage(len(data))
	m.Header = protocol.PutHeader(m.Header, src, protocol.SignallingEvent, eventId)
	m.Body = append(m.Body, data...)
	if err := c.socket.SendMsg(m); err != nil {
		m.Free()
		log.Errorf("<event> %d<->%s, send error: %s", src, protocol.StringEvent(eventId), err)
		return err
	}

	return nil
}

func (c *Client) Send(src, dest uint32, data []byte) error {
	m := mangos.NewMessage(len(data))
	signalling := protocol.SignallingAssign
	if dest == c.event {
		signalling = protocol.SignallingEvent
	}
	m.Header = protocol.PutHeader(m.Header, src, signalling, dest)
	m.Body = append(m.Body, data...)
	if err := c.socket.SendMsg(m); err != nil {
		m.Free()
		log.Errorf("<event> %d<->%d, send error: %s", src, dest, err)
		return err
	}

	return nil
}

func (c *Client) Broadcast(data []byte) error {
	m := mangos.NewMessage(len(data))
	m.Header = protocol.PutHeader(m.Header, 0, protocol.SignallingAssign, 0)
	if err := c.socket.SendMsg(m); err != nil {
		m.Free()
		log.Errorf("<event> broadcast error: %s", err)
		return err
	}

	return nil
}

func (c *Client) Close() error {
	c.Lock()
	if c.closed {
		c.Unlock()
		return mproto.ErrClosed
	}

	c.closed = true
	for _, t := range c.times {
		t.Stop()
	}
	clear(c.times)
	c.establish = false
	c.Unlock()
	return c.socket.Close()
}

func (c *Client) Pipe(id uint32) protocol.Pipe {
	return c.proto.Pipe(id)
}

func (c *Client) RangePipes(f func(uint32, protocol.Pipe) bool) {
	c.proto.RangePipes(f)
}
