/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package protocol

import (
	`errors`
)

var (
	ErrBadEventName = errors.New("invalid event name")
	ErrBadAddress   = errors.New("invalid socket address")
	ErrNoWatcher    = errors.New("no connected watcher")
	ErrNoSource     = errors.New("no connected source")
)
