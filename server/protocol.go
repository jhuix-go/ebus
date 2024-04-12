/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package server

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"go.nanomsg.org/mangos/v3"
	mproto "go.nanomsg.org/mangos/v3/protocol"

	"github.com/jhuix-go/ebus/pkg/log"

	"github.com/jhuix-go/ebus/pkg/queue"
	"github.com/jhuix-go/ebus/protocol"
)

func randPipe(m []*Pipe) *Pipe {
	if len(m) == 0 {
		return nil
	}

	if len(m) == 1 {
		return m[0]
	}

	n := rand.IntN(len(m))
	return m[n]
}

func hashPipe(hash uint64, m []*Pipe) *Pipe {
	if len(m) == 0 {
		return nil
	}

	if len(m) == 1 {
		return m[0]
	}

	n := hash % uint64(len(m))
	return m[n]
}

type Protocol struct {
	sync.RWMutex
	closed     bool
	closeQ     chan struct{}
	sizeQ      chan struct{}
	pipes      map[uint32]*Pipe
	eventPipes map[uint32][]*Pipe
	recvQLen   int
	sendQLen   int
	recvExpire time.Duration
	recvQ      chan recvQEntry
	wg         sync.WaitGroup
	hook       protocol.PipeEventHook
}

// Protocol identity information.
const (
	Self     = protocol.ProtoEventBus
	Peer     = protocol.ProtoEvent
	SelfName = "ebus"
	PeerName = "event"
)

var (
	nilQ <-chan time.Time
)

const defaultQLen = 128

func (s *Protocol) SendMsg(m *mproto.Message) error {
	s.RLock()
	defer s.RUnlock()
	if s.closed {
		return mproto.ErrClosed
	}

	if len(m.Header) != protocol.DefaultHeaderLength && len(m.Header) != protocol.DefaultEventHeaderLength {
		return protocol.ErrBadHeader
	}

	h := protocol.Header{Data: m.Header}
	src := h.Src()
	dest := h.Dest()
	if src == 0 {
		return mproto.ErrNoPeers
	}

	if dest > 0 {
		switch h.SignallingType() {
		case protocol.SignallingEvent: // dest is event
			if pipes, ok := s.eventPipes[dest]; ok {
				var p *Pipe
				if h.HasHash() {
					hash := h.Hash()
					p = hashPipe(hash, pipes)
				} else {
					p = randPipe(pipes)
				}
				if p != nil {
					m.Clone()
					h.SetSignallingType(protocol.SignallingAssign)
					h.SetDest(p.p.ID())
					h.SetHeaderLength(protocol.DefaultHeaderLength)
					m.Header = m.Header[:protocol.DefaultHeaderLength]
					select {
					case p.sendQ.EnqueueC() <- m:
					default:
						// back-pressure, but we do not exert
						m.Free()
					}
					m.Free()
					return nil
				}
			}
		case protocol.SignallingAssign, protocol.SignallingControl: // dest is pipe
			if src == dest {
				return mproto.ErrCanceled
			}

			if p, ok := s.pipes[dest]; ok {
				m.Clone()
				select {
				case p.sendQ.EnqueueC() <- m:
				default:
					// back-pressure, but we do not exert
					m.Free()
				}
				m.Free()
				return nil
			}
		}

		return mproto.ErrNoPeers
	}

	// broadcast send
	for _, p := range s.pipes {
		if p.p.ID() == src {
			continue
		}

		m.Clone()
		select {
		case p.sendQ.EnqueueC() <- m:
		default:
			// back-pressure, but we do not exert
			m.Free()
		}
	}
	m.Free()
	return nil
}

func (s *Protocol) RecvMsg() (*mproto.Message, error) {
	for {
		s.RLock()
		rq := s.recvQ
		cq := s.closeQ
		zq := s.sizeQ
		s.RUnlock()

		select {
		case <-cq:
			return nil, mproto.ErrClosed
		case <-zq:
			continue
		case entry := <-rq:
			m, p := entry.m, entry.p
			h := protocol.Header{Data: m.Header}
			// control signalling be not transmit
			if h.SignallingType() == protocol.SignallingControl {
				if h.IsRegisterEvent() {
					_ = s.addEventPipe(p.Event(), p)
				}
				m.Free()
				continue
			}

			return m, nil
		}
	}
}

