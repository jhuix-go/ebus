/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package consul

import (
	"context"
	"encoding/json"
	"fmt"
	`net`
	`net/http`
	`runtime`
	"sync"
	"time"

	consul "github.com/hashicorp/consul/api"

	"github.com/jhuix-go/ebus/pkg/discovery/registry"
	`github.com/jhuix-go/ebus/pkg/log`
)

type Registrar struct {
	sync.RWMutex
	client   *consul.Client
	cfg      *Config
	canceler map[string]context.CancelFunc
}

type Config struct {
	ConsulCfg   *consul.Config
	RegistryDir string
	Ttl         time.Duration
}

func NewRegistrar(cfg *Config) (registry.Registrar, error) {
	if cfg.ConsulCfg.Transport == nil {
		cfg.ConsulCfg.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		}
	}
	if cfg.ConsulCfg.HttpClient == nil {
		httpClient, err := consul.NewHttpClient(cfg.ConsulCfg.Transport, cfg.ConsulCfg.TLSConfig)
		if err != nil {
			return nil, err
		}

		httpClient.Timeout = 3 * time.Second
		cfg.ConsulCfg.HttpClient = httpClient
	}
	c, err := consul.NewClient(cfg.ConsulCfg)
	if err != nil {
		return nil, err
	}
	return &Registrar{
		canceler: make(map[string]context.CancelFunc),
		client:   c,
		cfg:      cfg,
	}, nil
}

func (r *Registrar) Register(service *registry.ServiceInfo) error {
	// register service
	metadata, err := json.Marshal(service.Metadata)
	if err != nil {
		return err
	}

	serviceName := r.cfg.RegistryDir
	if len(serviceName) > 0 {
		serviceName += "-"
	}
	serviceName += service.Name
	if len(service.Version) > 0 {
		serviceName += "-" + service.Version
	}
	tags := make([]string, 0)
	tags = append(tags, string(metadata))
	register := func() error {
		regis := &consul.AgentServiceRegistration{
			ID:      service.InstanceId,
			Name:    serviceName,
			Address: service.Address,
			Tags:    tags,
			Check: &consul.AgentServiceCheck{
				TTL:                            fmt.Sprintf("%ds", r.cfg.Ttl/time.Second),
				Status:                         consul.HealthPassing,
				DeregisterCriticalServiceAfter: "1m",
			}}
		err = r.client.Agent().ServiceRegister(regis)
		if err != nil {
			return fmt.Errorf("register service to consul error: %s\n", err.Error())
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.Lock()
	r.canceler[service.InstanceId] = cancel
	r.Unlock()
	keepAliveTicker := time.NewTicker(r.cfg.Ttl / 5)
	registerTicker := time.NewTicker(time.Minute)
	err = register()
	for {
		if err != nil {
			log.Errorf("consul registry error: %v", err)
			err = nil
		}

		select {
		case <-ctx.Done():
			keepAliveTicker.Stop()
			registerTicker.Stop()
			_ = r.client.Agent().ServiceDeregister(service.InstanceId)
			return nil
		case <-keepAliveTicker.C:
			err = r.client.Agent().PassTTL("service:"+service.InstanceId, "")
		case <-registerTicker.C:
			err = register()
		}
	}
}

func (r *Registrar) Unregister(service *registry.ServiceInfo) error {
	r.RLock()
	cancel, ok := r.canceler[service.InstanceId]
	r.RUnlock()

	if ok {
		cancel()
	}
	return nil
}

func (r *Registrar) Close() {
	r.RLock()
	for _, cancel := range r.canceler {
		cancel()
	}
	r.RUnlock()
}
