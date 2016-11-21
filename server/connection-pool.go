package server

import (
	"log"

	"github.com/gorilla/websocket"
	"sync"
	"errors"
)

type ConnectionPool struct {
	name string
	//pool chan (*ProxyConnection)

	size int
	pool []*ProxyConnection
	lock sync.RWMutex

	done chan(struct{})
}

func NewProxyPool(name string, size int) (cp *ConnectionPool) {
	cp = new(ConnectionPool)
	cp.name = name
	cp.size = size
	//cp.pool = make(chan *ProxyConnection, 1000)
	cp.pool = make([]*ProxyConnection,0)
	return
}

func (cp *ConnectionPool) Register(ws *websocket.Conn) (err error) {
	log.Printf("Registering new connection from %s", cp.name)
	pc := NewProxyConnection(cp, ws)
	err = cp.Offer(pc)
	return
}

var ConnectionPoolFull error = errors.New("ConnectionPool is full")
func (cp *ConnectionPool) Offer(pc *ProxyConnection) (err error){
	//cp.pool <- pc
	cp.lock.Lock()
	defer cp.lock.Unlock()

	size := cp.clean()
	if (size < cp.size){
		cp.pool = append(cp.pool,pc)
	} else {
		err = ConnectionPoolFull
		log.Printf("Discarding connection from %s because pool is full", cp.name)
		pc.Close()
	}

	return
}

func (cp *ConnectionPool) Take() (pc *ProxyConnection) {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	size := cp.clean()
	if size == 0 {
		return
	}

	//select {
	//case pc := <-cp.pool:
	//	return pc
	//default:
	//	return nil
	//}

	// Shift
	pc, cp.pool = cp.pool[0], cp.pool[1:]
	return
}

//func (cp *ConnectionPool) TakeWithTimeout(d time.Duration) (p *ProxyConnection, err error) {
//	select {
//	case p = <-cp.pool:
//	case <-time.After(d):
//		err = errors.New("timeout")
//	}
//	return
//}

// This MUST be surrounded by cp.lock
func (cp *ConnectionPool) clean() int {
	if len(cp.pool) == 0 {
		return 0
	}

	// TODO use the no allocation slice trick
	var pool []*ProxyConnection // == nil
	for _, pc := range cp.pool {
		if pc.status == IDLE {
			pool = append(pool, pc)
		}
	}
	cp.pool = pool

	return len(cp.pool)
}

func (cp *ConnectionPool) isEmpty() bool {
	cp.lock.Lock()
	defer cp.lock.Unlock()
	return cp.clean() == 0
}