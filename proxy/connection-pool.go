package proxy

import (
	"sync"
	"log"
	"time"
	"fmt"
)

type ConnectionPool struct {
	proxy *IzolatorProxy
	target string

	connections []*ProxyConnection
	lock sync.Mutex

	done chan(struct{})
}

func NewConnectionPool(proxy *IzolatorProxy, target string) (cp *ConnectionPool){
	cp = new(ConnectionPool)
	cp.proxy = proxy
	cp.target = target
	cp.connections = make([]*ProxyConnection,0)
	cp.done = make(chan(struct{}))
	return
}

func (cp *ConnectionPool) Start() {
	for {
		err := cp.Connect()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	log.Printf("Connected to %s\n", cp.target)

	go func() {
		ticker := time.Tick(time.Second)
		for {
			select {
			case <- cp.done:
				break
			case <- ticker:
				cp.GarbageCollector()
			}
		}
	}()
}

func (cp *ConnectionPool) Connect() (err error){
	ps := cp.Size()

	if ps.idle >= cp.proxy.config.PoolIdleSize || ps.total >= cp.proxy.config.PoolMaxSize {
		return
	}

	if ps.connecting >= cp.proxy.config.PoolIdleSize - ps.idle {
		return
	}

	//log.Printf("%s pool size : %v",cp.target, ps)

	conn := NewProxyConnection(cp)
	cp.Add(conn)

	err = conn.Connect()
	if err != nil {
		log.Printf("Unable to connect to %s : %s",cp.target,err)
		cp.Remove(conn)
	}

	return
}

func (cp *ConnectionPool) GarbageCollector() {
	ps := cp.Size()
	log.Printf("%s pool size : %v",cp.target, ps)

	if ps.total == 0 {
		err := cp.Connect()
		if err != nil {
			return
		}
	}

	if ps.idle < cp.proxy.config.PoolIdleSize {
		for i := ps.idle ; i < cp.proxy.config.PoolIdleSize ; i++ {
			go cp.Connect()
		}
	}

	if ps.idle > cp.proxy.config.PoolIdleSize {
		// Remove old connection ( ou new ? )
	}
}

type PoolSize struct {
	connecting int
	idle int
	running int
	closed int
	total int
}

func (ps *PoolSize) String() string {
	return fmt.Sprintf("Connecting %d, idle %d, running %d, closed %d, total %d",ps.connecting,ps.idle,ps.running,ps.closed,ps.total)
}

func (cp *ConnectionPool) Size() (ps *PoolSize){
	cp.lock.Lock()
	defer cp.lock.Unlock()

	ps = new(PoolSize)
	ps.total = len(cp.connections)
	for _, connection :=  range cp.connections {
		switch connection.status {
		case CONNECTING:
			ps.connecting++
		case IDLE:
			ps.idle++
		case RUNNING:
			ps.running++
		case CLOSED:
			ps.closed++
		}
	}

	return
}

func (cp *ConnectionPool) Add(conn *ProxyConnection){
	cp.lock.Lock()
	defer cp.lock.Unlock()

	cp.connections = append(cp.connections,conn)

}

func (cp *ConnectionPool) Remove(conn *ProxyConnection){
	cp.lock.Lock()
	defer cp.lock.Unlock()

	// This trick uses the fact that a slice shares the same backing array and capacity as the original,
	// so the storage is reused for the filtered slice. Of course, the original contents are modified.
	filtered := cp.connections[:0]
	for _, c := range cp.connections {
		if conn != c {
			filtered = append(filtered, c)
		}
	}
	cp.connections = filtered
}

func (cp *ConnectionPool) Shutdown(){
	close(cp.done)
	for _, conn := range cp.connections {
		conn.Close()
	}
}