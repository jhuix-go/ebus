/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package protocol

import (
	`fmt`

	`go.nanomsg.org/mangos/v3`
	`go.nanomsg.org/mangos/v3/errors`
	`go.nanomsg.org/mangos/v3/protocol`
)

const (
	ProtoEventBus = 170
	ProtoEvent    = 171
)

const (
	ErrBadHeader = errors.ErrBadHeader
)

const (
	OptionHeartTime = "HEART_TIME"
	OptionReconnect = "RECONNECT"
)

const (
	PipeEbus = 0x65627573 // ebus
)

type Pipe interface {
	// Event returns event id
	Event() uint32

	// SetEvent is set event id
	SetEvent(uint32)

	// RemoteID is remote id
	RemoteID() uint32

	// LocalAddr is local address
	LocalAddr() string

	// RemoteAddr is remote address
	RemoteAddr() string

	// Pipe is mangos pipe
	Pipe() mangos.Pipe

	// ID returns a unique 31-bit value associated with this.
	// The value is unique for a given socket, at a given time.
	ID() uint32

	// Close does what you think.
	Close() error

	// Stop does with no reconnect
	Stop()

	// SendMsg sends a message.  On success, it returns nil. This is a
	// blocking call.
	SendMsg(*protocol.Message) error

	// RecvMsg receives a message.  It blocks until the message is
	// received.  On error, the pipe is closed and nil is returned.
	RecvMsg() *protocol.Message

	// SetPrivate is used to set protocol private data.
	SetPrivate(interface{})

	// GetPrivate returns the previously stored protocol private data.
	GetPrivate() interface{}
}

func Link(p Pipe) string {
	return fmt.Sprintf("%s(%d)<->%s(%d)", p.LocalAddr(), p.ID(), p.RemoteAddr(), p.RemoteID())
}

func LocalPipe(p mangos.Pipe) Pipe {
	if pp, ok := p.(protocol.Pipe); ok {
		if pipe, ok := pp.GetPrivate().(Pipe); ok {
			return pipe
		}
	}

	return nil
}

func PipeEvent(p mangos.Pipe) uint32 {
	pp := LocalPipe(p)
	if pp == nil {
		return 0
	}

	return pp.Event()
}

const (
	PipeEventRegistered = iota + mangos.PipeEventDetached + 1
	PipeEventHeartBeat
)

type PipeEventHook func(mangos.PipeEvent, Pipe) interface{}

type Protocol interface {
	protocol.Protocol
	SetPipeEventHook(PipeEventHook)
	WaitAllPipe()
}

type Socket interface {
	// Info returns information about the protocol (numbers and names)
	// and peer protocol.
	Info() mangos.ProtocolInfo

	// Close closes the open Socket.  Further operations on the socket
	// will return ErrClosed.
	Close() error

	// Send puts the message on the outbound send queue.  It blocks
	// until the message can be queued, or the send deadline expires.
	// If a queued message is later dropped for any reason,
	// there will be no notification back to the application.
	Send([]byte) error

	// Recv receives a complete message.  The entire message is received.
	Recv() ([]byte, error)

	// SendMsg puts the message on the outbound send.  It works like Send,
	// but allows the caller to supply message headers.  AGAIN, the Socket
	// ASSUMES OWNERSHIP OF THE MESSAGE.
	SendMsg(*protocol.Message) error

	// RecvMsg receives a complete message, including the message header,
	// which is useful for protocols in raw mode.
	RecvMsg() (*protocol.Message, error)

	// Dial connects a remote endpoint to the Socket.  The function
	// returns immediately, and an asynchronous goroutine is started to
	// establish and maintain the connection, reconnecting as needed.
	// If the address is invalid, then an error is returned.
	Dial(addr string) error

	DialOptions(addr string, options map[string]interface{}) error

	// NewDialer returns a Dialer object which can be used to get
	// access to the underlying configuration for dialing.
	NewDialer(addr string, options map[string]interface{}) (mangos.Dialer, error)

	// Listen connects a local endpoint to the Socket.  Remote peers
	// may connect (e.g. with Dial) and will each be "connected" to
	// the Socket.  The accepter logic is run in a separate goroutine.
	// The only error possible is if the address is invalid.
	Listen(addr string) error

	ListenOptions(addr string, options map[string]interface{}) error

	NewListener(addr string, options map[string]interface{}) (mangos.Listener, error)

	// GetOption is used to retrieve an option for a socket.
	GetOption(name string) (interface{}, error)

	// SetOption is used to set an option for a socket.
	SetOption(name string, value interface{}) error

	// OpenContext creates a new Context.  If a protocol does not
	// support separate contexts, this will return an error.
	OpenContext() (mangos.Context, error)

	// SetPipeEventHook sets a PipeEventHook function to be called when a
	// Pipe is added or removed from this socket (connect/disconnect).
	SetPipeEventHook(PipeEventHook)
}
