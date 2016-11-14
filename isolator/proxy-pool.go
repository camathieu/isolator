package isolator

import (
	"time"
	"errors"
)

type ProxyPool struct {
	pool chan(*Proxy)
}

func NewProxyPool() (pp *ProxyPool) {
	pp = new(ProxyPool)
	pp.pool = make(chan *Proxy,1000)
	return
}

func (pp *ProxyPool) Offer(c *Proxy) {
	pp.pool <- c
}

func (pp *ProxyPool) Take() (*Proxy){
	return <- pp.pool
}

func (pp *ProxyPool) TakeWithTimeout(d time.Duration) (p *Proxy, err error){
	select {
	case p = <- pp.pool:
	case <- time.After(d):
		err = errors.New("timeout")
	}
	return
}