/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package server

import (
	"net"
	`sync/atomic`
	"time"

	"go.nanomsg.org/mangos/v3"
	mproto "go.nanomsg.org/mangos/v3/protocol"

	`github.com/jhuix-go/ebus/pkg/log`
	"github.com/jhuix-go/ebus/pkg/queue"
	"github.com/jhuix-go/ebus/protocol"
)

type RecvQueue interface {
	RecvQ(hash uint64) (recvQ chan struct{}, sizeQ chan struct{})
}

type Pipe struct {
	p          mproto.Pipe
	proto      *Protocol
	event      uint32
	remote     uint32
	state      atomic.Int64
	closeQ     chan struct{}
	sendQ      *queue.Queue[*mproto.Message]
	recvExpire time.Duration
	data       interface{}
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

func (p *Pipe) add() {
	p.state.Add(1)
}

func (p *Pipe) release() {
	if p.state.Add(-1) == 0 {
		p.data = nil
		p.proto = nil
		p.p = nil
	}
}

func (p *Pipe) sender() {
	s := p.proto
	pp := p.p
	log.Infof("pipe %s sender is sending...", protocol.InetNtoA(p.ID()))
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
	log.Infof("pipe %s sender is ended", protocol.InetNtoA(p.ID()))
	p.release()
	s.wg.Done()
}

func (p *Pipe) receiver() {
	s := p.proto
	pp := p.p
	log.Infof("pipe %s receiver is recving...", protocol.InetNtoA(p.ID()))
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

		headerLength := protocol.DefaultHeaderLength
		h := protocol.Header{Data: m.Body[:headerLength]}
		if h.Flag() != protocol.DefaultFlag {
			m.Free()
			break
		}

		headerLength = int(h.HeaderLength())
		if headerLength < protocol.DefaultHeaderLength || headerLength > protocol.MaxHeaderLength {
			m.Free()
			break
		}

		s.RLock()
		sq := p.sendQ
		cq := p.closeQ
		s.RUnlock()

		m.Header = m.Body[:headerLength]
		m.Body = m.Body[headerLength:]
		if h.IsRegisterEvent() { // register event and register remote
			p.remote = h.Src()
			p.event = h.Dest()
		} else if h.IsHeart() { // response heart
			m.Clone()
			h.SetDest(pp.ID())
			h.SetSrc(protocol.PipeEbus)
			select {
			case sq.EnqueueC() <- m:
			default:
				m.Free()
			}
			m.Free()
			continue
		}

		zq, rq := s.RecvQ(pp.ID())
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
	log.Infof("pipe %s receiver is ended...", protocol.InetNtoA(p.ID()))
	p.release()
	s.wg.Done()
}
