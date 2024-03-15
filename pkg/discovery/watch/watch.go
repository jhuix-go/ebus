/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package watch

type PickInfo struct {
	Name string
	Hash string
}

type Watcher interface {
	Watch()
	Pick(*PickInfo) any
	Update(addr string, content any)
	Remove(addr string)
	Close()
}
