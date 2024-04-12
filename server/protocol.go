/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package server

import (
	"fmt"
	"math/rand/v2"
	`runtime/debug`
	"slices"
	"sync"
	`sync/atomic`
	"time"

	"go.nanomsg.org/mangos/v3"
	mproto "go.nanomsg.org/mangos/v3/protocol"

	"github.com/jhuix-go/ebus/pkg/log"

	"github.com/jhuix-go/ebus/pkg/queue"
	"github.com/jhuix-go/ebus/protocol"
)

// func randMapPipe(m map[uint32]*Pipe) *Pipe {
// 	r := rand.Intn(len(m))
// 	for _, v := range m {
// 		if r == 0 {
// 			return v
// 		}
//
// 		r--
// 	}
// 	return nil
// }

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

type aggregateQ struct {
	sync.RWMutex
	trace    bool
	recvQLen int
	running  atomic.Bool
	closeQ   chan struct{}
	sizeQ    chan struct{}
	recvQ    chan recvQEntry
}

func newAggregateQ() *aggregateQ {
	return &aggregateQ{
		recvQLen: defaultQLen,
		closeQ:   make(chan struct{}),
		sizeQ:    make(chan struct{}),
		recvQ:    make(chan recvQEntry, defaultQLen),
	}
}

type Protocol struct {
	sync.RWMutex
	closed     bool
	pipes      map[uint32]*Pipe
	eventPipes map[uint32][]*Pipe
	recvQLen   int
	sendQLen   int
	recvExpire time.Duration
	recvQS     []*aggregateQ
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

const (
	defaultQLen          = 128
	defaultExchangeLines = 4
)

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
				// p := randMapPipe(pipes)
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
					h.SetHeaderLength(protocol.DefaultHeaderLength)
					h.SetDest(p.p.ID())
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
	return nil, mproto.ErrProtoOp
}

func (s *Protocol) exchange(index int, q *aggregateQ) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("exchange %d panic error: %v, stack:\n %s", index, err, debug.Stack())
		} else {
			log.Infof("event bus exchange line %d exist", index)
		}
		q.running.Store(false)
		s.wg.Done()
	}()

	log.Infof("event bus exchange line %d starting...", index)
	for {
		q.RLock()
		rq := q.recvQ
		cq := q.closeQ
		zq := q.sizeQ
		q.RUnlock()

		select {
		case <-cq:
			return
		case <-zq:
			continue
		case entry := <-rq:
			m, p := entry.m, entry.p
			h := protocol.Header{Data: m.Header}
			// control signalling be not transmit
			if h.SignallingType() == protocol.SignallingControl || h.Dest() == protocol.PipeEbus {
				if h.IsRegisterEvent() {
					_ = s.addEventPipe(p.Event(), p)
				}
				m.Free()
				continue
			}

			if q.trace {
				log.Debugf("exchange line %d recv message: event=%s header=%s data_size=%d",
					index, protocol.EventName(protocol.PipeEvent(m.Pipe)), protocol.StringHeader(m.Header), len(m.Body))
			}

			// router send
			if err := s.SendMsg(m); err != nil {
				m.Free()
				log.Errorf("exchange line %d send message: event=%s header=%s error=%v",
					index, protocol.EventName(protocol.PipeEvent(m.Pipe)), protocol.StringHeader(m.Header), err)
			}
		}
	}
}

func (s *Protocol) Exchange() {
	for i, q := range s.recvQS {
		if q.running.CompareAndSwap(false, true) {
			s.wg.Add(1)
			go s.exchange(i+1, q)
		}
	}
}

func (s *Protocol) RecvQ(hash uint32) (sizeQ chan struct{}, recvQ chan recvQEntry) {
	index := hash % uint32(len(s.recvQS))
	q := s.recvQS[index]
	q.RLock()
	defer q.RUnlock()
	return q.sizeQ, q.recvQ
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
			s.Lock()
			s.recvQLen = v
			s.Unlock()
			for _, q := range s.recvQS {
				newQ := make(chan recvQEntry, v)
				sizeQ := make(chan struct{})
				q.Lock()
				q.recvQLen = v
				q.recvQ = newQ
				sizeQ, q.sizeQ = q.sizeQ, sizeQ
				q.Unlock()
				close(sizeQ)
			}
			return nil
		}
		return mproto.ErrBadValue

	case protocol.OptionTraceMessage:
		if v, ok := value.(bool); ok {
			for _, q := range s.recvQS {
				q.Lock()
				q.trace = v
				q.Unlock()
			}
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
		s.Lock()
		v := s.recvExpire
		s.Unlock()
		return v, nil
	case mproto.OptionWriteQLen:
		s.Lock()
		v := s.sendQLen
		s.Unlock()
		return v, nil
	case mproto.OptionReadQLen:
		s.Lock()
		v := s.recvQLen
		s.Unlock()
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
		// pipes = make(map[uint32]*Pipe)
		pipes = make([]*Pipe, 0, 1)
		// s.eventPipes[event] = pipes
	}
	// pipes[p.p.ID()] = p
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
	s.wg.Add(1)
	p.add()
	go p.sender()
	s.wg.Add(1)
	p.add()
	go p.receiver()
	return nil
}

func (s *Protocol) RemovePipe(pp mproto.Pipe) {
	if p, ok := pp.GetPrivate().(*Pipe); ok {
		s.Lock()
		delete(s.pipes, pp.ID())
		if pipes, ok := s.eventPipes[p.event]; ok {
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

	for _, q := range s.recvQS {
		close(q.closeQ)
	}

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
func NewProtocol(exchangeLines int) *Protocol {
	if exchangeLines <= 0 {
		exchangeLines = defaultExchangeLines
	}
	s := &Protocol{
		pipes:      make(map[uint32]*Pipe),
		eventPipes: make(map[uint32][]*Pipe),
		recvQS:     make([]*aggregateQ, exchangeLines),
		sendQLen:   defaultQLen,
		recvQLen:   defaultQLen,
	}
	for i := 0; i < exchangeLines; i++ {
		s.recvQS[i] = newAggregateQ()
	}
	return s
}
