package singleflight

import "sync"

//doing request
type call struct {
	//only once get in,avoid call repeatly
	wg  sync.WaitGroup
	val interface{}
	err error
}

//diff key call
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	//call is doing
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, nil
	}
	//first to do the call
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()
	//get val
	c.val, c.err = fn()
	c.wg.Done()
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}
