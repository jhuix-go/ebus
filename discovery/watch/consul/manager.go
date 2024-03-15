/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package consul

import (
	`context`
	`sync`

	`github.com/jhuix-go/ebus/discovery/balancer`
	`github.com/jhuix-go/ebus/discovery/watch`
	`github.com/jhuix-go/ebus/log`
)

type manager struct {
	sync.RWMutex
	balance string
	builder balancer.PickerBuilder
	picker  balancer.Picker
	nodes   map[string]*balancer.Address
	handler watch.ClientManagerHandler
}

func newManager(balance string, handler watch.ClientManagerHandler) *manager {
	b := balancer.Builder(balance)
	if b == nil {
		b = balancer.Builder(balancer.Random)
	}
	return &manager{
		balance: balance,
		builder: b,
		nodes:   make(map[string]*balancer.Address),
		handler: handler,
	}
}

const subConnectionKey = "sub-connection"

func (m *manager) UpdateState(del *balancer.Address, add *balancer.Address) {
	var updated = false
	if del != nil {
		delete(m.nodes, del.Addr)
		m.handler.RemoveClient(del.Name, del.Addr)
		updated = true
	}

	if add != nil {
		m.nodes[add.Addr] = add
		if add.Attributes == nil {
			m.handler.AddClient(add.Name, add.Addr)
		}
		updated = true
	}

	if updated && m.builder != nil {
		info := balancer.PickerBuildInfo{ReadySCs: make(map[any]balancer.SubConnInfo)}
		for _, node := range m.nodes {
			clt := node.Attributes.Value(subConnectionKey)
			if clt != nil {
				info.ReadySCs[clt] = balancer.SubConnInfo{
					Address: balancer.Address{
						Name: node.Name,
						Addr: node.Addr,
					},
				}
			}
		}
		m.Lock()
		m.picker = m.builder.Build(info)
		m.Unlock()
	}
}

func (m *manager) Pick(info *watch.PickInfo) any {
	pickInfo := balancer.PickInfo{}
	if info != nil {
		pickInfo.Name = info.Name
		if len(info.Hash) != 0 {
			pickInfo.Ctx = context.WithValue(context.Background(), balancer.DefaultConsistentHashKey, info.Hash)
		}
	}
	m.RLock()
	ret, err := m.picker.Pick(pickInfo)
	m.RUnlock()
	if err != nil {
		log.Errorf("pick node {%s, %s}, error: %v", info.Name, info.Hash, err)
		return nil
	}

	return ret.Node
}
