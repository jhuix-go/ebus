/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package protocol

import (
	`sync`

	`go.nanomsg.org/mangos/v3`
	`go.nanomsg.org/mangos/v3/protocol`
)

type socket struct {
	sync.Mutex
	proto Protocol
	s     protocol.Socket
}

// Info returns information about the protocol (numbers and names)
// and peer protocol.
func (s *socket) Info() mangos.ProtocolInfo {
	return s.s.Info()
}

// Close closes the open Socket.  Further operations on the socket
// will return ErrClosed.
func (s *socket) Close() error {
	return s.s.Close()
}

// Send puts the message on the outbound send queue.  It blocks
// until the message can be queued, or the send deadline expires.
// If a queued message is later dropped for any reason,
// there will be no notification back to the application.
func (s *socket) Send(b []byte) error {
	return s.s.Send(b)
}

// Recv receives a complete message.  The entire message is received.
func (s *socket) Recv() ([]byte, error) {
	return s.s.Recv()
}

// SendMsg puts the message on the outbound send.  It works like Send,
// but allows the caller to supply message headers.  AGAIN, the Socket
// ASSUMES OWNERSHIP OF THE MESSAGE.
func (s *socket) SendMsg(m *protocol.Message) error {
	return s.s.SendMsg(m)
}

// RecvMsg receives a complete message, including the message header,
// which is useful for protocols in raw mode.
func (s *socket) RecvMsg() (*protocol.Message, error) {
	return s.s.RecvMsg()
}

// Dial connects a remote endpoint to the Socket.  The function
// returns immediately, and an asynchronous goroutine is started to
// establish and maintain the connection, reconnecting as needed.
// If the address is invalid, then an error is returned.
func (s *socket) Dial(addr string) error {
	return s.s.Dial(addr)
}

func (s *socket) DialOptions(addr string, options map[string]interface{}) error {
	return s.s.DialOptions(addr, options)
}

// NewDialer returns a Dialer object which can be used to get
// access to the underlying configuration for dialing.
func (s *socket) NewDialer(addr string, options map[string]interface{}) (mangos.Dialer, error) {
	return s.s.NewDialer(addr, options)
}

// Listen connects a local endpoint to the Socket.  Remote peers
// may connect (e.g. with Dial) and will each be "connected" to
// the Socket.  The accepter logic is run in a separate goroutine.
// The only error possible is if the address is invalid.
func (s *socket) Listen(addr string) error {
	return s.s.Listen(addr)
}

func (s *socket) ListenOptions(addr string, options map[string]interface{}) error {
	return s.s.ListenOptions(addr, options)
}

func (s *socket) NewListener(addr string, options map[string]interface{}) (mangos.Listener, error) {
	return s.s.NewListener(addr, options)
}

// GetOption is used to retrieve an option for a socket.
func (s *socket) GetOption(name string) (interface{}, error) {
	return s.s.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (s *socket) SetOption(name string, value interface{}) error {
	return s.s.SetOption(name, value)
}

// OpenContext creates a new Context.  If a protocol does not
// support separate contexts, this will return an error.
func (s *socket) OpenContext() (mangos.Context, error) {
	return s.s.OpenContext()
}

func (s *socket) SetPipeEventHook(h PipeEventHook) {
	s.proto.SetPipeEventHook(h)
}

// MakeSocket creates a Socket on top of a Protocol.
func MakeSocket(proto Protocol, hook mangos.PipeEventHook) Socket {
	s := &socket{proto: proto, s: protocol.MakeSocket(proto)}
	s.s.SetPipeEventHook(hook)
	return s
}
