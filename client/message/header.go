/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package message

import (
	`encoding/binary`
)

// Compressors are compressors supported by rpc. You can add customized compressor in Compressors.
var Compressors = map[CompressType]Compressor{}

const (
	MsgVersionOne byte = 1
	// MsgVersionTwo byte = 2
)

// ProtocolType is message type of requests and responses.
type ProtocolType byte

const (
	// Request is message type of request
	Request ProtocolType = iota
	// Response is message type of response
	Response
)

// StatusType is status of messages.
type StatusType byte

const (
	// Normal is normal requests and responses.
	Normal StatusType = iota
	// Error indicates some errors occur.
	Error
)

// CompressType defines decompression type.
type CompressType byte

const (
	// None does not compress.
	None CompressType = iota
	// Gzip uses gzip compression.
	Gzip
	Snappy
	ZStd
)

// SerializeType defines serialization type of payload.
type SerializeType byte

const (
	// SerializeNone uses raw []byte and don't serialize/deserialize
	SerializeNone SerializeType = iota
	// ProtoBuffer for payload.
	ProtoBuffer
	// JSON for payload.
	JSON
	// JSONIter for payload
	JSONIter
)

const HeaderLength = 12

// Header is the first part of Message and has fixed size.
// Format:
//   version uint8
//   protocolType uint8:7
//   oneway uint8:6
//   statusType uint8:4-5
//   serializeType uint8:0-3
//   compressType uint8
//   reserved uint8
//   messageID uint64
type Header [HeaderLength]byte

func NewHeader(h []byte) *Header {
	if cap(h) < HeaderLength {
		return nil
	}

	header := Header(h[:HeaderLength])
	return &header
}

// Version returns version of app protocol.
func (h *Header) Version() byte {
	return h[0]
}

// SetVersion sets version for this header.
func (h *Header) SetVersion(v byte) {
	h[0] = v
}

// MessageType returns the message protocol type.
func (h *Header) MessageType() ProtocolType {
	return ProtocolType(h[1] & 0x80 >> 7)
}

// SetMessageType sets message type.
func (h *Header) SetMessageType(mt ProtocolType) {
	h[1] = h[1] | (byte(mt) << 7)
}

// IsOneway returns whether the message is one-way message.
// If true, server won't send responses.
func (h *Header) IsOneway() bool {
	return h[1]&0x40 == 0x40
}

// SetOneway sets the oneway flag.
func (h *Header) SetOneway(oneway bool) {
	if oneway {
		h[1] = h[1] | 0x40
	} else {
		h[1] = h[1] &^ 0x40
	}
}

// StatusType returns the message status type.
func (h *Header) StatusType() StatusType {
	return StatusType((h[1] & 0x30) >> 4)
}

// SetStatusType sets message status type.
func (h *Header) SetStatusType(mt StatusType) {
	h[1] = (h[1] &^ 0x30) | ((byte(mt) << 4) & 0x30)
}

// SerializeType returns serialization type of payload.
func (h *Header) SerializeType() SerializeType {
	return SerializeType(h[1] & 0x0F)
}

// SetSerializeType sets the serialization type.
func (h *Header) SetSerializeType(st SerializeType) {
	h[1] = (h[1] &^ 0x0F) | (byte(st) & 0x0F)
}

// CompressType returns compression type of messages.
func (h *Header) CompressType() CompressType {
	return CompressType(h[2])
}

// SetCompressType sets the compression type.
func (h *Header) SetCompressType(ct CompressType) {
	h[2] = byte(ct)
}

// MsgID returns sequence number of messages.
func (h *Header) MsgID() uint64 {
	return binary.BigEndian.Uint64(h[4:])
}

// SetMsgID sets  sequence number.
func (h *Header) SetMsgID(id uint64) {
	binary.BigEndian.PutUint64(h[4:], id)
}

var (
	zeroHeaderArray Header
	zeroHeader      = zeroHeaderArray[1:]
)

func resetHeader(h *Header) {
	copy(h[1:], zeroHeader)
}
