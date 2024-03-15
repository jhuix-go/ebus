/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a MIT license
 * that can be found in the LICENSE file.
 */

package consul

import (
	"encoding/json"
	"sync"

	"github.com/hashicorp/consul/api"
	consulWatch "github.com/hashicorp/consul/api/watch"

	`github.com/jhuix-go/ebus/discovery/balancer`
	`github.com/jhuix-go/ebus/discovery/common/attributes`
	`github.com/jhuix-go/ebus/discovery/common/metadata`
	`github.com/jhuix-go/ebus/discovery/watch`
	`github.com/jhuix-go/ebus/log`
)

type nodeData struct {
	Addr     string      `json:"address"`
	Metadata metadata.MD `json:"metadata"`
}

type nodeAttributes struct {
	Remove     bool                   `json:"remove"`
	Addr       string                 `json:"address"`
	Attributes *attributes.Attributes `json:"attrs"`
}

type Config struct {
	ServiceName    string `json:"name" yaml:"name" toml:"name"`
	ServiceVersion string `json:"version,omitempty" yaml:"version,omitempty" toml:"version,omitempty"`
	RegistryDir    string `json:"registry_dir,omitempty" yaml:"registry_dir,omitempty" toml:"registry_dir,omitempty"`
	Balancer       string `json:"balancer,omitempty" yaml:"balancer,omitempty" toml:"balancer,omitempty"`
}

type Watcher struct {
	wp         *consulWatch.Plan
	config     *Config
	apiConfig  *api.Config
	manager    *manager
	wg         sync.WaitGroup
	nodes      map[string]nodeData
	cacheNodes map[string]nodeData
	nodesQ     chan map[string]nodeData
	attrsQ     chan nodeAttributes
	doneQ      chan struct{}
}

func NewWatcher(cfg *Config, apiCfg *api.Config, handler watch.ClientManagerHandler) (watch.Watcher, error) {
	watchName := cfg.RegistryDir
	if len(cfg.RegistryDir) > 0 {
		watchName += "-"
	}
	watchName += cfg.ServiceName
	if len(cfg.ServiceVersion) > 0 {
		watchName += "-" + cfg.ServiceVersion
	}
	wp, err := consulWatch.Parse(map[string]interface{}{
		"type":    "service",
		"service": watchName,
	})
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		wp:         wp,
		config:     cfg,
		apiConfig:  apiCfg,
		manager:    newManager(cfg.Balancer, handler),
		nodes:      map[string]nodeData{},
		cacheNodes: map[string]nodeData{},
		nodesQ:     make(chan map[string]nodeData, 10),
		attrsQ:     make(chan nodeAttributes, 10),
		doneQ:      make(chan struct{}),
	}
	wp.Handler = w.handler
	return w, nil
}

func (w *Watcher) Pick(info *watch.PickInfo) any {
	return w.manager.Pick(info)
}

func (w *Watcher) Close() {
	w.wp.Stop()
	w.wg.Wait()
	close(w.doneQ)
	clear(w.nodes)
	clear(w.cacheNodes)
}

func (w *Watcher) handler(idx uint64, data interface{}) {
	_ = idx
	entries, ok := data.([]*api.ServiceEntry)
	if !ok {
		return
	}

	if w.wp.IsStopped() {
		return
	}

	nodes := make(map[string]nodeData)
	for _, e := range entries {
		for _, check := range e.Checks {
			if check.ServiceID == e.Service.ID {
				if check.Status == api.HealthPassing {
					md := metadata.MD{}
					if len(e.Service.Tags) > 0 {
						err := json.Unmarshal([]byte(e.Service.Tags[0]), &md)
						if err != nil {
							log.Errorf("parse node data error: %v", err)
						}
					}
					nodes[e.Service.ID] = nodeData{e.Service.Address, md}
				}
				break
			}
		}
	}

	if !isSameNodes(w.cacheNodes, nodes) {
		w.cacheNodes = nodes
		w.nodesQ <- cloneAddresses(nodes)
	}
}

func (w *Watcher) Watch() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		_ = w.wp.RunWithConfig(w.apiConfig.Address, w.apiConfig)
	}()

	doneQ := w.doneQ
	nodesQ := w.nodesQ
	attrsQ := w.attrsQ
	for {
		select {
		case nodes := <-nodesQ:
			for k, node := range nodes {
				w.addClient(k, node)
			}
		case attrs := <-attrsQ:
			if attrs.Remove {
				w.removeClient(attrs.Addr)
			} else {
				w.updateClient(attrs.Addr, attrs.Attributes)
			}
		case <-doneQ:
			return
		}
	}
}

func (w *Watcher) Update(addr string, content any) {
	attrs := attributes.NewAttributes(subConnectionKey, content)
	w.attrsQ <- nodeAttributes{Remove: false, Addr: addr, Attributes: attrs}
}

func (w *Watcher) Remove(addr string) {
	w.attrsQ <- nodeAttributes{Remove: true, Addr: addr}
}

func (w *Watcher) addClient(k string, node nodeData) {
	if n, ok := w.nodes[k]; ok {
		if node.Addr != n.Addr {
			w.nodes[k] = node
			w.manager.UpdateState(&balancer.Address{Name: w.config.ServiceName, Addr: n.Addr},
				&balancer.Address{Name: w.config.ServiceName, Addr: node.Addr})
		}
	} else {
		w.nodes[k] = node
		w.manager.UpdateState(nil, &balancer.Address{Name: w.config.ServiceName, Addr: node.Addr})
	}
}

func (w *Watcher) updateClient(addr string, attrs *attributes.Attributes) {
	w.manager.UpdateState(nil, &balancer.Address{Name: w.config.ServiceName, Addr: addr, Attributes: attrs})
}

func (w *Watcher) removeClient(addr string) {
	w.manager.UpdateState(&balancer.Address{Name: w.config.ServiceName, Addr: addr}, nil)
}

func cloneAddresses(in map[string]nodeData) map[string]nodeData {
	out := make(map[string]nodeData, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isSameNodes(a1, a2 map[string]nodeData) bool {
	if len(a1) != len(a2) {
		return false
	}
	for _, addr1 := range a1 {
		found := false
		for _, addr2 := range a2 {
			if addr1.Addr == addr2.Addr {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
