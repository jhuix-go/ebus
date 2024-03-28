/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package protocol

import (
	`encoding/binary`
	`fmt`
)

// const (
// 	ProtocolTcp = 1
// 	ProtocolIpc = 2
// 	ProtocolWs  = 3
// 	ProtocolWss = 4
// )

const (
	DefaultFlag             = 0x6562 // eb
	DefaultVersion          = 0x1
	DefaultHeaderLength     = 12
	DefaultHashHeaderLength = 20
	MaxHeaderLength         = 60
	MaskVersion             = 0xF0
	MaskHeaderLength        = 0x0F
)

// Signalling Format (UINT8):
//    7   6   5   4   3   2   1   0
//    *   *   *   *   *   *   *   *
//    │   │   │   │   │           │
//    └─┬─┘   │   │   └─────┬─────┘
//   Reserve  │   │      Command - 0x00 ~ 0x0F
//            │   └> Event - 0x10
//            └> Control - 0x20
//   0    0   0   0 -> Assign - 0x00
const (
	SignallingCommand       uint8 = 0x0F
	SignallingAssign        uint8 = 0x00
	SignallingEvent         uint8 = 0x10
	SignallingControl       uint8 = 0x20
	SignallingHeart         uint8 = 0x00
	SignallingRegisterEvent uint8 = 0x01
	SignallingHash          uint8 = 0x02
	SignallingEventHash           = SignallingEvent | SignallingHash
)

/* Header Format:
   	flag         uint16
   	version      uint8:4-7
    headerLength uint8:0-3 (12, 20)
   	signalling   uint8
   	src          uint32
   	dest         uint32
    hash         uint64
*/

type Header struct {
	Data []byte
}

func PutHeader(header []byte, src uint32, signalling uint8, dest uint32) []byte {
	headerLength := DefaultHeaderLength
	if cap(header) < headerLength {
		return header
	}

	header = header[:headerLength]
	h := Header{Data: header}
	h.SetFlag(DefaultFlag)
	h.SetVersion(DefaultVersion)
	h.SetHeaderLength(uint8(headerLength))
	h.SetSignalling(signalling)
	h.SetSrc(src)
	h.SetDest(dest)
	return header
}

func PutHashHeader(header []byte, src uint32, dest uint32, hash uint64) []byte {
	headerLength := DefaultHashHeaderLength
	if cap(header) < headerLength {
		return header
	}

	header = header[:headerLength]
	h := Header{Data: header}
	h.SetFlag(DefaultFlag)
	h.SetVersion(DefaultVersion)
	h.SetHeaderLength(uint8(headerLength))
	h.SetSignalling(SignallingEventHash)
	h.SetSrc(src)
	h.SetDest(dest)
	h.SetHash(hash)
	return header
}

func StringHeader(header []byte) string {
	h := Header{Data: header}
	flag := string(h.Data[:2])
	if h.Signalling()&SignallingEvent != 0 || h.Dest() == PipeEbus {
		dest := string(h.Data[8:12])
		return fmt.Sprintf("{\"flag\":\"%s\",\"signalling\":0x%x,\"src\":%d,\"dest\":\"%s\"}",
			flag, h.Signalling(), h.Src(), dest)
	}

	return fmt.Sprintf("{\"flag\":\"%s\",\"signalling\":0x%x,\"src\":%d,\"dest\":%d}",
		flag, h.Signalling(), h.Src(), h.Dest())
}

func StringEvent(event uint32) string {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:4], event)
	return string(buf[:])
}

func EventNameN(name string) uint32 {
	var id uint32
	if len(name) != 0 {
		var buf [4]byte
		length := min(4, len(name))
		for i := 0; i < length; i++ {
			buf[i] = name[i]
		}
		id = binary.BigEndian.Uint32(buf[:4])
	}
	return id
}

func (h *Header) Flag() uint16 {
	return binary.BigEndian.Uint16(h.Data[:2])
}

func (h *Header) SetFlag(v uint16) {
	binary.BigEndian.PutUint16(h.Data[:2], v)
}

func (h *Header) Version() uint8 {
	return (h.Data[2] * MaskVersion) >> 4
}

func (h *Header) SetVersion(v uint8) {
	h.Data[2] = (h.Data[2] & MaskHeaderLength) | ((v << 4) & MaskVersion)
}

func (h *Header) HeaderLength() uint8 {
	return (h.Data[2] & MaskHeaderLength) * 4
}

func (h *Header) SetHeaderLength(v uint8) {
	h.Data[2] = (h.Data[2] & MaskVersion) | (uint8(v/4) & MaskHeaderLength)
}

func (h *Header) Signalling() uint8 {
	return h.Data[3]
}

func (h *Header) SetSignalling(v uint8) {
	h.Data[3] = v
}

func (h *Header) SignallingType() uint8 {
	return h.Signalling() & ^SignallingCommand
}

func (h *Header) SetSignallingType(v uint8) {
	h.Data[3] = (h.Data[3] & SignallingCommand) | v
}

func (h *Header) IsHeart() bool {
	return h.Signalling() == SignallingControl|SignallingHeart
}

func (h *Header) IsRegisterEvent() bool {
	return h.Signalling() == SignallingControl|SignallingRegisterEvent
}

func (h *Header) HasHash() bool {
	return h.Signalling()&SignallingCommand == SignallingHash
}

// func (h *Header) IsRegisterRemote() bool {
// 	return h.Signalling() == SignallingControl|SignallingRegisterRemote
// }

func (h *Header) Src() uint32 {
	return binary.BigEndian.Uint32(h.Data[4:8])
}

func (h *Header) SetSrc(v uint32) {
	binary.BigEndian.PutUint32(h.Data[4:8], v)
}

func (h *Header) Dest() uint32 {
	return binary.BigEndian.Uint32(h.Data[8:12])
}

func (h *Header) SetDest(v uint32) {
	binary.BigEndian.PutUint32(h.Data[8:12], v)
}

func (h *Header) Hash() uint64 {
	return binary.BigEndian.Uint64(h.Data[12:20])
}

func (h *Header) SetHash(v uint64) {
	binary.BigEndian.PutUint64(h.Data[12:20], v)
}
