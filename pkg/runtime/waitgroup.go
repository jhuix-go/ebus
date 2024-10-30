package runtime

import "sync"

type WaitGroup struct {
	wg sync.WaitGroup
}

func (g *WaitGroup) Wait() {
	g.wg.Wait()
}

func (g *WaitGroup) Start(f func()) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		f()
	}()
}
