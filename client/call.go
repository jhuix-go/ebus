/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package client

import (
	`context`
	`errors`

	`go.nanomsg.org/mangos/v3`

	`github.com/jhuix-go/ebus/client/message`
	`github.com/jhuix-go/ebus/pkg/log`
	`github.com/jhuix-go/ebus/protocol`
)

// Call represents an active RPC.
type Call struct {
	ID            uint64
	ServicePath   string            // The name of the service and method to call.
	ServiceMethod string            // The name of the service and method to call.
	ReqMetadata   map[string]string // metadata
	ResMetadata   map[string]string
	Content       interface{}
	Args          interface{} // The argument to the function (*struct).
	Reply         interface{} // The reply from the function (*struct).
	Error         error       // After completion, the error status.
	Done          chan *Call  // Strobes when call is complete.
}

func (c *Call) done() {
	select {
	case c.Done <- c:
		// ok
	default:
		log.Debugf("discarding Call reply due to insufficient Done chan capacity")
	}
}

func (c *XClient) send(ctx context.Context, src, dest uint32, eventHash uint64, call *Call) {
	// meta := message.DecodeMetaContext(ctx)
	// if meta == nil {
	//	return
	// }

	// Register this call.
	c.Lock()
	if c.closed {
		call.Error = ErrShutdown
		c.Unlock()
		call.done()
		return
	}

	serializeType := message.SerializeType(c.opt.SerializeType)
	codec := message.Codecs[serializeType]
	if codec == nil {
		call.Error = ErrUnsupportedCodec
		c.Unlock()
		call.done()
		return
	}

	if c.pending == nil {
		c.pending = make(map[uint64]*Call)
	}
	id := message.CreateID(c.cfg.ServiceId, c.seq.Add(1))
	call.ID = id
	c.pending[id] = call
	c.Unlock()

	data, err := codec.Encode(call.Args)
	if err != nil {
		c.Lock()
		delete(c.pending, id)
		c.Unlock()
		call.Error = err
		call.done()
		return
	}

	totalL := message.HeaderLength + 4 + (4 + message.SizeMeta(call.ReqMetadata)) + (4 + len(data))
	m := mangos.NewMessage(totalL)
	m.Header = protocol.PutHeader(m.Header, src, protocol.SignallingAssign, dest, eventHash)
	req := message.GetMessage()
	defer req.Free()

	req.SetMessageType(message.Request)
	req.SetMsgID(call.ID)
	if call.Reply == nil {
		req.SetOneway(true)
	}
	req.SetSerializeType(message.SerializeType(c.opt.SerializeType))
	if call.ReqMetadata != nil {
		req.Metadata = call.ReqMetadata
	}
	req.ServicePath = call.ServicePath
	req.ServiceMethod = call.ServiceMethod
	// if c.cfg.MinZipSize > 0 && int64(len(data)) > c.cfg.MinZipSize {
	// 	compressType := message.CompressType(c.opt.CompressType)
	// 	if compressType != message.None {
	// 		req.SetCompressType(compressType)
	// 	}
	// }
	req.Payload = data
	body := req.Encode(m.Body)
	m.Body = m.Body[:len(body)]
	err = c.socket.SendMsg(m)
	if c.opt.TraceMessage {
		log.Debugf("c.sent for %s.%s, args: %+v in case of client call", call.ServicePath, call.ServiceMethod, call.Args)
	}
	if err != nil {
		m.Free()
		c.Lock()
		call = c.pending[id]
		delete(c.pending, id)
		c.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
		return
	}

	if req.IsOneway() {
		c.Lock()
		call = c.pending[id]
		delete(c.pending, id)
		c.Unlock()
		if call != nil {
			call.done()
		}
	}
}

