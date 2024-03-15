/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package registry

type MD map[string][]string

type ServiceInfo struct {
	InstanceId string
	Name       string
	Version    string
	Address    string
	Metadata   MD
}

type Registrar interface {
	Register(service *ServiceInfo) error
	Unregister(service *ServiceInfo) error
	Close()
}
