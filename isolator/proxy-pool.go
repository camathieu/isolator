package isolator

import (
	"time"
	"errors"
	"github.com/gorilla/websocket"
	"log"
)

type ProxyPool struct {
	name string
	pool chan(*ProxyConnection)
}

func NewProxyPool(name string) (pp *ProxyPool) {
	pp = new(ProxyPool)
	pp.name = name
	pp.pool = make(chan *ProxyConnection,1000)

	return
}

func (pp *ProxyPool) Register(ws *websocket.Conn) {
	log.Printf("Registering new connection from %s",pp.name)
	pc := NewProxyConnection(pp,ws)
	pp.Offer(pc)
}

func (pp *ProxyPool) Offer(pc *ProxyConnection) {
	pp.pool <- pc
}

func (pp *ProxyPool) Take() (*ProxyConnection){
	select {
	case pc := <-pp.pool:
		return pc
	default:
		return nil
	}
}

func (pp *ProxyPool) TakeWithTimeout(d time.Duration) (p *ProxyConnection, err error){
	select {
	case p = <- pp.pool:
	case <- time.After(d):
		err = errors.New("timeout")
	}
	return
}