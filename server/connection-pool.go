package server

import (
	"time"
	"errors"
	"github.com/gorilla/websocket"
	"log"
)

type ConnectionPool struct {
	name string
	pool chan(*ProxyConnection)
}

func NewProxyPool(name string) (cp *ConnectionPool) {
	cp = new(ConnectionPool)
	cp.name = name
	cp.pool = make(chan *ProxyConnection,1000)

	return
}

func (cp *ConnectionPool) Register(ws *websocket.Conn) {
	log.Printf("Registering new connection from %s", cp.name)
	pc := NewProxyConnection(cp,ws)
	cp.Offer(pc)
}

func (cp *ConnectionPool) Offer(pc *ProxyConnection) {
	cp.pool <- pc
}

func (cp *ConnectionPool) Take() (*ProxyConnection){
	select {
	case pc := <-cp.pool:
		return pc
	default:
		return nil
	}
}

func (cp *ConnectionPool) TakeWithTimeout(d time.Duration) (p *ProxyConnection, err error){
	select {
	case p = <- cp.pool:
	case <- time.After(d):
		err = errors.New("timeout")
	}
	return
}