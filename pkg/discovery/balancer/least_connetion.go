/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package balancer

import (
	"math/rand"
	"sync/atomic"
	"time"
)

const LeastConnection = "least_connection"

func init() {
	Register(newLeastConnectionPickerBuilder())
}

type leastConnectionPickerBuilder struct {
	name string
}

// newLeastConnectionPickBuilder creates a new leastConnection balancer picker builder.
func newLeastConnectionPickerBuilder() PickerBuilder {
	return &leastConnectionPickerBuilder{LeastConnection}
}

func (b *leastConnectionPickerBuilder) Build(buildInfo PickerBuildInfo) Picker {
	if len(buildInfo.ReadySCs) == 0 {
		return NewErrPicker(ErrNoSubConnAvailable)
	}

	var nodes []*leastNode
	for sc, _ := range buildInfo.ReadySCs {
		nodes = append(nodes, &leastNode{sc, 0})
	}
	return &leastConnectionPicker{
		nodes: nodes,
		rand:  rand.New(rand.NewSource(time.Now().Unix())),
	}
}

func (b *leastConnectionPickerBuilder) Name() string {
	return b.name
}

type leastNode struct {
	conn     any
	inflight int64
}

type leastConnectionPicker struct {
	nodes []*leastNode
	rand  *rand.Rand
}

func (p *leastConnectionPicker) Pick(info PickInfo) (ret PickResult, err error) {
	if len(p.nodes) == 0 || len(p.nodes) == 0 {
		err = ErrNoSubConnAvailable
		return
	}

	var node *leastNode
	if len(p.nodes) == 1 {
		node = p.nodes[0]
	} else {
		a := p.rand.Intn(len(p.nodes))
		b := p.rand.Intn(len(p.nodes))
		if a == b {
			b = (b + 1) % len(p.nodes)
		}
		if p.nodes[a].inflight < p.nodes[b].inflight {
			node = p.nodes[a]
		} else {
			node = p.nodes[b]
		}
	}
	atomic.AddInt64(&node.inflight, 1)
	ret.Node = node.conn
	ret.Done = func(info DoneInfo) {
		atomic.AddInt64(&node.inflight, -1)
	}
	return
}
