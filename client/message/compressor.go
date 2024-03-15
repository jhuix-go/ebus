/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package message

import (
	`io`
)

type Stream interface {
	io.Writer
	io.Reader
	Bytes() []byte
	Detach() []byte
	Reset()
	Free()
}

// Compressor defines a common compression interface.
type Compressor interface {
	Zip([]byte) (Stream, error)
	Unzip([]byte) (Stream, error)
}
