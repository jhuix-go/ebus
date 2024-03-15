/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package balancer

import (
	"math/rand"
	"time"
)

const Random = "random"

func init() {
	Register(newRandomPickerBuilder())
}

type randomPickerBuilder struct {
	name string
}

// newRandomBuilder creates a new random balancer picker builder.
func newRandomPickerBuilder() PickerBuilder {
	return &randomPickerBuilder{Random}
}

func (*randomPickerBuilder) Build(buildInfo PickerBuildInfo) Picker {
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
	return &randomPicker{
		nodes: nodes,
		rand:  rand.New(rand.NewSource(time.Now().Unix())),
	}
}

func (b *randomPickerBuilder) Name() string {
	return b.name
}

type randomPicker struct {
	nodes []any
	rand  *rand.Rand
}

func (p *randomPicker) Pick(info PickInfo) (ret PickResult, err error) {
	if len(p.nodes) == 0 {
		err = ErrNoSubConnAvailable
		return
	}

	ret.Node = p.nodes[p.rand.Intn(len(p.nodes))]
	return
}
