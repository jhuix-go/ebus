/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package client

import (
	"sync"
	"time"

	`go.nanomsg.org/mangos/v3`
	mproto "go.nanomsg.org/mangos/v3/protocol"

	`github.com/jhuix-go/ebus/protocol`
)

// Protocol identity information.
const (
	Self     = protocol.ProtoEvent
	Peer     = protocol.ProtoEventBus
	SelfName = "event"
	PeerName = "ebus"
)

type Protocol struct {
	closed      bool
	closeQ      chan struct{}
	sizeQ       chan struct{}
	recvQ       chan recvQEntry
	pipes       map[uint32]*Pipe
	events      map[mangos.Dialer]uint32
	recvExpire  time.Duration
	sendExpire  time.Duration
	heartExpire time.Duration
	recvTimer   *time.Timer
	sendTimer   *time.Timer
	recvQLen    int
	sendQLen    int
	bestEffort  bool
	reconnect   bool
	wg          sync.WaitGroup
	hook        protocol.PipeEventHook
	sync.Mutex
}

type recvQEntry struct {
	m *mproto.Message
	p *Pipe
}

var (
	nilQ    <-chan time.Time
	closedQ chan time.Time
)

func init() {
	closedQ = make(chan time.Time)
	close(closedQ)
}

const defaultQLen = 128

func (s *Protocol) SendMsg(m *mproto.Message) error {
	timeQ := nilQ
	s.Lock()
	if s.closed {
		s.Unlock()
		return mproto.ErrClosed
	}

	if len(m.Header) < protocol.DefaultHeaderLength {
		s.Unlock()
		return protocol.ErrBadHeader
	}

	h := protocol.Header{Data: m.Header}
	id := h.Src()
	if id == 0 {
		s.Unlock()
		return mproto.ErrNoPeers
	}

	p, ok := s.pipes[id]
	if !ok {
		s.Unlock()
		return mproto.ErrNoPeers
	}

	bestEffort := s.bestEffort
	if bestEffort {
		timeQ = closedQ
	} else if s.sendExpire > 0 {
		if s.sendTimer == nil {
			s.sendTimer = time.NewTimer(s.sendExpire)
		} else {
			s.sendTimer.Reset(s.sendExpire)
		}
		defer s.sendTimer.Stop()
		timeQ = s.sendTimer.C
	}
	sizeQ := s.sizeQ
	closeQ := s.closeQ
	s.Unlock()

	select {
	case <-closeQ:
		return mproto.ErrClosed
	case <-p.closeQ:
		return mproto.ErrClosed
	case <-timeQ:
		if bestEffort {
			m.Free()
			return nil
		}

		return mproto.ErrSendTimeout

	case <-sizeQ:
		m.Free()
		return nil

	case p.sendQ <- m:
		return nil
	}
}

func (s *Protocol) RecvMsg() (*mproto.Message, error) {
	defer func() {
		if s.recvTimer != nil {
			s.recvTimer.Stop()
		}
	}()
	for {
		timeQ := nilQ
		s.Lock()
		if s.recvExpire > 0 {
			if s.recvTimer == nil {
				s.recvTimer = time.NewTimer(s.recvExpire)
			} else {
				s.recvTimer.Reset(s.recvExpire)
			}
			timeQ = s.recvTimer.C
		}
		closeQ := s.closeQ
		recvQ := s.recvQ
		sizeQ := s.sizeQ
		hook := s.hook
		s.Unlock()
		select {
		case <-closeQ:
			return nil, mproto.ErrClosed
		case <-timeQ:
			return nil, mproto.ErrRecvTimeout
		case entry := <-recvQ:
			m, p := entry.m, entry.p
			h := protocol.Header{Data: m.Header}
			if h.IsHeart() {
				m.Free()
				continue
			}

			if h.IsRegisterRemote() {
				hook(protocol.PipeEventRegistered, p)
				m.Free()
				continue
			}

			return m, nil
		case <-sizeQ:
		}
	}
}

func (s *Protocol) SetOption(name string, value interface{}) error {
	switch name {
	case protocol.OptionHeartTime:
		if v, ok := value.(time.Duration); ok {
			s.Lock()
			s.heartExpire = v
			s.Unlock()
			return nil
		}
		return mproto.ErrBadValue

	case protocol.OptionReconnect:
		if v, ok := value.(bool); ok {
			s.Lock()
			s.reconnect = v
			s.Unlock()
			return nil
		}
		return mproto.ErrBadValue

	case mproto.OptionBestEffort:
		if v, ok := value.(bool); ok {
			s.Lock()
			s.bestEffort = v
			s.Unlock()
			return nil
		}
		return mproto.ErrBadValue

	case mproto.OptionRecvDeadline:
		if v, ok := value.(time.Duration); ok {
			s.Lock()
			s.recvExpire = v
			s.Unlock()
			return nil
		}
		return mproto.ErrBadValue

	case mproto.OptionSendDeadline:
		if v, ok := value.(time.Duration); ok {
			s.Lock()
			s.sendExpire = v
			s.Unlock()
			return nil
		}
		return mproto.ErrBadValue

	case mproto.OptionReadQLen:
		if v, ok := value.(int); ok && v >= 0 {
			recvQ := make(chan recvQEntry, v)
			sizeQ := make(chan struct{})
			s.Lock()
			s.recvQLen = v
			s.recvQ = recvQ
			sizeQ, s.sizeQ = s.sizeQ, sizeQ
			s.Unlock()
			close(sizeQ)

			return nil
		}
		return mproto.ErrBadValue

	case mproto.OptionWriteQLen:
		if v, ok := value.(int); ok && v >= 0 {
			sizeQ := make(chan struct{})
			s.Lock()
			s.sendQLen = v
			sizeQ, s.sizeQ = s.sizeQ, sizeQ
			s.Unlock()
			close(sizeQ)

			return nil
		}
		return mproto.ErrBadValue

	}

	return mproto.ErrBadOption
}

