/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package balancer

import (
	"fmt"
)

const (
	ConsistentHash           = "consistent_hash"
	DefaultConsistentHashKey = "consistent-hash"
)

func init() {
	Register(newConsistentHashPickerBuilder())
}

type consistentHashPickerBuilder struct {
	name string
}

func newConsistentHashPickerBuilder() PickerBuilder {
	return &consistentHashPickerBuilder{ConsistentHash}
}

func (b *consistentHashPickerBuilder) Build(buildInfo PickerBuildInfo) Picker {
	if len(buildInfo.ReadySCs) == 0 {
		return NewErrPicker(ErrNoSubConnAvailable)
	}

	picker := &consistentHashPicker{
		nodes: make(map[string]any),
		hash:  NewKetama(DefaultReplicas, nil),
	}
	for sc, conInfo := range buildInfo.ReadySCs {
		weight := GetWeight(conInfo.Address)
		for i := 0; i < weight; i++ {
			node := wrapAddr(conInfo.Address.Addr, i)
			picker.hash.Add(node)
			picker.nodes[node] = sc
		}
	}
	return picker
}

func (b *consistentHashPickerBuilder) Name() string {
	return b.name
}

type consistentHashPicker struct {
	nodes map[string]any
	hash  *Ketama
}

func (p *consistentHashPicker) Pick(info PickInfo) (ret PickResult, err error) {
	if info.Ctx == nil || len(p.nodes) == 0 {
		err = ErrNoContextAvailable
		return
	}

	key, _ := info.Ctx.Value(DefaultConsistentHashKey).(string)
	if len(key) == 0 {
		err = ErrNoContextAvailable
		return
	}

	addr, _ := p.hash.Get(key)
	if len(addr) == 0 {
		err = ErrNoSubConnAvailable
		return
	}

	ret.Node, _ = p.nodes[addr]
	return
}

func wrapAddr(addr string, idx int) string {
	return fmt.Sprintf("%s-%d", addr, idx)
}
