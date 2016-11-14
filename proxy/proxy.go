package proxy

import (
	"net/http"
)

type Proxy struct {
	config *ProxyConfig
	client *http.Client
	pool map[string]*ConnectionPool
}

func NewProxy(config *ProxyConfig) (p *Proxy){
	p = new(Proxy)
	p.config = config
	p.client = new(http.Client)
	p.pool = make(map[string]*ConnectionPool)
	return
}

func (p *Proxy) Start() {
	for _, target := range p.config.Targets {
		pool := NewConnectionPool(p,target)
		p.pool[target] = pool
		go pool.Start()
	}
}