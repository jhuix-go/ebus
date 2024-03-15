/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
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
	DefaultFlag         = 0x6562 // eb
	DefaultVersion      = 0x1
	DefaultHeaderLength = 12
	MaskVersion         = 0xF0
	MaskHeaderLength    = 0x0F
)

const (
	SignallingAssign         uint8 = 0x00
	SignallingEvent          uint8 = 0x10
	SignallingControl        uint8 = 0x20
	SignallingCommand        uint8 = 0x0F
	SignallingHeart          uint8 = 0x00
	SignallingRegisterEvent  uint8 = 0x01
	SignallingRegisterRemote uint8 = 0x02
)

/*
   	flag         uint16
   	version      uint8:0-4
       headerLength uint8:5-8
   	signalling   uint8
   	src          uint32
   	dest         uint32
*/

type Header struct {
	Data []byte
}

func PutHeader(header []byte, src uint32, signalling uint8, dest uint32) []byte {
	if cap(header) < DefaultHeaderLength {
		return header
	}

	header = header[:DefaultHeaderLength]
	h := Header{Data: header}
	h.SetFlag(DefaultFlag)
	h.SetVersion(DefaultVersion)
	h.SetHeaderLength(DefaultHeaderLength)
	h.SetSignalling(signalling)
	h.SetSrc(src)
	h.SetDest(dest)
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

func (h *Header) IsHeart() bool {
	return h.Signalling() == SignallingControl|SignallingHeart
}

func (h *Header) IsRegisterEvent() bool {
	return h.Signalling() == SignallingControl|SignallingRegisterEvent
}

func (h *Header) IsRegisterRemote() bool {
	return h.Signalling() == SignallingControl|SignallingRegisterRemote
}

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
