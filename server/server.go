package server

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"sync"
	"time"
	"fmt"
	"math/rand"
)

// IzolatorServer is a Reverse HTTP Proxy over WebSocket
type IzolatorServer struct {
	upgrader websocket.Upgrader

	// Remote IzolatorProxies
	proxies map[string]*ConnectionPool
	proxiesNames []string

	lock sync.RWMutex
}

// New Server return a new IzolatorServer instance
func NewIzolatorServer() (is *IzolatorServer) {
	rand.Seed(time.Now().Unix())

	is = new(IzolatorServer)
	is.proxies = make(map[string]*ConnectionPool)
	is.upgrader = websocket.Upgrader{}
	return
}

// Start IzolatorServer HTTP server
func (is *IzolatorServer) Start() {
	r := http.NewServeMux()
	r.HandleFunc("/proxy", is.proxy)
	r.HandleFunc("/register", is.register)
	r.HandleFunc("/stats", is.stats)
	r.HandleFunc("/test", is.stats)
	r.HandleFunc("/close", is.close)

	s := &http.Server{Addr: "127.0.0.1:8080", Handler: r}
	log.Fatal(s.ListenAndServeTLS("ssl/cert.pem","ssl/key.pem"))
}

// This is the way for clients to execute HTTP requests through a IzolatorProxy
func (is *IzolatorServer) proxy(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v %v",r.Method,r.Header)

	if len(is.proxiesNames) == 0 {
		http.Error(w,fmt.Sprintf("No proxy available"),526)
	}

	index := rand.Intn(len(is.proxiesNames))
	start := time.Now()

	for {
		if time.Now().Sub(start).Seconds() > 1 {
			break
		}

		// Get a pool
		is.lock.RLock()
		index = (index + 1) % len(is.proxiesNames)
		proxy := is.proxies[is.proxiesNames[index]]
		log.Printf("%v / %v",index,len(is.proxiesNames))
		is.lock.RUnlock()

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

// This is the way for IzolatorProxies to offer websocket connections
func (is *IzolatorServer) register(w http.ResponseWriter, r *http.Request) {
	ws, err := is.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade proxy error : %v", err)
		http.Error(w,fmt.Sprintf("upgrade proxy error : %v", err), 400)
		return
	}

	log.Println("ws conn type : %s",ws.Type())

	// The first message should contains the remote IzolatorProxy name
	_, greeting, err := ws.ReadMessage()
	if err != nil {
		log.Println("greeting error")
		ws.Close()
		return
	}

	name := string(greeting)

	is.lock.Lock()
	defer is.lock.Unlock()

	// Get that IzolatorProxy websocket pool
	proxy, ok := is.proxies[name]
	if (!ok){
		proxy = NewProxyPool(name)
		is.proxies[name] = proxy
		is.proxiesNames = append(is.proxiesNames, name)
	}

	// Add the ws to the pool
	proxy.Register(ws)
}

func (is *IzolatorServer) close(w http.ResponseWriter, r *http.Request) {
	log.Printf("close")

	if len(is.proxiesNames) == 0 {
		http.Error(w,fmt.Sprintf("No proxy available"),526)
	}

	proxy := is.proxies[is.proxiesNames[0]]
	if pc := proxy.Take(); pc != nil {
		pc.Close()
		return
	} else {
		http.Error(w,fmt.Sprintf("No proxy connection available"),526)
	}
}

func (is *IzolatorServer) stats(w http.ResponseWriter, r *http.Request) {
	//TODO
}

func (is *IzolatorServer) test(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}