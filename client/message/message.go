/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package message

import (
	`sync`

	`github.com/jhuix-go/ebus/log`
)

const (
	// ServiceError contains error info of service invocation
	ServiceError = "__service_error__"
)

// Message is the generic type of Request and Response.
type Message struct {
	Header
	ServicePath   string
	ServiceMethod string
	Metadata      map[string]string
	Payload       []byte
	Content       interface{}
}

func newMessage() *Message {
	return &Message{}
}

// NewMessageWithHeader creates an empty message.
func newMessageWithHeader() *Message {
	m := newMessage()
	m.Header.SetVersion(MsgVersionOne)
	m.Header.SetSerializeType(ProtoBuffer)
	m.Header.SetCompressType(None)
	return m
}

var spMessage sync.Pool

func GetMessage() *Message {
	return spMessage.Get().(*Message)
}

func PutMessage(m *Message) {
	spMessage.Put(m)
}

func init() {
	spMessage = sync.Pool{New: func() interface{} {
		return newMessageWithHeader()
	}}
}

// Clone clones from a message.
func (m *Message) Clone() *Message {
	c := GetMessage()
	c.Header = m.Header
	c.ServicePath = m.ServicePath
	c.ServiceMethod = m.ServiceMethod
	return c
}

// Encode encodes messages.
func (m *Message) Encode(data []byte) []byte {
	buf := &BufferWrite{buf: data}
	buf.AppendBytes(m.Header[:])
	spL := len(m.ServicePath)
	smL := len(m.ServiceMethod)
	payload := m.Payload
	if m.CompressType() != None {
		compressor := Compressors[m.CompressType()]
		if compressor == nil {
			m.SetCompressType(None)
		} else {
			b, err := compressor.Zip(m.Payload)
			if err != nil {
				m.SetCompressType(None)
				payload = m.Payload
			} else {
				defer b.Free()
				payload = b.Bytes()
			}
		}
	}
	mdL := SizeMeta(m.Metadata)
	dataL := len(payload)
	totalL := 4 + (4 + spL) + (4 + smL) + (4 + mdL) + (4 + dataL)
	buf.AppendInt(totalL)
	buf.AppendVarchar(m.ServicePath)
	buf.AppendVarchar(m.ServiceMethod)
	buf.AppendInt(mdL)
	encodeMeta(m.Metadata, buf)
	buf.AppendInt(dataL)
	buf.AppendBytes(payload)
	return buf.Detach()
}

func SizeMeta(m map[string]string) (n int) {
	for k, v := range m {
		n += len(k) + 4
		n += len(v) + 4
	}
	return
}

// len,string,len,string,......
func encodeMeta(m map[string]string, bb *BufferWrite) {
	if len(m) == 0 {
		return
	}
	for k, v := range m {
		bb.AppendVarchar(k)
		bb.AppendVarchar(v)
	}
}

func decodeMeta(bb *BufferRead) (map[string]string, error) {
	l := bb.Int()
	if l <= 0 {
		return nil, ErrMetadataIsEmpty
	}

	m := make(map[string]string, 10)
	n := 0
	for n < l {
		k := bb.Varchar()
		if len(k) == 0 {
			return nil, ErrMetaKVMissing
		}

		v := bb.Varchar()
		if len(v) == 0 {
			return nil, ErrMetaKVMissing
		}

		m[k] = v
	}

	return m, nil
}

// Decode decodes a message from reader.
func (m *Message) Decode(data []byte) error {
	m.Header = Header(data[:HeaderLength])
	if m.Header.Version() != MsgVersionOne {
		log.Errorf("wrong message version: %v", m.Header.Version())
		return ErrVersionNotMatch
	}

	buf := &BufferRead{buf: data[HeaderLength:]}
	totalSize := buf.Int()
	if totalSize > buf.Len() {
		log.Errorf("wrong message size: %v", totalSize)
		return ErrTooLongSize
	}

	m.ServicePath = buf.Varchar()
	m.ServiceMethod = buf.Varchar()

	meta, err := decodeMeta(buf)
	if err != nil {
		return err
	}

	m.Metadata = meta

	_ = buf.Int()
	m.Payload, _ = buf.Buf()
	if m.CompressType() != None {
		compressor := Compressors[m.CompressType()]
		if compressor == nil {
			return ErrUnsupportedCompressor
		}
		b, err := compressor.Unzip(m.Payload)
		if err != nil {
			return err
		}

		defer b.Free()
		m.Payload = b.Detach()
	}

	return err
}

// Reset clean data of this message but keep allocated data
func (m *Message) Reset() {
	resetHeader(&m.Header)
	m.Metadata = nil
	m.Payload = []byte{}
	m.ServicePath = ""
	m.ServiceMethod = ""
}

func (m *Message) Free() {
	m.Reset()
	PutMessage(m)
}
