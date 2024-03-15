/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package watch

type ClientManagerHandler interface {
	AddClient(name string, address string)
	UpdateClient(name string, address string, conn any)
	RemoveClient(name string, address string)
}
