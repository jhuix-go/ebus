/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package balancer

import (
	"math/rand"
)

const RoundRobin = "round_robin"

func init() {
	Register(newRoundRobinPickerBuilder())
}

type roundRobinPickerBuilder struct {
	name string
}

// newRoundRobinPickerBuilder creates a new roundRobin balancer builder.
func newRoundRobinPickerBuilder() PickerBuilder {
	return &roundRobinPickerBuilder{RoundRobin}
}

func (*roundRobinPickerBuilder) Build(buildInfo PickerBuildInfo) Picker {
	if len(buildInfo.ReadySCs) == 0 {
		return NewErrPicker(ErrNoSubConnAvailable)
	}

	var nodes []any
	for sc, sci := range buildInfo.ReadySCs {
		weight := GetWeight(sci.Address)
		for i := 0; i < weight; i++ {
			nodes = append(nodes, sc)
		}
	}
	next := rand.Intn(len(nodes))
	return &roundRobinPicker{
		nodes: nodes,
		next:  next,
	}
}

func (b *roundRobinPickerBuilder) Name() string {
	return b.name
}

type roundRobinPicker struct {
	nodes []any
	next  int
}

func (p *roundRobinPicker) Pick(info PickInfo) (ret PickResult, err error) {
	if len(p.nodes) == 0 {
		err = ErrNoSubConnAvailable
		return
	}

	ret.Node = p.nodes[p.next]
	p.next = (p.next + 1) % len(p.nodes)
	return
}
