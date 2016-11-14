package isolator

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"sync"
	"time"
	"fmt"
	"math/rand"
)

type Isolator struct {
	upgrader websocket.Upgrader

	pools map[string]*ProxyPool
	poolsNames []string

	lock sync.RWMutex
}

func NewIsolator() (i *Isolator) {
	rand.Seed(time.Now().Unix())

	i = new(Isolator)
	i.pools = make(map[string]*ProxyPool)
	i.upgrader = websocket.Upgrader{}
	return
}

func (i *Isolator) Start() {
	r := http.NewServeMux()
	r.HandleFunc("/proxy", i.proxy)
	r.HandleFunc("/register", i.register)
	r.HandleFunc("/stats", i.stats)
	r.HandleFunc("/test", i.stats)

	s := &http.Server{Addr: "127.0.0.1:8080", Handler: r}
	log.Fatal(s.ListenAndServe())
}

// This is the way for client to execute HTTP requests through a proxy
func (i *Isolator) proxy(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v %v",r.Method,r.Header)

	if len(i.poolsNames) == 0 {
		http.Error(w,fmt.Sprintf("No proxy available"),526)
	}

	index := rand.Intn(len(i.poolsNames))
	start := time.Now()

	for {
		if time.Now().Sub(start).Seconds() > 1 {
			break
		}

		// Get a pool
		i.lock.RLock()
		index = (index + 1) % len(i.poolsNames)
		proxy := i.pools[i.poolsNames[index]]
		log.Printf("%v / %v",index,len(i.poolsNames))
		i.lock.RUnlock()

		if pc := proxy.Take(); pc != nil {
			if pc.status != IDLE && time.Now().Sub(start).Seconds() < 1 {
				continue
			}

			err := pc.proxyRequest(w, r)
			if err == nil {
				// Everything went well we can reuse the connection
				//proxy.Offer(pc)
				pc.Close()
			} else {
				// An error occurred throw the connection away
				log.Println(err)
				pc.Close()

				// Try to return an error to the client
				// This might fail if response headers have already been sent
				http.Error(w, err.Error(), 526)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	http.Error(w,fmt.Sprintf("Unable to get a proxy connection"),526)
}

// This is the way for proxy to offer websocket connections
func (i *Isolator) register(w http.ResponseWriter, r *http.Request) {
	ws, err := i.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade proxy error : %v", err)
		http.Error(w,fmt.Sprintf("upgrade proxy error : %v", err), 400)
		return
	}

	//
	_, greeting, err := ws.ReadMessage()
	if err != nil {
		log.Println("greeting error")
		ws.Close()
		return
	}

	hostname := string(greeting)

	i.lock.Lock()
	defer i.lock.Unlock()

	// Get that proxy connection pool
	pool, ok := i.pools[hostname]
	if (!ok){
		pool = NewProxyPool(hostname)
		i.pools[hostname] = pool
		i.poolsNames = append(i.poolsNames,hostname)
	}

	// Add the ws to the pool
	pool.Register(ws)
}

func (i *Isolator) stats(w http.ResponseWriter, r *http.Request) {
	//TODO
}

func (i *Isolator) test(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}