/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package message

import (
	`errors`
)

var (
	// ErrMetaKVMissing some keys or values are missing.
	ErrMetaKVMissing = errors.New("wrong metadata lines. some keys or values are missing")
	// ErrMessageTooLong message is too long
	ErrMessageTooLong        = errors.New("message is too long")
	ErrUnsupportedCompressor = errors.New("unsupported compressor")
	ErrMetadataIsEmpty       = errors.New("metadata is empty")
	ErrVersionNotMatch       = errors.New("version is not match")
	ErrInvalidMessage        = errors.New("invalid message")
	ErrTooLongSize           = errors.New("data too long size")
)