func (s *Protocol) GetOption(option string) (interface{}, error) {
	switch option {
	case mproto.OptionRaw:
		return false, nil
	case protocol.OptionReconnect:
		s.Lock()
		v := s.reconnect
		s.Unlock()
		return v, nil
	case mproto.OptionBestEffort:
		s.Lock()
		v := s.bestEffort
		s.Unlock()
		return v, nil
	case mproto.OptionRecvDeadline:
		s.Lock()
		v := s.recvExpire
		s.Unlock()
		return v, nil
	case mproto.OptionSendDeadline:
		s.Lock()
		v := s.sendExpire
		s.Unlock()
		return v, nil
	case mproto.OptionReadQLen:
		s.Lock()
		v := s.recvQLen
		s.Unlock()
		return v, nil
	case mproto.OptionWriteQLen:
		s.Lock()
		v := s.sendQLen
		s.Unlock()
		return v, nil
	}

	return nil, mproto.ErrBadOption
}

func (s *Protocol) AddPipe(pp mproto.Pipe) error {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return mproto.ErrClosed
	}

	var p *Pipe
	data := pp.GetPrivate()
	if data == nil {
		p = &Pipe{
			p:           pp,
			s:           s,
			closeQ:      make(chan struct{}),
			sendQ:       make(chan *mproto.Message, s.sendQLen),
			heartExpire: s.heartExpire,
			hook:        s.hook,
		}
		pp.SetPrivate(p)
	} else {
		p = data.(*Pipe)
	}
	s.pipes[pp.ID()] = p
	go p.receiver()
	go p.sender()
	return nil
}

func (s *Protocol) RemovePipe(pp mproto.Pipe) {
	p := pp.GetPrivate().(*Pipe)
	close(p.closeQ)
	s.Lock()
	delete(s.pipes, p.p.ID())
	ph := s.hook
	var dialer mangos.Dialer
	if mp, ok := pp.(mangos.Pipe); ok {
		if !s.reconnect {
			dialer = mp.Dialer()
			delete(s.events, dialer)
		} else {
			s.events[mp.Dialer()] = p.event
		}
	}
	s.Unlock()

	if ph != nil {
		ph(mangos.PipeEventDetached, p)
	}
	pp.SetPrivate(nil)
	p.release()
	if dialer != nil {
		_ = dialer.Close()
	}
}

func (s *Protocol) OpenContext() (mproto.Context, error) {
	return nil, mproto.ErrProtoOp
}

func (*Protocol) Info() mproto.Info {
	return mproto.Info{
		Self:     Self,
		Peer:     Peer,
		SelfName: SelfName,
		PeerName: PeerName,
	}
}

func (s *Protocol) Close() error {
	s.Lock()
	if s.closed {
		s.Unlock()
		return mproto.ErrClosed
	}
	s.closed = true
	s.Unlock()
	close(s.closeQ)
	return nil
}

func (s *Protocol) SetPipeEventHook(v protocol.PipeEventHook) {
	s.Lock()
	s.hook = v
	s.Unlock()
}

func (s *Protocol) WaitAllPipe() {
	s.wg.Wait()
}

func (s *Protocol) Pipe(id uint32) protocol.Pipe {
	s.Lock()
	p, _ := s.pipes[id]
	s.Unlock()
	return p
}

func (s *Protocol) RangePipes(f func(uint32, protocol.Pipe) bool) {
	s.Lock()
	for id, p := range s.pipes {
		if !f(id, p) {
			break
		}
	}
	s.Unlock()
}

func (s *Protocol) pipeEventHook(pe mangos.PipeEvent, mp mangos.Pipe) {
	s.Lock()
	ph := s.hook
	s.Unlock()
	if pp, ok := mp.(mproto.Pipe); ok {
		switch pe {
		case mangos.PipeEventAttaching:
			s.Lock()
			id, _ := s.events[mp.Dialer()]
			s.Unlock()
			p := &Pipe{
				p:           pp,
				s:           s,
				event:       id,
				closeQ:      make(chan struct{}),
				sendQ:       make(chan *mproto.Message, s.sendQLen),
				heartExpire: s.heartExpire,
				hook:        s.hook,
			}
			pp.SetPrivate(p)
			if ph != nil {
				ph(pe, p)
			}
		case mangos.PipeEventDetached:
			p := pp.GetPrivate().(*Pipe)
			if ph != nil {
				ph(pe, p)
			}
			p.release()
			pp.SetPrivate(nil)
		default:
			if ph != nil {
				ph(pe, pp.GetPrivate().(*Pipe))
			}
		}
	}
}

// NewProtocol returns a new protocol implementation.
func NewProtocol() *Protocol {
	s := &Protocol{
		pipes:    make(map[uint32]*Pipe),
		events:   make(map[mangos.Dialer]uint32),
		closeQ:   make(chan struct{}),
		sizeQ:    make(chan struct{}),
		recvQ:    make(chan recvQEntry, defaultQLen),
		recvQLen: defaultQLen,
		sendQLen: defaultQLen,
	}
	return s
}
