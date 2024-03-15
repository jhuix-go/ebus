/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package message

import (
	`encoding/binary`
	`encoding/hex`
	`io`
	`math`
	`unsafe`

	`github.com/jhuix-go/ebus/log`
)

type BufferRead struct {
	buf []byte
	off int
	err error
}

func NewBufferRead(buf []byte) *BufferRead {
	return &BufferRead{buf, 0, nil}
}

func (b *BufferRead) Error() error {
	return b.err
}

func (b *BufferRead) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
	b.err = nil
}

func (b *BufferRead) Buf() ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}

	return b.buf[b.off:], nil
}

func (b *BufferRead) Offset() int {
	return b.off
}

func (b *BufferRead) Byte() byte {
	if b.err != nil {
		return 0
	}
	if b.off+1 > b.Len() {
		log.Errorf("decode byte error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := b.buf[b.off]
	b.off += 1
	return x
}

func (b *BufferRead) Int16() int16 {
	if b.err != nil {
		return 0
	}
	if b.off+2 > b.Len() {
		log.Errorf("decode int16 error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := binary.LittleEndian.Uint16(b.buf[b.off : b.off+2])
	b.off += 2
	return int16(x)
}

func (b *BufferRead) UInt16() uint16 {
	if b.err != nil {
		return 0
	}
	if b.off+2 > b.Len() {
		log.Errorf("decode uint16 error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := binary.LittleEndian.Uint16(b.buf[b.off : b.off+2])
	b.off += 2
	return x
}

func (b *BufferRead) Int() int {
	return int(b.Int32())
}

func (b *BufferRead) UInt() uint {
	return uint(b.UInt32())
}

func (b *BufferRead) Int32() int32 {
	if b.err != nil {
		return 0
	}
	if b.off+4 > b.Len() {
		log.Errorf("decode int32 error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := binary.LittleEndian.Uint32(b.buf[b.off : b.off+4])
	b.off += 4
	return int32(x)
}

func (b *BufferRead) UInt32() uint32 {
	if b.err != nil {
		return 0
	}
	if b.off+4 > b.Len() {
		log.Errorf("decode uint32 error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := binary.LittleEndian.Uint32(b.buf[b.off : b.off+4])
	b.off += 4
	return x
}

func (b *BufferRead) UInt64() uint64 {
	if b.err != nil {
		return 0
	}
	if b.off+8 > b.Len() {
		log.Errorf("decode uint64 error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := binary.LittleEndian.Uint64(b.buf[b.off : b.off+8])
	b.off += 8
	return x
}

func (b *BufferRead) Int64() int64 {
	if b.err != nil {
		return 0
	}
	if b.off+8 > b.Len() {
		log.Errorf("decode int64 error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := int64(binary.LittleEndian.Uint64(b.buf[b.off : b.off+8]))
	b.off += 8
	return x
}

func (b *BufferRead) Double() float64 {
	if b.err != nil {
		return 0
	}
	if b.off+8 > b.Len() {
		log.Errorf("decode double error - {off: %d, size: %d}", b.off, b.Len())
		b.err = io.EOF
		return 0
	}
	x := math.Float64frombits(binary.LittleEndian.Uint64(b.buf[b.off : b.off+8]))
	b.off += 8
	return x
}

func (b *BufferRead) Bytes(size int) []byte {
	if b.err != nil {
		return nil
	}
	if b.off+size > b.Len() {
		log.Errorf("decode size_bytes(%d) error - {off: %d, size: %d}", size, b.off, b.Len())
		b.err = io.EOF
		return nil
	}
	x := make([]byte, size)
	copy(x, b.buf[b.off:b.off+size])
	b.off += size
	return x
}

func (b *BufferRead) String() string {
	if b.err != nil {
		return ""
	}

	off := 0
	for {
		if b.buf[b.off+off] == 0 || b.off+off >= b.Len() || off >= math.MaxUint16 {
			break
		}

		off++
	}

	data := b.buf[b.off : b.off+off]
	b.off += off
	return *(*string)(unsafe.Pointer(&data))
}

func (b *BufferRead) Varchar() string {
	if b.err != nil {
		return ""
	}

	size := b.Int()
	if b.off+size > b.Len() {
		size = b.Len() - b.off
		log.Errorf("decode uint64 error - {off: %d, len: %d, size: %d}", b.off, size, b.Len())
	}
	data := b.buf[b.off : b.off+size]
	b.off += size
	return *(*string)(unsafe.Pointer(&data))
}

func (b *BufferRead) DumpSize(size int) string {
	if b.off+size > b.Len() {
		size = b.Len() - b.off
	}
	return hex.Dump(b.buf[b.off : b.off+size])
}

func (b *BufferRead) Dump() string {
	return hex.Dump(b.buf[b.off:b.Len()])
}

func (b *BufferRead) Len() int {
	return len(b.buf)
}

type BufferWrite struct {
	buf []byte
}

func NewBufferWrite(buf []byte) *BufferWrite {
	return &BufferWrite{buf}
}

func (b *BufferWrite) Len() int {
	return len(b.buf)
}

func (b *BufferWrite) Reset() {
	b.buf = b.buf[:0]
}

func (b *BufferWrite) Buf() []byte {
	return b.buf
}

func (b *BufferWrite) Detach() []byte {
	buf := b.buf
	b.buf = nil
	return buf
}

func (b *BufferWrite) AppendByte(s byte) {
	b.buf = append(b.buf, s)
}

func (b *BufferWrite) AppendInt16(s int16) {
	b.buf = append(b.buf, 0, 0)
	binary.LittleEndian.PutUint16(b.buf[len(b.buf)-2:], uint16(s))
}

func (b *BufferWrite) AppendUInt16(s uint16) {
	b.buf = append(b.buf, 0, 0)
	binary.LittleEndian.PutUint16(b.buf[len(b.buf)-2:], s)
}

func (b *BufferWrite) AppendInt(s int) {
	b.buf = append(b.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(b.buf[len(b.buf)-4:], uint32(s))
}

func (b *BufferWrite) AppendUInt(s uint) {
	b.buf = append(b.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(b.buf[len(b.buf)-4:], uint32(s))
}

func (b *BufferWrite) AppendInt32(s int32) {
	b.buf = append(b.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(b.buf[len(b.buf)-4:], uint32(s))
}

func (b *BufferWrite) AppendUInt32(s uint32) {
	b.buf = append(b.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(b.buf[len(b.buf)-4:], s)
}

func (b *BufferWrite) AppendInt64(s int64) {
	b.buf = append(b.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(b.buf[len(b.buf)-8:], uint64(s))
}

func (b *BufferWrite) AppendUInt64(s uint64) {
	b.buf = append(b.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(b.buf[len(b.buf)-8:], s)
}

func (b *BufferWrite) AppendDouble(s float64) {
	b.buf = append(b.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(b.buf[len(b.buf)-8:], math.Float64bits(s))
}

func (b *BufferWrite) AppendBytes(s []byte) {
	b.buf = append(b.buf, s...)
}

func (b *BufferWrite) AppendString(s string) {
	b.buf = append(b.buf, s[:]...)
	b.buf = append(b.buf, 0)
}

func (b *BufferWrite) AppendVarchar(s string) {
	b.AppendInt(len(s))
	b.buf = append(b.buf, s[:]...)
}