func (s *Protocol) SetOption(name string, value interface{}) error {
	switch name {
	case mproto.OptionRecvDeadline:
		if v, ok := value.(time.Duration); ok {
			s.Lock()
			s.recvExpire = v
			s.Unlock()
			return nil
		}
		return mproto.ErrBadValue

	case mproto.OptionWriteQLen:
		if v, ok := value.(int); ok && v >= 0 {
			s.Lock()
			s.sendQLen = v
			s.Unlock()
			return nil
		}
		return mproto.ErrBadValue

	case mproto.OptionReadQLen:
		if v, ok := value.(int); ok && v >= 0 {
			newQ := make(chan recvQEntry, v)
			sizeQ := make(chan struct{})
			s.Lock()
			s.recvQLen = v
			s.recvQ = newQ
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
	case mproto.OptionRecvDeadline:
		s.RLock()
		v := s.recvExpire
		s.RUnlock()
		return v, nil
	case mproto.OptionWriteQLen:
		s.RLock()
		v := s.sendQLen
		s.RUnlock()
		return v, nil
	case mproto.OptionReadQLen:
		s.RLock()
		v := s.recvQLen
		s.RUnlock()
		return v, nil
	}

	return nil, mproto.ErrBadOption
}

func (s *Protocol) addEventPipe(event uint32, p *Pipe) error {
	s.Lock()
	if s.closed {
		s.Unlock()
		return mproto.ErrClosed
	}

	p.event = event
	pipes, ok := s.eventPipes[event]
	if !ok {
		pipes = make([]*Pipe, 0, 1)
	}
	pipes = append(pipes, p)
	s.eventPipes[event] = pipes

	ph := s.hook
	s.Unlock()

	if ph != nil {
		ph(protocol.PipeEventRegistered, p)
	}
	return nil
}

func (s *Protocol) String() string {
	info := s.Info()
	return fmt.Sprintf("{\"self\":%d,\"self_name\":\"%s\",\"peer\":%d,\"peer_name\":\"%s\"}",
		info.Self, info.SelfName, info.Peer, info.PeerName)
}

func (s *Protocol) registerEvent(p *Pipe) {
	m := mangos.NewMessage(0)
	defer m.Free()
	m.Header = protocol.PutHeader(m.Header, protocol.PipeEbus,
		protocol.SignallingControl|protocol.SignallingRegisterEvent, p.ID(), 0)
	if err := p.SendMsg(m); err != nil {
		log.Errorf("%s, register event error: %s", s.String(), err)
	}
}

func (s *Protocol) AddPipe(pp mproto.Pipe) error {
	s.Lock()
	if s.closed {
		s.Unlock()
		return mproto.ErrClosed
	}

	var p *Pipe
	data := pp.GetPrivate()
	if data == nil {
		p = &Pipe{
			p:          pp,
			proto:      s,
			recvExpire: s.recvExpire,
			closeQ:     make(chan struct{}),
			sendQ:      queue.NewQueueWithSize[*mproto.Message](s.sendQLen, s.sendQLen),
		}
		p.add()
		pp.SetPrivate(p)
	} else {
		p = data.(*Pipe)
	}
	s.pipes[pp.ID()] = p
	s.Unlock()

	s.registerEvent(p)
	p.add()
	s.wg.Add(1)
	go p.sender()
	p.add()
	s.wg.Add(1)
	go p.receiver()
	return nil
}

func (s *Protocol) RemovePipe(pp mproto.Pipe) {
	if p, ok := pp.GetPrivate().(*Pipe); ok {
		s.Lock()
		delete(s.pipes, pp.ID())
		if pipes, ok := s.eventPipes[p.event]; ok {
			// delete(pipes, pp.ID())
			pipes = slices.DeleteFunc(pipes, func(pe *Pipe) bool {
				return pe == p
			})
			if len(pipes) == 0 {
				delete(s.eventPipes, p.event)
			} else {
				s.eventPipes[p.event] = pipes
			}
		}
		s.Unlock()
		close(p.closeQ)
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

func (s *Protocol) closeAllPipes() {
	s.RLock()
	for _, p := range s.pipes {
		go p.Close()
	}
	s.RUnlock()
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
	s.closeAllPipes()
	s.wg.Wait()
	return nil
}

func (s *Protocol) SetPipeEventHook(v protocol.PipeEventHook) {
	s.Lock()
	s.hook = v
	s.Unlock()
}

func (s *Protocol) Pipe(id uint32) protocol.Pipe {
	s.RLock()
	p, _ := s.pipes[id]
	s.RUnlock()
	return p
}

func (s *Protocol) RangePipes(f func(uint32, protocol.Pipe) bool) {
	s.RLock()
	for id, p := range s.pipes {
		if !f(id, p) {
			break
		}
	}
	s.RUnlock()
}

func (s *Protocol) pipeEventHook(pe mangos.PipeEvent, mp mangos.Pipe) {
	s.RLock()
	ph := s.hook
	s.RUnlock()
	if pp, ok := mp.(mproto.Pipe); ok {
		switch pe {
		case mangos.PipeEventAttaching:
			p := &Pipe{
				p:          pp,
				proto:      s,
				recvExpire: s.recvExpire,
				closeQ:     make(chan struct{}),
				sendQ:      queue.NewQueueWithSize[*mproto.Message](s.sendQLen, s.sendQLen),
			}
			p.add()
			pp.SetPrivate(p)
			if ph != nil {
				ph(pe, p)
			}
		case mangos.PipeEventDetached:
			p := pp.GetPrivate().(*Pipe)
			if p != nil {
				if ph != nil {
					ph(pe, p)
				}
				pp.SetPrivate(nil)
				p.release()
			}
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
		pipes:      make(map[uint32]*Pipe),
		eventPipes: make(map[uint32][]*Pipe),
		closeQ:     make(chan struct{}),
		sizeQ:      make(chan struct{}),
		recvQ:      make(chan recvQEntry, defaultQLen),
		sendQLen:   defaultQLen,
		recvQLen:   defaultQLen,
	}
	return s
}
