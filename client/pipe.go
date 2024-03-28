/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package client

import (
	`net`
	`time`

	`go.nanomsg.org/mangos/v3`
	mproto `go.nanomsg.org/mangos/v3/protocol`

	`github.com/jhuix-go/ebus/protocol`
)

type Pipe struct {
	p           mproto.Pipe
	s           *Protocol
	event       uint32
	remote      uint32
	closeQ      chan struct{}
	sendQ       chan *mproto.Message
	heartExpire time.Duration
	heartTimer  *time.Ticker
	hook        protocol.PipeEventHook
	data        interface{}
}

func (p *Pipe) Event() uint32 {
	return p.event
}

func (p *Pipe) SetEvent(v uint32) {
	p.event = v
}

func (p *Pipe) RemoteID() uint32 {
	return p.remote
}

func (p *Pipe) LocalAddr() string {
	if v, err := p.Pipe().GetOption(mangos.OptionLocalAddr); err == nil {
		return v.(net.Addr).String()
	}

	return ""
}

func (p *Pipe) RemoteAddr() string {
	if v, err := p.Pipe().GetOption(mangos.OptionRemoteAddr); err == nil {
		return v.(net.Addr).String()
	}

	return ""
}

func (p *Pipe) Pipes() map[uint32]*Pipe {
	return p.s.pipes
}

func (p *Pipe) Pipe() mangos.Pipe {
	return p.p.(mangos.Pipe)
}

func (p *Pipe) ID() uint32 {
	return p.p.ID()
}

func (p *Pipe) Close() error {
	return p.p.Close()
}

func (p *Pipe) Stop() {
	if mp, ok := p.p.(mangos.Pipe); ok {
		_ = mp.Dialer().Close()
	}
	_ = p.p.Close()
	return
}

func (p *Pipe) SendMsg(m *mproto.Message) error {
	return p.p.SendMsg(m)
}

func (p *Pipe) RecvMsg() *mproto.Message {
	return p.p.RecvMsg()
}

func (p *Pipe) SetPrivate(v interface{}) {
	p.data = v
}

func (p *Pipe) GetPrivate() interface{} {
	return p.data
}

func (p *Pipe) release() {
	p.s = nil
	p.hook = nil
	p.data = nil
}

func (p *Pipe) receiver() {
	s := p.s
	pp := p.p
	s.wg.Add(1)
outer:
	for {
		m := pp.RecvMsg()
		if m == nil {
			break
		}

		if len(m.Body) < protocol.DefaultHeaderLength {
			m.Free() // ErrGarbled
			continue
		}

		headerLength := protocol.DefaultHeaderLength
		h := protocol.Header{Data: m.Body[:headerLength]}
		if h.Flag() != protocol.DefaultFlag {
			m.Free()
			break
		}

		headerLength = int(h.HeaderLength())
		if headerLength < protocol.DefaultHeaderLength || headerLength > protocol.MaxHeaderLength {
			m.Free()
			continue
		}

		// m.Header = append(m.Header, m.Body[:headerLength]...)
		m.Header = m.Body[:headerLength]
		m.Body = m.Body[headerLength:]
		if h.IsRegisterEvent() {
			p.remote = h.Src()
		}
		s.Lock()
		recvQ := s.recvQ
		sizeQ := s.sizeQ
		s.Unlock()

		h.SetDest(pp.ID())
		entry := recvQEntry{m, p}
		select {
		case recvQ <- entry:
		case <-sizeQ:
			m.Free()
		case <-p.closeQ:
			m.Free()
			break outer
		}
	}
	_ = pp.Close()
	s.wg.Done()
}

func (p *Pipe) sender() {
	s := p.s
	pp := p.p
	s.wg.Add(1)
	timeQ := nilQ
	if p.heartExpire > 0 {
		if p.heartTimer == nil {
			p.heartTimer = time.NewTicker(p.heartExpire)
		} else {
			p.heartTimer.Reset(p.heartExpire)
		}
		timeQ = p.heartTimer.C
	}
outer:
	for {
		var m *mproto.Message
		select {
		case m = <-p.sendQ:
		case <-timeQ:
			if p.hook == nil {
				continue
			}
			m, _ = p.hook(protocol.PipeEventHeartBeat, p).(*mproto.Message)
			if m == nil {
				continue
			}
		case <-p.closeQ:
			break outer
		}

		if e := pp.SendMsg(m); e != nil {
			m.Free()
			break
		}
	}
	if p.heartTimer != nil {
		p.heartTimer.Stop()
		p.heartTimer = nil
	}
	_ = pp.Close()
	s.wg.Done()
}
