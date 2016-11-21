package server

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"strings"
	"strconv"

	"github.com/gorilla/websocket"
	"net/url"
)

// IzolatorServer is a Reverse HTTP Proxy over WebSocket
type IzolatorServer struct {
	upgrader   websocket.Upgrader

	// Remote IzolatorProxies
	pools      []*ConnectionPool
	lock       sync.RWMutex
}

// New Server return a new IzolatorServer instance
func NewIzolatorServer() (is *IzolatorServer) {
	rand.Seed(time.Now().Unix())

	is = new(IzolatorServer)
	is.upgrader = websocket.Upgrader{}

	is.pools = make([]*ConnectionPool,0)

	return
}

// Start IzolatorServer HTTP server
func (is *IzolatorServer) Start() {
	go func(){
		for {
			<- time.After(30 * time.Second)
			is.clean()
		}
	}()

	r := http.NewServeMux()
	r.HandleFunc("/proxy", is.proxy)
	r.HandleFunc("/register", is.register)
	r.HandleFunc("/stats", is.stats)
	r.HandleFunc("/test", is.stats)

	s := &http.Server{Addr: "127.0.0.1:8080", Handler: r}
	log.Fatal(s.ListenAndServe())
}

// Remove empty conenction pools
func (is *IzolatorServer) clean() int {
	is.lock.Lock()
	defer is.lock.Unlock()

	if len(is.pools) == 0 {
		return 0
	}

	var pools []*ConnectionPool // == nil
	for _, cp := range is.pools {
		if cp.clean() > 0 {
			pools = append(pools, cp)
		} else {
			log.Printf("Removing empty connection pool : %s", cp.name)
		}
	}
	is.pools = pools

	return len(is.pools)
}

// This is the way for clients to execute HTTP requests through an IzolatorProxy
func (is *IzolatorServer) proxy(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v %v", r.Method, r.Header)

	if len(is.pools) == 0 {
		http.Error(w, "No proxy available", 526)
		return
	}

	// Parse destination URL
	dstURL := r.Header.Get("X-PROXY-DESTINATION")
	if dstURL == "" {
		http.Error(w, "Missing X-PROXY-DESTINATION header", 526)
		return
	}
	URL, err := url.Parse(dstURL)
	if err != nil {
		http.Error(w, "Unable to parse X-PROXY-DESTINATION header", 526)
		return
	}
	r.URL = URL

	start := time.Now()

	// Randomly select a pool ( other load-balancing strategies could be implemented )
	// and try to acquire a connection. This should be refactored to use channels
	index := rand.Intn(len(is.pools))
	for {
		if time.Now().Sub(start).Seconds() > 1 {
			break
		}

		// Get a pool
		is.lock.RLock()
		index = (index + 1) % len(is.pools)
		cp := is.pools[index]
		log.Printf("%v / %v", index, len(is.pools))
		is.lock.RUnlock()

		// Get a connection
		if pc := cp.Take(); pc != nil {
			if pc.status != IDLE && time.Now().Sub(start).Seconds() < 1 {
				continue
			}

			// Send the request to the proxy
			err := pc.proxyRequest(w, r)
			if err == nil {
				// Everything went well we can reuse the connection
				cp.Offer(pc)
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

	http.Error(w, fmt.Sprintf("Unable to get a proxy connection"), 526)
}

// This is the way for IzolatorProxies to offer websocket connections
func (is *IzolatorServer) register(w http.ResponseWriter, r *http.Request) {
	ws, err := is.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade proxy error : %v", err)
		http.Error(w, fmt.Sprintf("upgrade proxy error : %v", err), 400)
		return
	}

	// The first message should contains the remote IzolatorProxy name and size
	_, greeting, err := ws.ReadMessage()
	if err != nil {
		log.Printf("Unable to read greeting message : %s",err)
		ws.Close()
		return
	}

	// Parse the greeting message
	split := strings.Split(string(greeting), "_")
	name := split[0]
	size, err := strconv.Atoi(split[1])
	if err != nil {
		log.Printf("Invalid greeting message : %s", err)
	}

	is.lock.Lock()
	defer is.lock.Unlock()

	// Get that IzolatorProxy websocket pool
	var pool *ConnectionPool
	for _, cp := range is.pools {
		if cp.name == name {
			pool = cp
			break
		}
	}
	if pool == nil {
		pool = NewProxyPool(name,size)
		is.pools = append(is.pools,pool)
	}

	// Add the ws to the pool
	pool.Register(ws)
}

func (is *IzolatorServer) stats(w http.ResponseWriter, r *http.Request) {
	//TODO
}

func (is *IzolatorServer) test(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (is *IzolatorServer) Shutdown(){
	// TODO
}