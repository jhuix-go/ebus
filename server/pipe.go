/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package server

import (
	`net`
	`time`

	`go.nanomsg.org/mangos/v3`
	mproto `go.nanomsg.org/mangos/v3/protocol`

	`github.com/jhuix-go/ebus/pkg/queue`
	`github.com/jhuix-go/ebus/protocol`
)

type Pipe struct {
	p          mproto.Pipe
	proto      *Protocol
	event      uint32
	remote     uint32
	closeQ     chan struct{}
	sendQ      *queue.Queue[*mproto.Message]
	recvExpire time.Duration
	data       interface{}
	// sendQ      chan *mproto.Message
}

type recvQEntry struct {
	m *mproto.Message
	p *Pipe
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
	_ = p.Close()
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
	p.proto = nil
	p.data = nil
}

func (p *Pipe) sender() {
	s := p.proto
	pp := p.p
	s.wg.Add(1)
outer:
	for {
		var m *mproto.Message
		select {
		case <-p.closeQ:
			break outer
		case m = <-p.sendQ.DequeueC():
		}

		if err := pp.SendMsg(m); err != nil {
			m.Free()
			break
		}
	}
	_ = pp.Close()
	s.wg.Done()
}

func (p *Pipe) receiver() {
	s := p.proto
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
			continue outer
		}

		m.Header = append(m.Header, m.Body[:protocol.DefaultHeaderLength]...)
		h := protocol.Header{Data: m.Header}
		if h.Flag() != protocol.DefaultFlag {
			m.Free()
			break
		}

		s.Lock()
		rq := s.recvQ
		zq := s.sizeQ
		sq := p.sendQ
		cq := p.closeQ
		s.Unlock()

		m.Body = m.Body[protocol.DefaultHeaderLength:]
		// register event and response register remote
		if h.IsRegisterEvent() {
			p.remote = h.Src()
			p.event = h.Dest()
			m.Clone()
			h.SetDest(p.remote)
			h.SetSrc(pp.ID())
			h.SetSignalling(protocol.SignallingControl | protocol.SignallingRegisterRemote)
			select {
			case sq.EnqueueC() <- m:
			case <-cq:
				m.Free()
			default:
				m.Free()
			}
			m.Free()
			continue
		}

		// response heart
		if h.IsHeart() {
			m.Clone()
			h.SetDest(h.Src())
			h.SetSrc(pp.ID())
			select {
			case sq.EnqueueC() <- m:
			case <-cq:
				m.Free()
			default:
				m.Free()
			}
			m.Free()
			continue
		}

		h.SetSrc(pp.ID())
		entry := recvQEntry{m: m, p: p}
		select {
		case rq <- entry:
		case <-cq:
			m.Free()
		case <-zq:
			m.Free() // discard this one
		}
	}
	_ = pp.Close()
	s.wg.Done()
}
