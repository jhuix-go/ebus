/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package server

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/transport/all"

	"github.com/jhuix-go/ebus/pkg/log"
	"github.com/jhuix-go/ebus/protocol"
)

type Server struct {
	cfg    *Config
	proto  *Protocol
	socket protocol.Socket
	wg     sync.WaitGroup
}

func NewServer(cfg *Config) *Server {
	config := defaultServerConfig
	if cfg != nil {
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
		config.TraceMessage = cfg.TraceMessage
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
	// if cfg.WriteTimeout > 0 {
	// 	_ = socket.SetOption(mangos.OptionSendDeadline, cfg.WriteTimeout)
	// }
	return &Server{
		cfg:    cfg,
		proto:  proto,
		socket: socket,
	}
}

func (s *Server) SetTraceMessage(trace bool) {
	s.cfg.TraceMessage = trace
}

func (s *Server) SetSendSize(v int) {
	if v > 0 && s.cfg.SendChanSize != v {
		s.cfg.SendChanSize = v
		if s.socket != nil {
			_ = s.socket.SetOption(mangos.OptionWriteQLen, v)
		}
	}
}

func (s *Server) SetRecvSize(v int) {
	if v > 0 && s.cfg.RecvChanSize != v {
		s.cfg.RecvChanSize = v
		if s.socket != nil {
			_ = s.socket.SetOption(mangos.OptionReadQLen, v)
		}
	}
}

func (s *Server) SetReadTimeout(v time.Duration) {
	if v > 0 && s.cfg.ReadTimeout != v {
		s.cfg.ReadTimeout = v
		if s.socket != nil {
			_ = s.socket.SetOption(mangos.OptionRecvDeadline, v)
		}
	}
}

// func (s *Server) SetWriteTimeout(v time.Duration) {
// 	if v > 0 && s.cfg.WriteTimeout != v {
// 		s.cfg.WriteTimeout = v
// 		if s.socket != nil {
// 			_ = s.socket.SetOption(mangos.OptionSendDeadline, v)
// 		}
// 	}
// }

func (s *Server) pipeEventHook(pe mangos.PipeEvent, pp protocol.Pipe) interface{} {
	switch pe {
	case mangos.PipeEventAttaching:
		log.Infof("connection attaching: %s(%s)<->%s(%d)",
			pp.LocalAddr(), protocol.InetNtoA(pp.ID()), pp.RemoteAddr(), pp.RemoteID())
	case mangos.PipeEventAttached:
		log.Infof("connection attached: %s(%s)<->%s(%d)",
			pp.LocalAddr(), protocol.InetNtoA(pp.ID()), pp.RemoteAddr(), pp.RemoteID())
	case mangos.PipeEventDetached:
		log.Infof("connection closed: %s(%s)<->%s(%d)",
			pp.LocalAddr(), protocol.InetNtoA(pp.ID()), pp.RemoteAddr(), pp.RemoteID())
	case protocol.PipeEventRegistered:
		log.Infof("connection registered as %s event service: %s(%s)<->%s(%d)",
			protocol.EventName(pp.Event()), pp.LocalAddr(), protocol.InetNtoA(pp.ID()), pp.RemoteAddr(), pp.RemoteID())
	default:
	}
	return nil
}

func (s *Server) Listen(addr string) error {
	if len(addr) == 0 {
		addr = s.cfg.Address
	}
	if len(addr) == 0 {
		return protocol.ErrBadAddress
	}

	s.socket.SetPipeEventHook(s.pipeEventHook)
	if err := s.socket.Listen(addr); err != nil {
		log.Errorf("listen %s error: %v", s.cfg.Address, err)
		return err
	}

	log.Infof("listen %s succeed", addr)
	return nil
}

func (s *Server) SendEvent(src, event uint32, hash uint64, data []byte) error {
	m := mangos.NewMessage(len(data))
	if hash != 0 {
		m.Header = protocol.PutHashHeader(m.Header, src, event, hash)
	} else {
		m.Header = protocol.PutHeader(m.Header, src, protocol.SignallingEvent, event)
	}
	m.Body = append(m.Body, data...)
	if err := s.socket.SendMsg(m); err != nil {
		m.Free()
		log.Errorf("%d<->%s, send error: %s", src, protocol.EventName(event), err)
		return err
	}

	return nil
}

func (s *Server) Send(src, dest uint32, data []byte) error {
	m := mangos.NewMessage(len(data))
	m.Header = protocol.PutHeader(m.Header, src, protocol.SignallingAssign, dest)
	m.Body = append(m.Body, data...)
	if err := s.socket.SendMsg(m); err != nil {
		m.Free()
		log.Errorf("%d<->%d, send error: %s", src, dest, err)
		return err
	}

	return nil
}

func (s *Server) Broadcast(data []byte) error {
	m := mangos.NewMessage(len(data))
	m.Header = protocol.PutHeader(m.Header, protocol.PipeEbus, protocol.SignallingAssign, 0)
	if err := s.socket.SendMsg(m); err != nil {
		m.Free()
		log.Errorf("broadcast error: %s", err)
		return err
	}

	return nil
}

func (s *Server) String() string {
	info := s.socket.Info()
	return fmt.Sprintf("%s: {\"self\":%d,\"self_name\":\"%s\",\"peer\":%d,\"peer_name\":\"%s\"}",
		s.cfg.Address, info.Self, info.SelfName, info.Peer, info.PeerName)
}

func (s *Server) Pipe(id uint32) protocol.Pipe {
	return s.proto.Pipe(id)
}

func (s *Server) RangePipes(f func(uint32, protocol.Pipe) bool) {
	s.proto.RangePipes(f)
}

func (s *Server) Serve() {
	s.wg.Add(1)
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recv panic error: %v, stack:\n %s", err, debug.Stack())
		} else {
			log.Infof("event bus closed")
		}
		_ = s.Close()
		s.wg.Done()
	}()

	for {
		m, err := s.socket.RecvMsg()
		if err != nil {
			log.Errorf("recv message error: %v", err)
			if errors.Is(err, mangos.ErrClosed) {
				break
			}

			continue
		}

		if s.cfg.TraceMessage {
			log.Debugf("recv message: event=%s header=%s data_size=%d",
				protocol.EventName(protocol.PipeEvent(m.Pipe)), protocol.StringHeader(m.Header), len(m.Body))
		}
		h := protocol.Header{Data: m.Header}
		if h.Dest() == protocol.PipeEbus {
			m.Free()
			continue
		}

		// router send
		if err = s.socket.SendMsg(m); err != nil {
			m.Free()
			log.Errorf("send message: event=%s header=%s error=%v",
				protocol.EventName(protocol.PipeEvent(m.Pipe)), protocol.StringHeader(m.Header), err)
		}
	}
}

func (s *Server) Close() error {
	return s.socket.Close()
}

func (s *Server) Stop() {
	_ = s.Close()
	s.proto.WaitAllPipe()
	s.wg.Wait()
	log.Infof("event bus stopped")
}
