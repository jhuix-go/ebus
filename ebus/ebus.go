/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package ebus

import (
	`sync`

	`github.com/jhuix-go/ebus/pkg/app`
	`github.com/jhuix-go/ebus/pkg/discovery`
	`github.com/jhuix-go/ebus/pkg/discovery/registry`
	`github.com/jhuix-go/ebus/pkg/log`
	`github.com/jhuix-go/ebus/server`
)

type Service struct {
	cfg       *ServiceConfig
	svr       *server.Server
	registrar registry.Registrar
	svcInfo   *registry.ServiceInfo
	wg        sync.WaitGroup
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ParseCommandArgs() {
	// log.SetLogger(NewLogger())
}

func (s *Service) Initialize(cfg app.Config) error {
	err := s.InitializeConfig(cfg)
	if err != nil {
		return err
	}

	if s.cfg.Discovery != nil && len(s.cfg.Discovery.Endpoints) > 0 {
		registrar, err := discovery.NewConsulRegistry(s.cfg.Discovery)
		if err != nil {
			log.Errorf("new consul registrar failed: %v", err)
			return err
		}

		s.registrar = registrar
		s.svcInfo = &registry.ServiceInfo{
			InstanceId: s.cfg.Discovery.NodeID,
			Name:       s.cfg.Discovery.ServiceName,
			Version:    s.cfg.Discovery.ServiceVersion,
			Address:    s.cfg.Discovery.Address,
		}
	}

	s.svr = server.NewServer(s.cfg.Service)
	return nil
}

func (s *Service) RunLoop() (err error) {
	if s.svr == nil {
		log.Errorf("server run failed: server not created.")
		return nil
	}

	if err = s.svr.Listen(s.cfg.Service.Address); err != nil {
		return
	}

	if s.registrar != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			log.Infof("register is starting...")
			_ = s.registrar.Register(s.svcInfo)
			log.Infof("register is exited.")
		}()
	}

	log.Infof("service is starting...")
	s.svr.Serve()
	// s.wg.Add(1)
	// go func() {
	// 	defer s.wg.Done()
	//
	// 	log.Infof("service is starting...")
	// 	s.svr.Serve()
	// 	log.Infof("service is exited.")
	// }()
	return
}

func (s *Service) Destroy() {
	log.Infof("service is stopping...")
	if s.registrar != nil {
		// _ = s.registrar.Unregister(s.svcInfo)
		s.registrar.Close()
	}
	if s.svr != nil {
		s.svr.Stop()
	}
	s.wg.Wait()
	log.Infof("service is stopped.")
}