func (c *XClient) receive(m *mangos.Message) (*mangos.Message, error) {
	if len(m.Body) < message.HeaderLength {
		m.Free()
		return nil, message.ErrInvalidMessage
	}

	h := message.Header(m.Body[:message.HeaderLength])
	if h.Version() != message.MsgVersionOne {
		m.Free()
		log.Errorf("client received data that version not match: %v", h)
		return nil, message.ErrVersionNotMatch
	}

	id := h.MsgID()
	var call *Call
	isRequestMessage := h.MessageType() == message.Request
	if !isRequestMessage {
		c.Lock()
		call = c.pending[id]
		delete(c.pending, id)
		c.Unlock()
	}
	if c.opt.TraceMessage {
		log.Debugf("client received %v", h)
	}
	if call == nil {
		return m, nil
	}

	res := message.GetMessage()
	defer res.Free()
	if err := res.Decode(m.Body); err != nil {
		m.Free()
		log.Warnf("failed to decode response: %v", err)
		return nil, err
	}

	switch {
	case res.StatusType() == message.Error:
		// We've got an error response. Give this to the request
		if len(res.Payload) > 0 {
			data := res.Payload
			codec := message.Codecs[res.SerializeType()]
			if codec != nil {
				_ = codec.Decode(data, call.Reply)
			}
		}
		if len(res.Metadata) > 0 {
			call.ResMetadata = res.Metadata
			call.Error = errors.New(res.Metadata[message.XServiceError])
		}
		call.done()
	default:
		var err error
		data := res.Payload
		if len(data) > 0 {
			codec := message.Codecs[res.SerializeType()]
			if codec == nil {
				err = ErrUnsupportedCodec
			} else {
				err = codec.Decode(data, call.Reply)
			}
		}
		if err != nil {
			call.Error = err
		}
		if len(res.Metadata) > 0 {
			call.ResMetadata = res.Metadata
		}
		call.done()
	}
	m.Free()
	return nil, nil
}

// Go invokes the function asynchronously. It returns the Call structure representing
// the invocation. The done channel will signal when the call is complete by returning
// the same Call object. If done is nil, Go will allocate a new channel.
// If non-nil, done must be buffered or Go will deliberately crash.
func (c *XClient) Go(ctx context.Context, src, dest uint32, eventHash uint64,
		servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10) // buffered.
	} else {
		// If caller passes done != nil, it must arrange that
		// done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel. If the channel
		// is totally unbuffered, it's best not to run at all.
		if cap(done) == 0 {
			log.Errorf("done channel is unbuffered")
			return nil
		}
	}

	call := &Call{}
	call.ServicePath = servicePath
	call.ServiceMethod = serviceMethod
	meta := message.DecodeReqMetaContext(ctx)
	if len(meta) != 0 { // copy meta in context to meta in requests
		call.ReqMetadata = meta
	}
	call.Content = message.DecodeContentContext(ctx)
	call.Args = args
	call.Reply = reply
	call.Done = done

	if c.opt.TraceMessage {
		log.Debugf("c.Go send request for %s.%s, args: %+v in case of client call", servicePath, serviceMethod, args)
	}

	go c.send(ctx, src, dest, eventHash, call)

	return call
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (c *XClient) Call(ctx context.Context, src, dest uint32, eventHash uint64,
		servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	if c.opt.TraceMessage {
		log.Debugf("c.call for %s.%s, args: %+v in case of client call", servicePath, serviceMethod, args)
		defer func() {
			log.Debugf("c.call done for %s.%s, reply: %+v in case of client call", servicePath, serviceMethod, reply)
		}()
	}

	var cancel context.CancelFunc
	if c.opt.IdleTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.opt.IdleTimeout)
	}
	call := c.Go(ctx, src, dest, eventHash, servicePath, serviceMethod, args, reply, make(chan *Call, 1))

	var err error
	select {
	case <-ctx.Done(): // cancel by context
		if call != nil && call.ID != 0 {
			c.Lock()
			delete(c.pending, call.ID)
			c.Unlock()
			call.Error = ctx.Err()
			call.done()
		}
		err = ctx.Err()
	case cl := <-call.Done:
		err = cl.Error
	}
	if cancel != nil {
		cancel()
	}
	return err
}
