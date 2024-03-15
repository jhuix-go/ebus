/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package message

import (
	`time`
)

const (
	Epoch int64 = 1625068800 // 2021.7.1 00:00:00
)

func CreateID(sid int, seq uint32) uint64 {
	return uint64(time.Now().Unix()-Epoch)<<32 | uint64(sid&0x3FF)<<22 | uint64(seq&0x3FFFFF)
}
